package watcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/AgentNex/delvo/internal/index"
	"github.com/AgentNex/delvo/internal/scanner"
)

// ReconcileEvent reconciles a single filesystem change event against the SQLite index.
func ReconcileEvent(ctx context.Context, idx *index.Index, ev ChangeEvent) error {
	switch ev.Type {
	case ChangeRemove, ChangeRename:
		return idx.DeleteEntry(ctx, ev.Path)
	case ChangeCreate, ChangeWrite:
		fi, err := os.Lstat(ev.Path)
		if err != nil {
			if os.IsNotExist(err) {
				return idx.DeleteEntry(ctx, ev.Path)
			}
			return fmt.Errorf("stat event path %s: %w", ev.Path, err)
		}

		entry := scanner.ExtractEntry(ev.Path, fi)
		return idx.BatchUpsert(ctx, []scanner.Entry{entry})
	}

	return nil
}

// ReconcileDir performs a targeted scan and reconciliation of a single directory against the index.
func ReconcileDir(ctx context.Context, idx *index.Index, dirPath string) error {
	cleanDir := filepath.Clean(dirPath)

	diskEntries, err := os.ReadDir(cleanDir)
	if err != nil {
		if os.IsNotExist(err) {
			return idx.DeleteEntry(ctx, cleanDir)
		}
		return fmt.Errorf("read disk dir %s: %w", cleanDir, err)
	}

	diskMap := make(map[string]os.DirEntry, len(diskEntries))
	for _, de := range diskEntries {
		diskMap[de.Name()] = de
	}

	dbEntries, err := idx.GetEntriesInDir(ctx, cleanDir)
	if err != nil {
		return fmt.Errorf("get db entries for %s: %w", cleanDir, err)
	}

	dbMap := make(map[string]scanner.Entry, len(dbEntries))
	for _, e := range dbEntries {
		dbMap[e.Name] = e
	}

	// 1. Remove DB entries that no longer exist on disk
	for name, e := range dbMap {
		if _, exists := diskMap[name]; !exists {
			_ = idx.DeleteEntry(ctx, e.Path)
		}
	}

	// 2. Add or update disk entries in DB
	toUpsert := make([]scanner.Entry, 0, len(diskEntries))
	for name, de := range diskMap {
		fullPath := filepath.Join(cleanDir, name)
		fi, err := de.Info()
		if err != nil {
			continue
		}

		existing, exists := dbMap[name]
		if !exists || existing.Size != fi.Size() || existing.ModTime.Unix() != fi.ModTime().Unix() {
			toUpsert = append(toUpsert, scanner.ExtractEntry(fullPath, fi))
		}
	}

	if len(toUpsert) > 0 {
		return idx.BatchUpsert(ctx, toUpsert)
	}

	return nil
}
