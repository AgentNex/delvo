package ui

import (
	"fmt"
	"strings"

	"github.com/AgentNex/delvo/internal/scanner"
)

// HeaderParams contains data to render the top navigation and search bar.
type HeaderParams struct {
	ActiveTab    int // 0: Browse, 1: Search, 2: DiskUsage, 3: Filter
	SearchQuery  string
	SearchActive bool
	MatchCount   int
	TotalCount   int
	CurrentPath  string
	Progress     scanner.ScanProgress
	Width        int
	Theme        Theme
}

// RenderHeader builds the 3-line top bar of Delvo.
func RenderHeader(p HeaderParams) string {
	var lines []string

	// Line 1: Branding and Tabs
	title := p.Theme.Title.Render("DELVO") + " " + p.Theme.Subtitle.Render("v0.1.0")

	tabs := []string{"[1] Browse", "[2] Search", "[3] Disk Usage", "[4] Filters"}
	var styledTabs []string
	for i, tab := range tabs {
		if i == p.ActiveTab {
			styledTabs = append(styledTabs, p.Theme.RowActive.Render(tab))
		} else {
			styledTabs = append(styledTabs, p.Theme.HelpDesc.Render(tab))
		}
	}
	tabsRow := strings.Join(styledTabs, " ")

	line1 := title + "  " + tabsRow
	if p.Progress.IsProvisional {
		scanBadge := p.Theme.Badge.Render("SCANNING...")
		line1 = line1 + " " + scanBadge
	}
	lines = append(lines, TruncateString(line1, p.Width))

	// Line 2: Path or Search Input
	var line2 string
	if p.SearchActive || p.ActiveTab == 1 {
		prefix := p.Theme.Cyan.Render("🔍 Search: ")
		queryDisplay := p.SearchQuery
		if p.SearchActive {
			queryDisplay = queryDisplay + "█" // Cursor
		}
		stats := fmt.Sprintf(" (%d matches / %d indexed)", p.MatchCount, p.TotalCount)
		line2 = prefix + queryDisplay + p.Theme.HelpDesc.Render(stats)
	} else {
		prefix := p.Theme.Cyan.Render("📂 Path: ")
		line2 = prefix + p.Theme.FileName.Render(p.CurrentPath)
	}
	lines = append(lines, TruncateString(line2, p.Width))

	// Line 3: Subtle separator
	sep := strings.Repeat("─", p.Width)
	lines = append(lines, p.Theme.PanelBorderDim.Render(sep))

	return strings.Join(lines, "\n")
}
