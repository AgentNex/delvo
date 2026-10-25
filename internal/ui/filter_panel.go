package ui

import (
	"fmt"
	"strings"

	"github.com/AgentNex/delvo/internal/filter"
)

// FilterPanelParams supplies state for rendering the filter and sort bar/panel.
type FilterPanelParams struct {
	Filter filter.FilterState
	Sort   filter.SortState
	Width  int
	Theme  Theme
}

// RenderFilterStatusBar renders a concise 1-line status indicator of active sort & filters.
func RenderFilterStatusBar(p FilterPanelParams) string {
	sortDir := "▲"
	if p.Sort.Descending {
		sortDir = "▼"
	}
	sortStr := fmt.Sprintf("Sort: %s %s", p.Sort.Field, sortDir)

	var filterParts []string
	if p.Filter.Type != filter.TypeAll {
		filterParts = append(filterParts, fmt.Sprintf("Type: %s", p.Filter.Type))
	}
	if p.Filter.Extension != "" {
		filterParts = append(filterParts, fmt.Sprintf("Ext: %s", p.Filter.Extension))
	}
	if p.Filter.Size != filter.SizeAll {
		filterParts = append(filterParts, fmt.Sprintf("Size: %s", p.Filter.Size))
	}
	if p.Filter.Time != filter.TimeAll {
		filterParts = append(filterParts, fmt.Sprintf("Time: %s", p.Filter.Time))
	}
	if p.Filter.ShowHidden {
		filterParts = append(filterParts, "Hidden: ON")
	}

	filterStr := "Filters: None"
	if len(filterParts) > 0 {
		filterStr = "Filters: " + strings.Join(filterParts, " | ")
	}

	content := fmt.Sprintf(" %s  │  %s", p.Theme.HelpKey.Render(sortStr), p.Theme.HelpDesc.Render(filterStr))
	return TruncateString(content, p.Width)
}
