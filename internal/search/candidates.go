package search

import "github.com/AgentNex/delvo/internal/scanner"

// SearchResult wraps a matched Entry with match metadata for UI rendering.
type SearchResult struct {
	Entry          scanner.Entry
	Score          int
	MatchedIndices []int
}

// SearchOptions defines the parameters for a search operation.
type SearchOptions struct {
	Query         string
	CaseSensitive bool
	ScopeAllRoots bool
	CurrentDir    string
	ExactOnly     bool
}

// EntryList implements fuzzy.Source for a slice of scanner.Entry.
type EntryList []scanner.Entry

func (el EntryList) Len() int {
	return len(el)
}

func (el EntryList) String(i int) string {
	return el[i].Name
}

// EntryPathList implements fuzzy.Source matching full path.
type EntryPathList []scanner.Entry

func (el EntryPathList) Len() int {
	return len(el)
}

func (el EntryPathList) String(i int) string {
	return el[i].Path
}
