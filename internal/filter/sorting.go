package filter

import (
	"sort"
	"strings"

	"github.com/AgentNex/delvo/internal/scanner"
)

// SortField identifies the attribute used for sorting entries.
type SortField string

const (
	SortByName    SortField = "name"
	SortBySize    SortField = "size"
	SortByModTime SortField = "mod_time"
	SortByExt     SortField = "extension"
)

// SortState defines the active sorting criteria.
type SortState struct {
	Field      SortField
	Descending bool
	DirsFirst  bool
}

// DefaultSortState returns the standard file manager sort (dirs first, name asc).
func DefaultSortState() SortState {
	return SortState{
		Field:      SortByName,
		Descending: false,
		DirsFirst:  true,
	}
}

// SortEntries sorts a slice of entries in place according to SortState.
func SortEntries(entries []scanner.Entry, s SortState) {
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]

		// 1. Group directories first if enabled
		if s.DirsFirst && a.IsDir != b.IsDir {
			return a.IsDir
		}

		// 2. Attribute comparison
		var less bool
		switch s.Field {
		case SortBySize:
			if a.Size == b.Size {
				less = strings.ToLower(a.Name) < strings.ToLower(b.Name)
			} else {
				less = a.Size < b.Size
			}
		case SortByModTime:
			if a.ModTime.Equal(b.ModTime) {
				less = strings.ToLower(a.Name) < strings.ToLower(b.Name)
			} else {
				less = a.ModTime.Before(b.ModTime)
			}
		case SortByExt:
			if a.Extension == b.Extension {
				less = strings.ToLower(a.Name) < strings.ToLower(b.Name)
			} else {
				less = strings.ToLower(a.Extension) < strings.ToLower(b.Extension)
			}
		case SortByName:
			fallthrough
		default:
			less = strings.ToLower(a.Name) < strings.ToLower(b.Name)
		}

		if s.Descending {
			return !less
		}
		return less
	})
}
