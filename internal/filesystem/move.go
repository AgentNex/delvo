package filesystem

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// MoveItem safely moves an item from src to dst.
func MoveItem(src, dst string, roots []string) error {
	// 1. Boundary check: ensure both src and dst are within roots
	if !IsInsideRoots(src, roots) {
		return fmt.Errorf("source path is outside configured roots: %s", src)
	}
	if !IsInsideRoots(dst, roots) {
		return fmt.Errorf("destination path is outside configured roots: %s", dst)
	}

	fi, err := os.Lstat(src)
	if err != nil {
		return fmt.Errorf("stat source: %w", err)
	}

	// 2. Ensure parent of destination exists
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("create destination parent dir: %w", err)
	}

	// 3. Try standard atomic rename
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	// 4. Fallback for cross-device link errors (EXDEV)
	return copyAndRemove(src, dst, fi)
}

// RenameItem renames a file or directory within its current directory.
func RenameItem(src, newName string, roots []string) error {
	dir := filepath.Dir(src)
	dst := filepath.Join(dir, newName)
	return MoveItem(src, dst, roots)
}

// copyAndRemove copies src to dst and removes src upon successful copy.
func copyAndRemove(src, dst string, fi os.FileInfo) error {
	if fi.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(src)
		if err != nil {
			return fmt.Errorf("readlink src: %w", err)
		}
		if err := os.Symlink(target, dst); err != nil {
			return fmt.Errorf("create symlink: %w", err)
		}
		return os.Remove(src)
	}

	if fi.IsDir() {
		if err := os.MkdirAll(dst, fi.Mode().Perm()); err != nil {
			return fmt.Errorf("mkdir dst: %w", err)
		}

		entries, err := os.ReadDir(src)
		if err != nil {
			return fmt.Errorf("readdir src: %w", err)
		}

		for _, entry := range entries {
			childSrc := filepath.Join(src, entry.Name())
			childDst := filepath.Join(dst, entry.Name())
			childInfo, err := entry.Info()
			if err != nil {
				return fmt.Errorf("stat child %s: %w", childSrc, err)
			}
			if err := copyAndRemove(childSrc, childDst, childInfo); err != nil {
				return err
			}
		}

		return os.Remove(src)
	}

	// Regular file copy
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open src: %w", err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fi.Mode().Perm())
	if err != nil {
		return fmt.Errorf("create dst: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy data: %w", err)
	}

	if err := out.Sync(); err != nil {
		return fmt.Errorf("sync dst: %w", err)
	}

	// Close before removing on Windows or strict systems
	_ = in.Close()
	_ = out.Close()

	return os.Remove(src)
}
