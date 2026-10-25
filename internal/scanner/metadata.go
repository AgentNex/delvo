package scanner

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Entry represents metadata for a file or directory.
type Entry struct {
	Path          string    `json:"path"`
	ParentPath    string    `json:"parent_path"`
	Name          string    `json:"name"`
	Extension     string    `json:"extension"`
	IsDir         bool      `json:"is_dir"`
	Size          int64     `json:"size"`
	AllocatedSize int64     `json:"allocated_size"`
	ModTime       time.Time `json:"mod_time"`
	IsHidden      bool      `json:"is_hidden"`
	IsSymlink     bool      `json:"is_symlink"`
	SymlinkTarget string    `json:"symlink_target,omitempty"`
	Dev           uint64    `json:"dev,omitempty"`
	Ino           uint64    `json:"ino,omitempty"`
	ScanStatus    string    `json:"scan_status,omitempty"`
}

// ExtractEntry extracts metadata from os.FileInfo and constructs an Entry.
func ExtractEntry(path string, fi os.FileInfo) Entry {
	cleanPath := filepath.Clean(path)
	name := fi.Name()
	parent := filepath.Dir(cleanPath)
	isDir := fi.IsDir()
	size := fi.Size()
	allocatedSize := size
	modTime := fi.ModTime()
	isSymlink := fi.Mode()&os.ModeSymlink != 0
	isHidden := strings.HasPrefix(name, ".")

	var target string
	if isSymlink {
		if link, err := os.Readlink(cleanPath); err == nil {
			target = link
		}
	}

	ext := ""
	if !isDir {
		ext = strings.ToLower(filepath.Ext(name))
	}

	dev, ino, allocatedSize := getPlatformStat(fi, allocatedSize)

	return Entry{
		Path:          cleanPath,
		ParentPath:    parent,
		Name:          name,
		Extension:     ext,
		IsDir:         isDir,
		Size:          size,
		AllocatedSize: allocatedSize,
		ModTime:       modTime,
		IsHidden:      isHidden,
		IsSymlink:     isSymlink,
		SymlinkTarget: target,
		Dev:           dev,
		Ino:           ino,
		ScanStatus:    "complete",
	}
}
