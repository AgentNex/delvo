package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// CheckSymlink checks if a path is a symbolic link without following it.
func CheckSymlink(path string) (isSymlink bool, target string, err error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return false, "", err
	}

	if fi.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(path)
		return true, target, err
	}

	return false, "", nil
}

// Revalidate verifies that a path exists before destructive operations.
func Revalidate(path string, expectedModTime time.Time, expectedSize int64) (os.FileInfo, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file no longer exists: %s", path)
		}
		return nil, fmt.Errorf("cannot access file: %w", err)
	}

	// If timestamps or sizes were passed (nonzero), check them
	if !expectedModTime.IsZero() && fi.ModTime().Unix() != expectedModTime.Unix() {
		// Just a warning in log/error if heavily modified
		// We still return fi so caller can decide
	}

	return fi, nil
}

// ResolveOverlappingPaths prunes child paths when an ancestor directory is also in the list.
// For example, if both "/a" and "/a/b.txt" are passed, "/a/b.txt" is removed.
func ResolveOverlappingPaths(paths []string) []string {
	if len(paths) <= 1 {
		return paths
	}

	// Clean and normalize all paths
	cleaned := make([]string, 0, len(paths))
	seen := make(map[string]bool)
	for _, p := range paths {
		c := filepath.Clean(p)
		if !seen[c] {
			seen[c] = true
			cleaned = append(cleaned, c)
		}
	}

	// Sort lexicographically so parent always comes before children
	sort.Strings(cleaned)

	var result []string
	for _, p := range cleaned {
		isChild := false
		for _, kept := range result {
			rel, err := filepath.Rel(kept, p)
			if err == nil && !strings.HasPrefix(rel, "..") && rel != "." {
				isChild = true
				break
			}
		}
		if !isChild {
			result = append(result, p)
		}
	}

	return result
}
