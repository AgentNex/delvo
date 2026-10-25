package filter

import (
	"strings"
	"time"

	"github.com/AgentNex/delvo/internal/scanner"
)

// EntryTypeFilter determines what kind of entries are included.
type EntryTypeFilter string

const (
	TypeAll   EntryTypeFilter = "all"
	TypeFiles EntryTypeFilter = "files"
	TypeDirs  EntryTypeFilter = "dirs"
)

// SizeFilter represents size thresholds.
type SizeFilter string

const (
	SizeAll     SizeFilter = "all"
	SizeEmpty   SizeFilter = "empty" // 0 bytes
	SizeGt1MB   SizeFilter = ">1MB"
	SizeGt10MB  SizeFilter = ">10MB"
	SizeGt100MB SizeFilter = ">100MB"
	SizeGt1GB   SizeFilter = ">1GB"
)

// TimeFilter represents modification time windows.
type TimeFilter string

const (
	TimeAll   TimeFilter = "all"
	TimeToday TimeFilter = "today"
	TimeWeek  TimeFilter = "7d"
	TimeMonth TimeFilter = "30d"
)

// FilterState bundles all active filtering parameters.
type FilterState struct {
	Type       EntryTypeFilter
	Extension  string // e.g. ".go", ".mp4", or "" for any
	Size       SizeFilter
	Time       TimeFilter
	ShowHidden bool
}

// DefaultFilterState returns the initial default filter state.
func DefaultFilterState() FilterState {
	return FilterState{
		Type:       TypeAll,
		Extension:  "",
		Size:       SizeAll,
		Time:       TimeAll,
		ShowHidden: false,
	}
}

// Matches returns true if the entry satisfies the filter conditions.
func (f FilterState) Matches(e scanner.Entry) bool {
	// 1. Hidden filter
	if !f.ShowHidden && e.IsHidden {
		return false
	}

	// 2. Type filter
	switch f.Type {
	case TypeFiles:
		if e.IsDir {
			return false
		}
	case TypeDirs:
		if !e.IsDir {
			return false
		}
	}

	// 3. Extension filter
	if f.Extension != "" {
		if e.IsDir {
			return false
		}
		targetExt := f.Extension
		if !strings.HasPrefix(targetExt, ".") {
			targetExt = "." + targetExt
		}
		if !strings.EqualFold(e.Extension, targetExt) {
			return false
		}
	}

	// 4. Size filter
	switch f.Size {
	case SizeEmpty:
		if e.Size != 0 {
			return false
		}
	case SizeGt1MB:
		if e.Size <= 1*1024*1024 {
			return false
		}
	case SizeGt10MB:
		if e.Size <= 10*1024*1024 {
			return false
		}
	case SizeGt100MB:
		if e.Size <= 100*1024*1024 {
			return false
		}
	case SizeGt1GB:
		if e.Size <= 1024*1024*1024 {
			return false
		}
	}

	// 5. Time filter
	if f.Time != TimeAll {
		now := time.Now()
		var threshold time.Time
		switch f.Time {
		case TimeToday:
			threshold = now.Add(-24 * time.Hour)
		case TimeWeek:
			threshold = now.Add(-7 * 24 * time.Hour)
		case TimeMonth:
			threshold = now.Add(-30 * 24 * time.Hour)
		}

		if e.ModTime.Before(threshold) {
			return false
		}
	}

	return true
}

// ApplyFilters filters a slice of scanner.Entry according to FilterState.
func ApplyFilters(entries []scanner.Entry, f FilterState) []scanner.Entry {
	filtered := make([]scanner.Entry, 0, len(entries))
	for _, e := range entries {
		if f.Matches(e) {
			filtered = append(filtered, e)
		}
	}
	return filtered
}
