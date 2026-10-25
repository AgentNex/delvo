package tests

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/AgentNex/delvo/internal/index"
	"github.com/AgentNex/delvo/internal/scanner"
)

func TestIndexOperations(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	idx, err := index.New(dbPath)
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}
	defer idx.Close()

	ctx := context.Background()

	// 1. Batch upsert entries
	now := time.Now().Truncate(time.Second)
	entries := []scanner.Entry{
		{
			Path:          "/home/user/doc.txt",
			ParentPath:    "/home/user",
			Name:          "doc.txt",
			Extension:     ".txt",
			IsDir:         false,
			Size:          1024,
			AllocatedSize: 4096,
			ModTime:       now,
			IsHidden:      false,
			ScanStatus:    "complete",
		},
		{
			Path:          "/home/user/photos",
			ParentPath:    "/home/user",
			Name:          "photos",
			Extension:     "",
			IsDir:         true,
			Size:          0,
			AllocatedSize: 0,
			ModTime:       now,
			IsHidden:      false,
			ScanStatus:    "complete",
		},
	}

	if err := idx.BatchUpsert(ctx, entries); err != nil {
		t.Fatalf("BatchUpsert failed: %v", err)
	}

	// 2. Query entries in dir
	children, err := idx.GetEntriesInDir(ctx, "/home/user")
	if err != nil {
		t.Fatalf("GetEntriesInDir failed: %v", err)
	}
	if len(children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(children))
	}
	// Directory must come first due to ORDER BY is_dir DESC
	if !children[0].IsDir || children[0].Name != "photos" {
		t.Errorf("expected directory 'photos' first, got %v", children[0])
	}

	// 3. Single entry lookup
	single, err := idx.GetEntry(ctx, "/home/user/doc.txt")
	if err != nil || single == nil {
		t.Fatalf("GetEntry failed: %v, %v", err, single)
	}
	if single.Size != 1024 || single.AllocatedSize != 4096 {
		t.Errorf("metadata mismatch: %+v", single)
	}

	// 4. Update dir size
	if err := idx.UpdateDirSizes(ctx, map[string]int64{"/home/user/photos": 500000}); err != nil {
		t.Fatalf("UpdateDirSizes failed: %v", err)
	}
	photos, _ := idx.GetEntry(ctx, "/home/user/photos")
	if photos == nil || photos.Size != 500000 {
		t.Errorf("expected photos size 500000, got %+v", photos)
	}

	// 5. Delete entry
	if err := idx.DeleteEntry(ctx, "/home/user/doc.txt"); err != nil {
		t.Fatalf("DeleteEntry failed: %v", err)
	}
	deleted, _ := idx.GetEntry(ctx, "/home/user/doc.txt")
	if deleted != nil {
		t.Errorf("expected entry to be deleted, found %+v", deleted)
	}

	// 6. Test rebuild
	if err := idx.Rebuild(ctx); err != nil {
		t.Fatalf("Rebuild failed: %v", err)
	}
	afterRebuild, _ := idx.GetAllEntries(ctx)
	if len(afterRebuild) != 0 {
		t.Errorf("expected 0 entries after rebuild, got %d", len(afterRebuild))
	}
}
