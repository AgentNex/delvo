package search

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/AgentNex/delvo/internal/scanner"
	"github.com/sahilm/fuzzy"
)

// Search filters and ranks entries based on the given search options.
func Search(entries []scanner.Entry, opts SearchOptions) []SearchResult {
	if len(entries) == 0 {
		return nil
	}

	// 1. Filter candidates by current directory if not searching all roots
	var candidates []scanner.Entry
	if !opts.ScopeAllRoots && opts.CurrentDir != "" {
		curClean := filepath.Clean(opts.CurrentDir)
		for _, e := range entries {
			if e.ParentPath == curClean || e.Path == curClean {
				candidates = append(candidates, e)
			}
		}
	} else {
		candidates = entries
	}

	query := strings.TrimSpace(opts.Query)
	if query == "" {
		// Return all candidates with score 0
		results := make([]SearchResult, len(candidates))
		for i, c := range candidates {
			results[i] = SearchResult{
				Entry: c,
				Score: 0,
			}
		}
		return results
	}

	// 2. Check if query is a glob pattern
	isGlob := strings.ContainsAny(query, "*?[]")
	if isGlob {
		var results []SearchResult
		pattern := query
		if !opts.CaseSensitive {
			pattern = strings.ToLower(pattern)
		}

		for _, e := range candidates {
			target := e.Name
			if !opts.CaseSensitive {
				target = strings.ToLower(target)
			}

			matched, err := filepath.Match(pattern, target)
			if err == nil && matched {
				results = append(results, SearchResult{
					Entry: e,
					Score: 500 + CalculateBonus(e, query, opts.CaseSensitive),
				})
			}
		}

		sort.SliceStable(results, func(i, j int) bool {
			return results[i].Score > results[j].Score
		})
		return results
	}

	// 3. Exact matching mode if requested
	if opts.ExactOnly {
		var results []SearchResult
		q := query
		if !opts.CaseSensitive {
			q = strings.ToLower(q)
		}

		for _, e := range candidates {
			name := e.Name
			if !opts.CaseSensitive {
				name = strings.ToLower(name)
			}
			if strings.Contains(name, q) {
				results = append(results, SearchResult{
					Entry: e,
					Score: 1000 + CalculateBonus(e, query, opts.CaseSensitive),
				})
			}
		}

		sort.SliceStable(results, func(i, j int) bool {
			return results[i].Score > results[j].Score
		})
		return results
	}

	// 4. Fuzzy matching using sahilm/fuzzy
	pattern := query
	if !opts.CaseSensitive {
		pattern = strings.ToLower(pattern)
	}

	// Match on filename first
	nameSource := make([]string, len(candidates))
	for i, c := range candidates {
		if opts.CaseSensitive {
			nameSource[i] = c.Name
		} else {
			nameSource[i] = strings.ToLower(c.Name)
		}
	}

	matches := fuzzy.Find(pattern, nameSource)
	results := make([]SearchResult, 0, len(matches))
	seenIndex := make(map[int]bool)

	for _, m := range matches {
		entry := candidates[m.Index]
		bonus := CalculateBonus(entry, query, opts.CaseSensitive)
		results = append(results, SearchResult{
			Entry:          entry,
			Score:          m.Score*10 + bonus,
			MatchedIndices: m.MatchedIndexes,
		})
		seenIndex[m.Index] = true
	}

	// Also search full paths if not matched by basename and query contains path separator
	if strings.Contains(query, "/") {
		pathSource := make([]string, len(candidates))
		for i, c := range candidates {
			if opts.CaseSensitive {
				pathSource[i] = c.Path
			} else {
				pathSource[i] = strings.ToLower(c.Path)
			}
		}
		pathMatches := fuzzy.Find(pattern, pathSource)
		for _, pm := range pathMatches {
			if seenIndex[pm.Index] {
				continue
			}
			entry := candidates[pm.Index]
			results = append(results, SearchResult{
				Entry: entry,
				Score: pm.Score * 5, // slightly lower than direct filename match
			})
		}
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		// Tiebreak by directory first, then name
		if results[i].Entry.IsDir != results[j].Entry.IsDir {
			return results[i].Entry.IsDir
		}
		return results[i].Entry.Name < results[j].Entry.Name
	})

	return results
}
