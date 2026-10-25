package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/AgentNex/delvo/internal/filesystem"
)

func TestValidateRoot(t *testing.T) {
	tempDir := t.TempDir()

	valid, err := filesystem.ValidateRoot(tempDir)
	if err != nil {
		t.Fatalf("expected valid root, got error: %v", err)
	}
	if valid != tempDir {
		t.Errorf("expected %s, got %s", tempDir, valid)
	}

	// Test non-existent path
	_, err = filesystem.ValidateRoot(filepath.Join(tempDir, "does-not-exist"))
	if err == nil {
		t.Errorf("expected error for non-existent path, got nil")
	}

	// Test file instead of dir
	filePath := filepath.Join(tempDir, "file.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err = filesystem.ValidateRoot(filePath)
	if err == nil {
		t.Errorf("expected error for file root, got nil")
	}
}

func TestIsInsideRoots(t *testing.T) {
	root1 := "/data/data/com.termux/files/home"
	root2 := "/data/data/com.termux/files/home/storage/shared"
	roots := []string{root1, root2}

	// Inside root1
	if !filesystem.IsInsideRoots("/data/data/com.termux/files/home/delvo", roots) {
		t.Errorf("expected path inside root1")
	}

	// Inside root2
	if !filesystem.IsInsideRoots("/data/data/com.termux/files/home/storage/shared/DCIM", roots) {
		t.Errorf("expected path inside root2")
	}

	// Outside roots (traversal attempt)
	if filesystem.IsInsideRoots("/data/data/com.termux/files/usr/bin/bash", roots) {
		t.Errorf("expected path outside roots")
	}
	if filesystem.IsInsideRoots("/data/data/com.termux/files/home/../../etc/passwd", roots) {
		t.Errorf("expected traversal path outside roots")
	}
}

func TestResolveOverlappingPaths(t *testing.T) {
	paths := []string{
		"/a/b/c.txt",
		"/a/b",
		"/a/b/d/e.txt",
		"/x/y.txt",
		"/a",
	}

	resolved := filesystem.ResolveOverlappingPaths(paths)

	// Since /a is in the list, /a/b, /a/b/c.txt, and /a/b/d/e.txt must all be pruned
	if len(resolved) != 2 {
		t.Fatalf("expected 2 resolved paths, got %d: %v", len(resolved), resolved)
	}

	if resolved[0] != "/a" || resolved[1] != "/x/y.txt" {
		t.Errorf("expected [/a, /x/y.txt], got %v", resolved)
	}
}

func TestDeleteItemsAndSymlinkProtection(t *testing.T) {
	tempDir := t.TempDir()
	roots := []string{tempDir}

	// 1. Create target file and a symlink pointing to it
	targetFile := filepath.Join(tempDir, "target.txt")
	if err := os.WriteFile(targetFile, []byte("target-content"), 0644); err != nil {
		t.Fatal(err)
	}

	symlinkFile := filepath.Join(tempDir, "link.txt")
	if err := os.Symlink(targetFile, symlinkFile); err != nil {
		t.Skipf("symlink not supported on this filesystem: %v", err)
	}

	// 2. Delete symlink permanently
	results := filesystem.DeleteItems([]string{symlinkFile}, roots, filesystem.DeleteOptions{
		Permanent: true,
	})
	if len(results) != 1 || !results[0].Success {
		t.Fatalf("expected successful symlink delete, got: %v", results)
	}

	// 3. Verify symlink was deleted BUT target was preserved!
	if _, err := os.Lstat(symlinkFile); !os.IsNotExist(err) {
		t.Errorf("expected symlink to be deleted")
	}
	if _, err := os.Stat(targetFile); err != nil {
		t.Errorf("expected target file to remain untouched, got error: %v", err)
	}

	// 4. Test Trash workflow
	trashDir := filepath.Join(tempDir, "trash")
	regularFile := filepath.Join(tempDir, "regular.txt")
	if err := os.WriteFile(regularFile, []byte("sample"), 0644); err != nil {
		t.Fatal(err)
	}

	trashResults := filesystem.DeleteItems([]string{regularFile}, roots, filesystem.DeleteOptions{
		Permanent: false,
		TrashDir:  trashDir,
	})
	if len(trashResults) != 1 || !trashResults[0].Success {
		t.Fatalf("expected successful trash, got: %v", trashResults)
	}

	if _, err := os.Stat(regularFile); !os.IsNotExist(err) {
		t.Errorf("expected original file removed after trash")
	}
	if _, err := os.Stat(trashResults[0].MovedTo); err != nil {
		t.Errorf("expected trashed file at %s, got: %v", trashResults[0].MovedTo, err)
	}
}

func TestMoveItem(t *testing.T) {
	tempDir := t.TempDir()
	roots := []string{tempDir}

	src := filepath.Join(tempDir, "origin.txt")
	dst := filepath.Join(tempDir, "subdir", "moved.txt")

	if err := os.WriteFile(src, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := filesystem.MoveItem(src, dst, roots); err != nil {
		t.Fatalf("failed to move item: %v", err)
	}

	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("source should not exist after move")
	}

	data, err := os.ReadFile(dst)
	if err != nil || string(data) != "data" {
		t.Errorf("destination content mismatch: %v, %s", err, string(data))
	}
}
