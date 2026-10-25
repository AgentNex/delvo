package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidateRoot checks if a root path exists, is a directory, and is accessible.
func ValidateRoot(path string) (string, error) {
	abs, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("stat root: %w", err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", abs)
	}

	// Test readability by opening the directory
	f, err := os.Open(abs)
	if err != nil {
		return "", fmt.Errorf("directory not readable: %w", err)
	}
	_ = f.Close()

	return abs, nil
}

// IsInsideRoots verifies that targetPath is within at least one of the configured roots.
func IsInsideRoots(targetPath string, roots []string) bool {
	absTarget, err := filepath.Abs(filepath.Clean(targetPath))
	if err != nil {
		return false
	}

	for _, root := range roots {
		absRoot, err := filepath.Abs(filepath.Clean(root))
		if err != nil {
			continue
		}

		if absTarget == absRoot {
			return true
		}

		rel, err := filepath.Rel(absRoot, absTarget)
		if err != nil {
			continue
		}

		// If rel does not begin with "..", target is inside root
		if !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." {
			return true
		}
	}

	return false
}
