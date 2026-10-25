package tests

import (
	"testing"
	"time"

	"github.com/AgentNex/delvo/internal/filter"
	"github.com/AgentNex/delvo/internal/scanner"
)

func TestFiltering(t *testing.T) {
	now := time.Now()
	entries := []scanner.Entry{
		{Name: ".hidden_file", IsHidden: true, IsDir: false, Size: 100, Extension: ""},
		{Name: "app.go", IsHidden: false, IsDir: false, Size: 2 * 1024 * 1024, Extension: ".go", ModTime: now},
		{Name: "data.log", IsHidden: false, IsDir: false, Size: 50 * 1024 * 1024, Extension: ".log", ModTime: now.Add(-10 * 24 * time.Hour)},
		{Name: "zero.bin", IsHidden: false, IsDir: false, Size: 0, Extension: ".bin", ModTime: now},
		{Name: "folder", IsHidden: false, IsDir: true, Size: 52 * 1024 * 1024, Extension: ""},
	}

	// 1. Default: hide hidden files
	fDefault := filter.DefaultFilterState()
	res := filter.ApplyFilters(entries, fDefault)
	if len(res) != 4 {
		t.Errorf("expected 4 entries without hidden, got %d", len(res))
	}

	// 2. Type filter: directories only
	fDirs := filter.DefaultFilterState()
	fDirs.Type = filter.TypeDirs
	resDirs := filter.ApplyFilters(entries, fDirs)
	if len(resDirs) != 1 || resDirs[0].Name != "folder" {
		t.Errorf("expected only folder, got %v", resDirs)
	}

	// 3. Extension filter: .go
	fExt := filter.DefaultFilterState()
	fExt.Extension = ".go"
	resExt := filter.ApplyFilters(entries, fExt)
	if len(resExt) != 1 || resExt[0].Name != "app.go" {
		t.Errorf("expected only app.go, got %v", resExt)
	}

	// 4. Size filter: > 10MB
	fSize := filter.DefaultFilterState()
	fSize.Size = filter.SizeGt10MB
	resSize := filter.ApplyFilters(entries, fSize)
	if len(resSize) != 2 { // data.log (50MB) and folder (52MB)
		t.Errorf("expected 2 items > 10MB, got %d", len(resSize))
	}

	// 5. Size filter: empty files
	fEmpty := filter.DefaultFilterState()
	fEmpty.Size = filter.SizeEmpty
	resEmpty := filter.ApplyFilters(entries, fEmpty)
	if len(resEmpty) != 1 || resEmpty[0].Name != "zero.bin" {
		t.Errorf("expected zero.bin for empty filter, got %v", resEmpty)
	}
}

func TestSorting(t *testing.T) {
	now := time.Now()
	entries := []scanner.Entry{
		{Name: "zebra.txt", IsDir: false, Size: 10, ModTime: now.Add(-1 * time.Hour)},
		{Name: "alpha.txt", IsDir: false, Size: 500, ModTime: now},
		{Name: "beta_dir", IsDir: true, Size: 200, ModTime: now.Add(-2 * time.Hour)},
	}

	// 1. Sort by name asc, dirs first
	sName := filter.SortState{Field: filter.SortByName, Descending: false, DirsFirst: true}
	filter.SortEntries(entries, sName)
	if entries[0].Name != "beta_dir" || entries[1].Name != "alpha.txt" || entries[2].Name != "zebra.txt" {
		t.Errorf("unexpected sort by name: %v", entries)
	}

	// 2. Sort by size desc, dirs first
	sSize := filter.SortState{Field: filter.SortBySize, Descending: true, DirsFirst: true}
	filter.SortEntries(entries, sSize)
	if entries[0].Name != "beta_dir" || entries[1].Name != "alpha.txt" || entries[2].Name != "zebra.txt" {
		t.Errorf("unexpected sort by size: %v", entries)
	}
}
