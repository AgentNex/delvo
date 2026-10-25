package scanner

import (
	"fmt"
	"path/filepath"
	"strings"
)

type inodeKey struct {
	dev uint64
	ino uint64
}

// ComputeDirectorySizes computes bottom-up directory sizes for all directories in entries.
// It avoids double-counting hardlinks if dev/inode are present.
func ComputeDirectorySizes(entries []Entry) map[string]int64 {
	dirSizes := make(map[string]int64)
	seenInodes := make(map[inodeKey]bool)

	// Register all directories first
	for _, e := range entries {
		if e.IsDir {
			if _, ok := dirSizes[e.Path]; !ok {
				dirSizes[e.Path] = 0
			}
		}
	}

	// Accumulate file sizes up the tree
	for _, e := range entries {
		if e.IsDir {
			continue
		}

		// Avoid double counting hard-links
		if e.Dev != 0 && e.Ino != 0 {
			key := inodeKey{dev: e.Dev, ino: e.Ino}
			if seenInodes[key] {
				continue
			}
			seenInodes[key] = true
		}

		// Add file size to direct parent and all its ancestors that are in dirSizes
		cur := e.ParentPath
		for cur != "" && cur != "." {
			dirSizes[cur] += e.Size
			parent := filepath.Dir(cur)
			if parent == cur {
				break
			}
			cur = parent
		}
	}

	return dirSizes
}

// FormatSize returns a human-readable representation of bytes.
func FormatSize(bytes int64) string {
	const (
		_          = iota
		KB float64 = 1 << (10 * iota)
		MB
		GB
		TB
	)

	b := float64(bytes)
	switch {
	case b >= TB:
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", b/TB), "0"), ".") + " TB"
	case b >= GB:
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", b/GB), "0"), ".") + " GB"
	case b >= MB:
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.1f", b/MB), "0"), ".") + " MB"
	case b >= KB:
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.1f", b/KB), "0"), ".") + " KB"
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
