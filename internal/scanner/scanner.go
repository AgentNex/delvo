package scanner

import (
	"os"
	"path/filepath"
)

// ScanSingleDir scans only one directory without recursion.
func ScanSingleDir(dirPath string) ([]Entry, error) {
	cleanPath := filepath.Clean(dirPath)
	dirEntries, err := os.ReadDir(cleanPath)
	if err != nil {
		return nil, err
	}

	results := make([]Entry, 0, len(dirEntries))
	for _, d := range dirEntries {
		fp := filepath.Join(cleanPath, d.Name())
		fi, err := os.Lstat(fp)
		if err != nil {
			continue
		}
		results = append(results, ExtractEntry(fp, fi))
	}

	return results, nil
}
