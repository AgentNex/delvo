package tests

import (
	"testing"
	"time"

	"github.com/AgentNex/delvo/internal/scanner"
	"github.com/AgentNex/delvo/internal/search"
)

func TestFuzzySearchAndRanking(t *testing.T) {
	entries := []scanner.Entry{
		{Path: "/app/main.go", Name: "main.go", Extension: ".go", IsDir: false},
		{Path: "/app/cmd/dvo/main.go", Name: "main.go", Extension: ".go", IsDir: false},
		{Path: "/app/maintainer.txt", Name: "maintainer.txt", Extension: ".txt", IsDir: false},
		{Path: "/app/domain.go", Name: "domain.go", Extension: ".go", IsDir: false},
		{Path: "/app/images/logo.png", Name: "logo.png", Extension: ".png", IsDir: false},
	}

	// 1. Search for "main" - exact name matches must rank highest
	res := search.Search(entries, search.SearchOptions{
		Query:         "main",
		CaseSensitive: false,
		ScopeAllRoots: true,
	})

	if len(res) < 2 {
		t.Fatalf("expected at least 2 matches for 'main', got %d", len(res))
	}
	if res[0].Entry.Name != "main.go" {
		t.Errorf("expected top result to be main.go, got %s", res[0].Entry.Name)
	}

	// 2. Extension search for ".go"
	resGo := search.Search(entries, search.SearchOptions{
		Query:         ".go",
		CaseSensitive: false,
		ScopeAllRoots: true,
	})
	if len(resGo) != 3 {
		t.Errorf("expected 3 .go files, got %d", len(resGo))
	}

	// 3. Glob matching: "*.png"
	resGlob := search.Search(entries, search.SearchOptions{
		Query:         "*.png",
		CaseSensitive: false,
		ScopeAllRoots: true,
	})
	if len(resGlob) != 1 || resGlob[0].Entry.Name != "logo.png" {
		t.Errorf("expected logo.png for glob, got %v", resGlob)
	}

	// 4. Case sensitive mode
	entriesCase := []scanner.Entry{
		{Path: "/a/README.md", Name: "README.md"},
		{Path: "/a/readme.txt", Name: "readme.txt"},
	}
	resCase := search.Search(entriesCase, search.SearchOptions{
		Query:         "README",
		CaseSensitive: true,
		ExactOnly:     true,
	})
	if len(resCase) != 1 || resCase[0].Entry.Name != "README.md" {
		t.Errorf("expected only README.md in case sensitive mode, got %v", resCase)
	}
}

func BenchmarkFuzzySearch(b *testing.B) {
	// Generate 1000 candidate entries
	entries := make([]scanner.Entry, 1000)
	for i := 0; i < 1000; i++ {
		entries[i] = scanner.Entry{
			Path:      "/data/project/file_" + string(rune('a'+(i%26))) + ".txt",
			Name:      "file_" + string(rune('a'+(i%26))) + ".txt",
			Extension: ".txt",
			ModTime:   time.Now(),
		}
	}

	opts := search.SearchOptions{
		Query:         "file_m",
		CaseSensitive: false,
		ScopeAllRoots: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = search.Search(entries, opts)
	}
}
