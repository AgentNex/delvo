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

// dirWorkQueue coordinates directory traversal without channel capacity deadlocks.
type dirWorkQueue struct {
	mu       sync.Mutex
	cond     *sync.Cond
	dirs     []string
	inFlight int
	closed   bool
}

func newDirWorkQueue() *dirWorkQueue {
	q := &dirWorkQueue{
		dirs: make([]string, 0, 256),
	}
	q.cond = sync.NewCond(&q.mu)
	return q
}

func (q *dirWorkQueue) push(dirs ...string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return
	}
	q.dirs = append(q.dirs, dirs...)
	q.cond.Broadcast()
}

func (q *dirWorkQueue) pop(ctx context.Context) (string, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for len(q.dirs) == 0 && q.inFlight > 0 && !q.closed {
		if ctx.Err() != nil {
			q.closed = true
			q.cond.Broadcast()
			return "", false
		}
		q.cond.Wait()
	}

	if q.closed || len(q.dirs) == 0 {
		return "", false
	}

	last := len(q.dirs) - 1
	dir := q.dirs[last]
	q.dirs = q.dirs[:last]
	q.inFlight++
	return dir, true
}

func (q *dirWorkQueue) done() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.inFlight--
	if q.inFlight == 0 && len(q.dirs) == 0 {
		q.cond.Broadcast()
	}
}

func (q *dirWorkQueue) close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.closed = true
	q.cond.Broadcast()
}

// Scan runs the parallel scan across all configured roots.
func (s *Scanner) Scan(
	ctx context.Context,
	progressFn func(ScanProgress),
	batchFn func([]Entry) error,
) (*ScanResult, error) {
	scanCtx, cancelScan := context.WithCancel(ctx)
	defer cancelScan()

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

	// Dynamic work queue
	queue := newDirWorkQueue()

	// Cancellation listener for queue
	go func() {
		<-scanCtx.Done()
		queue.close()
	}()

	// Single writer channel for serializing SQLite transactions
	batchChan := make(chan []Entry, s.opts.Workers*2)
	var writerWg sync.WaitGroup
	if batchFn != nil {
		writerWg.Add(1)
		go func() {
			defer writerWg.Done()
			for batch := range batchChan {
				if err := batchFn(batch); err != nil {
					errMu.Lock()
					scanErrors = append(scanErrors, fmt.Sprintf("batch write: %v", err))
					errMu.Unlock()
				}
			}
		}()
	}

	var lastProgressTime time.Time
	var progressMu sync.Mutex
	sendProgress := func(curPath string, isComplete bool) {
		if progressFn == nil {
			return
		}
		progressMu.Lock()
		now := time.Now()
		if !isComplete && now.Sub(lastProgressTime) < 50*time.Millisecond {
			progressMu.Unlock()
			return
		}
		lastProgressTime = now
		progressMu.Unlock()

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

	// Bounded worker pool
	var activeWorkers sync.WaitGroup
	worker := func() {
		defer activeWorkers.Done()

		localBatch := make([]Entry, 0, s.opts.BatchSize)

		flushBatch := func() error {
			if len(localBatch) == 0 {
				return nil
			}

			batchCopy := make([]Entry, len(localBatch))
			copy(batchCopy, localBatch)

			allEntriesMu.Lock()
			allEntries = append(allEntries, batchCopy...)
			allEntriesMu.Unlock()

			if batchFn != nil {
				select {
				case batchChan <- batchCopy:
				case <-scanCtx.Done():
					return scanCtx.Err()
				}
			}
			localBatch = localBatch[:0]
			return nil
		}

		for {
			if scanCtx.Err() != nil {
				_ = flushBatch()
				return
			}

			dir, ok := queue.pop(scanCtx)
			if !ok {
				_ = flushBatch()
				return
			}

			sendProgress(dir, false)
			dirsCount.Add(1)

			entries, err := os.ReadDir(dir)
			if err != nil {
				errMu.Lock()
				scanErrors = append(scanErrors, fmt.Sprintf("skip %s: %v", dir, err))
				errMu.Unlock()
				queue.done()
				continue
			}

			var subdirs []string
			for _, d := range entries {
				if scanCtx.Err() != nil {
					_ = flushBatch()
					queue.done()
					return
				}

				fullPath := filepath.Join(dir, d.Name())

				// Use Lstat so we don't follow symlinks inadvertently
				fi, err := os.Lstat(fullPath)
				if err != nil {
					continue
				}

				entry := ExtractEntry(fullPath, fi)
				localBatch = append(localBatch, entry)

				if !entry.IsDir {
					filesCount.Add(1)
					bytesCount.Add(entry.Size)
				}

				if len(localBatch) >= s.opts.BatchSize {
					_ = flushBatch()
				}

				// Recurse into subdirectories if permitted
				if entry.IsDir && (!entry.IsSymlink || s.opts.FollowSymlinks) {
					if filesystem.IsInsideRoots(fullPath, s.opts.Roots) {
						subdirs = append(subdirs, fullPath)
					}
				}
			}

			if len(subdirs) > 0 {
				queue.push(subdirs...)
			}

			_ = flushBatch()
			queue.done()
		}
	}

	// Enqueue initial roots first
	var initialRoots []string
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
				select {
				case batchChan <- []Entry{rootEntry}:
				case <-scanCtx.Done():
				}
			}
		}

		initialRoots = append(initialRoots, absRoot)
	}

	if len(initialRoots) > 0 {
		queue.push(initialRoots...)
	}

	// Start worker pool after initial roots are enqueued
	for i := 0; i < s.opts.Workers; i++ {
		activeWorkers.Add(1)
		go worker()
	}

	activeWorkers.Wait()

	// Shut down writer channel
	close(batchChan)
	writerWg.Wait()

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

	sendProgress("Scan complete", true)

	return &ScanResult{
		Entries:  allEntries,
		DirSizes: dirSizes,
		Progress: finalProgress,
	}, ctx.Err()
}
