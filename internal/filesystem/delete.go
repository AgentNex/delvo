package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
)

var trashCounter atomic.Uint64

// DeleteOptions controls how deletions are executed.
type DeleteOptions struct {
	Permanent bool
	TrashDir  string
}

// DeleteResult records the outcome of deleting a single path.
type DeleteResult struct {
	Path       string `json:"path"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
	WasSymlink bool   `json:"was_symlink"`
	MovedTo    string `json:"moved_to,omitempty"`
}

// DeleteItems safely deletes or trashes items after revalidation and bounds checking.
func DeleteItems(paths []string, roots []string, opts DeleteOptions) []DeleteResult {
	resolved := ResolveOverlappingPaths(paths)
	results := make([]DeleteResult, 0, len(resolved))

	for _, p := range resolved {
		res := DeleteResult{Path: p}

		// 1. Boundary check: ensure target is within configured roots
		if !IsInsideRoots(p, roots) {
			res.Error = fmt.Sprintf("path is outside configured roots: %s", p)
			results = append(results, res)
			continue
		}

		// 2. Revalidate existence via Revalidate (do not follow symlinks)
		fi, err := Revalidate(p, time.Time{}, 0)
		if err != nil {
			res.Error = fmt.Sprintf("revalidate before delete: %v", err)
			results = append(results, res)
			continue
		}

		isSymlink := fi.Mode()&os.ModeSymlink != 0
		res.WasSymlink = isSymlink

		if !opts.Permanent {
			// Trash workflow
			trashDir := opts.TrashDir
			if trashDir == "" {
				trashDir = filepath.Join(os.TempDir(), "delvo_trash")
			}
			if err := os.MkdirAll(trashDir, 0700); err != nil {
				res.Error = fmt.Sprintf("create trash dir: %v", err)
				results = append(results, res)
				continue
			}

			// Store in a unique directory inside trashDir to never exceed filesystem filename limits
			subDirName := fmt.Sprintf("%d_%d", time.Now().UnixNano(), trashCounter.Add(1))
			itemTrashDir := filepath.Join(trashDir, subDirName)
			if err := os.MkdirAll(itemTrashDir, 0700); err != nil {
				res.Error = fmt.Sprintf("create item trash dir: %v", err)
				results = append(results, res)
				continue
			}

			destPath := filepath.Join(itemTrashDir, filepath.Base(p))

			if err := os.Rename(p, destPath); err != nil {
				// If rename failed (e.g. cross-device), try copy + remove
				if errCopy := copyAndRemove(p, destPath, fi); errCopy != nil {
					res.Error = fmt.Sprintf("move to trash: %v (copy fallback: %v)", err, errCopy)
					results = append(results, res)
					continue
				}
			}

			res.Success = true
			res.MovedTo = destPath
			results = append(results, res)
			continue
		}

		// Permanent delete workflow
		if isSymlink {
			// For symlinks, never delete the target. Remove the symlink itself.
			if err := os.Remove(p); err != nil {
				res.Error = fmt.Sprintf("remove symlink: %v", err)
			} else {
				res.Success = true
			}
		} else if fi.IsDir() {
			if err := os.RemoveAll(p); err != nil {
				res.Error = fmt.Sprintf("remove directory: %v", err)
			} else {
				res.Success = true
			}
		} else {
			if err := os.Remove(p); err != nil {
				res.Error = fmt.Sprintf("remove file: %v", err)
			} else {
				res.Success = true
			}
		}

		results = append(results, res)
	}

	return results
}
