package scanner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/AgentNex/delvo/internal/filesystem"
)

// ScanOptions configures the behavior of the Scanner.
type ScanOptions struct {
	Roots          []string
	Workers        int
	BatchSize      int
	FollowSymlinks bool
}

// ScanProgress communicates real-time progress of an active scan.
type ScanProgress struct {
	FilesScanned  int64         `json:"files_scanned"`
	DirsScanned   int64         `json:"dirs_scanned"`
	BytesScanned  int64         `json:"bytes_scanned"`
	CurrentPath   string        `json:"current_path"`
	Errors        []string      `json:"errors,omitempty"`
	IsComplete    bool          `json:"is_complete"`
	IsProvisional bool          `json:"is_provisional"`
	StartTime     time.Time     `json:"start_time"`
	Duration      time.Duration `json:"duration"`
}

// Scanner performs bounded parallel recursive filesystem scans.
type Scanner struct {
	opts ScanOptions
}

// New creates a new Scanner instance with the provided options.
func New(opts ScanOptions) *Scanner {
	if opts.Workers <= 0 {
		opts.Workers = 4
	}
	if opts.BatchSize <= 0 {
		opts.BatchSize = 250
	}
	return &Scanner{opts: opts}
}

// ScanResult holds the collected entries and progress information.
type ScanResult struct {
	Entries  []Entry
	DirSizes map[string]int64
	Progress ScanProgress
}

// Scan runs the parallel scan across all configured roots.
func (s *Scanner) Scan(
	ctx context.Context,
	progressFn func(ScanProgress),
	batchFn func([]Entry) error,
) (*ScanResult, error) {
	startTime := time.Now()
	var (
		filesCount atomic.Int64
		dirsCount  atomic.Int64
		bytesCount atomic.Int64
	)

	var errMu sync.Mutex
	var scanErrors []string

	// All collected entries (for calculating directory totals at the end)
	var allEntriesMu sync.Mutex
	var allEntries []Entry

	// Work queue for directories
	dirQueue := make(chan string, 10000)

	// Bounded worker pool
	var activeWorkers sync.WaitGroup
	var inFlightDirs atomic.Int64

	// Helper to send progress
	sendProgress := func(curPath string, isComplete bool) {
		if progressFn != nil {
			errMu.Lock()
			errsCopy := make([]string, len(scanErrors))
			copy(errsCopy, scanErrors)
			errMu.Unlock()

			progressFn(ScanProgress{
				FilesScanned:  filesCount.Load(),
				DirsScanned:   dirsCount.Load(),
				BytesScanned:  bytesCount.Load(),
				CurrentPath:   curPath,
				Errors:        errsCopy,
				IsComplete:    isComplete,
				IsProvisional: !isComplete,
				StartTime:     startTime,
				Duration:      time.Since(startTime),
			})
		}
	}

	// Worker logic
	worker := func() {
		defer activeWorkers.Done()

		localBatch := make([]Entry, 0, s.opts.BatchSize)

		flushBatch := func() error {
			if len(localBatch) == 0 {
				return nil
			}

			allEntriesMu.Lock()
			allEntries = append(allEntries, localBatch...)
			allEntriesMu.Unlock()

			if batchFn != nil {
				if err := batchFn(localBatch); err != nil {
					return err
				}
			}
			localBatch = localBatch[:0]
			return nil
		}

		for {
			select {
			case <-ctx.Done():
				_ = flushBatch()
				return
			case dir, ok := <-dirQueue:
				if !ok {
					_ = flushBatch()
					return
				}

				sendProgress(dir, false)
				dirsCount.Add(1)

				// Read dir entries
				entries, err := os.ReadDir(dir)
				if err != nil {
					errMu.Lock()
					scanErrors = append(scanErrors, fmt.Sprintf("skip %s: %v", dir, err))
					errMu.Unlock()
					inFlightDirs.Add(-1)
					continue
				}

				for _, d := range entries {
					select {
					case <-ctx.Done():
						_ = flushBatch()
						inFlightDirs.Add(-1)
						return
					default:
					}

					fullPath := filepath.Join(dir, d.Name())

					// Use Lstat so we don't follow symlinks inadvertently
					fi, err := os.Lstat(fullPath)
					if err != nil {
						// File might have disappeared or permission denied
						continue
					}

					entry := ExtractEntry(fullPath, fi)
					localBatch = append(localBatch, entry)

					if !entry.IsDir {
						filesCount.Add(1)
						bytesCount.Add(entry.Size)
					}

					if len(localBatch) >= s.opts.BatchSize {
						if err := flushBatch(); err != nil {
							errMu.Lock()
							scanErrors = append(scanErrors, fmt.Sprintf("batch write: %v", err))
							errMu.Unlock()
						}
					}

					// Recurse into subdirectories if permitted
					if entry.IsDir && !entry.IsSymlink {
						// Don't recurse if outside configured roots
						if filesystem.IsInsideRoots(fullPath, s.opts.Roots) {
							inFlightDirs.Add(1)
							select {
							case dirQueue <- fullPath:
							case <-ctx.Done():
								_ = flushBatch()
								inFlightDirs.Add(-1)
								return
							}
						}
					}
				}

				if err := flushBatch(); err != nil {
					errMu.Lock()
					scanErrors = append(scanErrors, fmt.Sprintf("batch write: %v", err))
					errMu.Unlock()
				}

				inFlightDirs.Add(-1)
			}
		}
	}

	// Start worker pool
	for i := 0; i < s.opts.Workers; i++ {
		activeWorkers.Add(1)
		go worker()
	}

	// Enqueue initial roots
	for _, root := range s.opts.Roots {
		absRoot, err := filesystem.ValidateRoot(root)
		if err != nil {
			errMu.Lock()
			scanErrors = append(scanErrors, fmt.Sprintf("root %s: %v", root, err))
			errMu.Unlock()
			continue
		}

		fi, err := os.Lstat(absRoot)
		if err == nil {
			rootEntry := ExtractEntry(absRoot, fi)
			allEntriesMu.Lock()
			allEntries = append(allEntries, rootEntry)
			allEntriesMu.Unlock()

			if batchFn != nil {
				_ = batchFn([]Entry{rootEntry})
			}
		}

		inFlightDirs.Add(1)
		dirQueue <- absRoot
	}

	// Monitor completion: close dirQueue when no more in-flight directories remain
	doneChan := make(chan struct{})
	go func() {
		for {
			select {
			case <-ctx.Done():
				close(dirQueue)
				close(doneChan)
				return
			default:
				if inFlightDirs.Load() == 0 {
					close(dirQueue)
					close(doneChan)
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()

	<-doneChan
	activeWorkers.Wait()

	// Compute bottom-up directory totals
	dirSizes := ComputeDirectorySizes(allEntries)

	// Update directory sizes in allEntries
	for i := range allEntries {
		if allEntries[i].IsDir {
			if sz, ok := dirSizes[allEntries[i].Path]; ok {
				allEntries[i].Size = sz
			}
		}
	}

	finalProgress := ScanProgress{
		FilesScanned:  filesCount.Load(),
		DirsScanned:   dirsCount.Load(),
		BytesScanned:  bytesCount.Load(),
		CurrentPath:   "Scan complete",
		Errors:        scanErrors,
		IsComplete:    true,
		IsProvisional: false,
		StartTime:     startTime,
		Duration:      time.Since(startTime),
	}

	if progressFn != nil {
		progressFn(finalProgress)
	}

	return &ScanResult{
		Entries:  allEntries,
		DirSizes: dirSizes,
		Progress: finalProgress,
	}, ctx.Err()
}
