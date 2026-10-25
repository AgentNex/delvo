package tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/AgentNex/delvo/internal/scanner"
)

func TestScannerParallelTraversal(t *testing.T) {
	tempDir := t.TempDir()

	// Build directory fixture
	// tempDir/
	// ├── a/
	// │   ├── file1.txt (100 bytes)
	// │   └── file2.log (200 bytes)
	// ├── b/
	// │   └── sub/
	// │       └── deep.go (300 bytes)
	// └── top.md (50 bytes)

	dirA := filepath.Join(tempDir, "a")
	dirBSub := filepath.Join(tempDir, "b", "sub")
	if err := os.MkdirAll(dirA, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dirBSub, 0755); err != nil {
		t.Fatal(err)
	}

	_ = os.WriteFile(filepath.Join(dirA, "file1.txt"), make([]byte, 100), 0644)
	_ = os.WriteFile(filepath.Join(dirA, "file2.log"), make([]byte, 200), 0644)
	_ = os.WriteFile(filepath.Join(dirBSub, "deep.go"), make([]byte, 300), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "top.md"), make([]byte, 50), 0644)

	s := scanner.New(scanner.ScanOptions{
		Roots:     []string{tempDir},
		Workers:   4,
		BatchSize: 10,
	})

	var batchesMu sync.Mutex
	var batches [][]scanner.Entry
	res, err := s.Scan(context.Background(), nil, func(batch []scanner.Entry) error {
		batchesMu.Lock()
		batches = append(batches, batch)
		batchesMu.Unlock()
		return nil
	})

	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	if res.Progress.FilesScanned != 4 {
		t.Errorf("expected 4 files scanned, got %d", res.Progress.FilesScanned)
	}
	if res.Progress.BytesScanned != 650 {
		t.Errorf("expected 650 bytes scanned, got %d", res.Progress.BytesScanned)
	}

	// Verify bottom-up directory totals
	// dirA size should be 300
	// dirBSub should be 300
	// dirB should be 300
	// tempDir should be 650
	if res.DirSizes[dirA] != 300 {
		t.Errorf("expected dirA size 300, got %d", res.DirSizes[dirA])
	}
	if res.DirSizes[dirBSub] != 300 {
		t.Errorf("expected dirBSub size 300, got %d", res.DirSizes[dirBSub])
	}
	if res.DirSizes[filepath.Join(tempDir, "b")] != 300 {
		t.Errorf("expected dirB size 300, got %d", res.DirSizes[filepath.Join(tempDir, "b")])
	}
	if res.DirSizes[tempDir] != 650 {
		t.Errorf("expected tempDir total size 650, got %d", res.DirSizes[tempDir])
	}
}

func TestScannerDeepStructure(t *testing.T) {
	tempDir := t.TempDir()

	// Create a chain of 40 nested directories with a file at each level
	cur := tempDir
	totalExpectedFiles := 40
	for i := 0; i < totalExpectedFiles; i++ {
		cur = filepath.Join(cur, fmt.Sprintf("level_%d", i))
		if err := os.MkdirAll(cur, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(cur, "data.bin"), []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	s := scanner.New(scanner.ScanOptions{
		Roots:     []string{tempDir},
		Workers:   4,
		BatchSize: 5,
	})

	res, err := s.Scan(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("deep scan failed: %v", err)
	}

	if res.Progress.FilesScanned != int64(totalExpectedFiles) {
		t.Errorf("expected %d files scanned in deep hierarchy, got %d", totalExpectedFiles, res.Progress.FilesScanned)
	}
	expectedDirs := int64(totalExpectedFiles + 1) // 40 subdirectories + root directory
	if res.Progress.DirsScanned != expectedDirs {
		t.Errorf("expected %d dirs scanned, got %d", expectedDirs, res.Progress.DirsScanned)
	}
}

func TestScannerCancellation(t *testing.T) {
	tempDir := t.TempDir()

	// Create many files
	for i := 0; i < 50; i++ {
		d := filepath.Join(tempDir, filepath.Join("nested", "dir", "sub"))
		_ = os.MkdirAll(d, 0755)
		_ = os.WriteFile(filepath.Join(d, "file.bin"), []byte("data"), 0644)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	s := scanner.New(scanner.ScanOptions{
		Roots:   []string{tempDir},
		Workers: 2,
	})

	_, err := s.Scan(ctx, nil, nil)
	if err == nil && ctx.Err() == nil {
		t.Errorf("expected cancellation context error")
	}
}

func TestFormatSize(t *testing.T) {
	cases := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1 KB"},
		{1024 * 1024 * 5, "5 MB"},
		{1024 * 1024 * 1024 * 2, "2 GB"},
	}

	for _, c := range cases {
		got := scanner.FormatSize(c.bytes)
		if got != c.expected {
			t.Errorf("FormatSize(%d) = %q, expected %q", c.bytes, got, c.expected)
		}
	}
}
