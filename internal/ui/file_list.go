package ui

import (
	"fmt"
	"strings"

	"github.com/AgentNex/delvo/internal/scanner"
	"github.com/charmbracelet/x/ansi"
)

// FileListParams supplies state to render the virtualized file table.
type FileListParams struct {
	Entries        []scanner.Entry
	Selected       map[string]struct{}
	Cursor         int
	Offset         int
	MaxRows        int
	Width          int
	Theme          Theme
	ParentSize     int64
	MatchedIndices map[int][]int // Entry index -> matched character positions
	ShowDiskUsage  bool
}

// RenderFileList produces the formatted lines for the visible window of files.
func RenderFileList(p FileListParams) string {
	if len(p.Entries) == 0 {
		emptyMsg := "  (no files found or empty directory)"
		return p.Theme.TextMuted.Render(emptyMsg)
	}

	start := p.Offset
	if start < 0 {
		start = 0
	}
	end := start + p.MaxRows
	if end > len(p.Entries) {
		end = len(p.Entries)
	}

	var sb strings.Builder

	for i := start; i < end; i++ {
		e := p.Entries[i]
		isCursor := i == p.Cursor
		_, isSelected := p.Selected[e.Path]

		// 0. Active row pointer
		cursorMark := "  "
		if isCursor {
			cursorMark = "▸ "
		}

		// 1. Selection marker
		var mark string
		if isSelected {
			mark = p.Theme.SelectedMark.Render("[✓]")
		} else {
			mark = p.Theme.UnselectedMark.Render("[ ]")
		}

		// 2. Type badge
		var typeBadge string
		if e.IsSymlink {
			typeBadge = p.Theme.SymlinkName.Render("[S]")
		} else if e.IsDir {
			typeBadge = p.Theme.DirName.Render("[D]")
		} else {
			typeBadge = p.Theme.FileName.Render("[F]")
		}

		// 3. Size string
		sizeStr := scanner.FormatSize(e.Size)
		sizeFormatted := fmt.Sprintf("%8s", sizeStr)

		// 4. Usage bar (if disk usage mode and directory)
		var usageBar string
		if p.ShowDiskUsage && p.Width > 55 {
			barW := 14
			if p.Width > 80 {
				barW = 18
			}
			usageBar = " " + RenderDiskBar(e.Size, p.ParentSize, barW, p.Theme)
		}

		// 5. Short time (e.g. "02 Jan 15:04")
		timeStr := e.ModTime.Format("02 Jan 15:04")

		// Calculate available width for name using ANSI visual width
		// Reserved: cursor(2) + mark(3) + space(1) + type(3) + space(1) + size(8) + space(1) + time(12) + usageBar
		reserved := 2 + 3 + 1 + 3 + 1 + 8 + 1 + 12 + ansi.StringWidth(usageBar) + 2
		nameWidth := p.Width - reserved
		if nameWidth < 8 {
			nameWidth = 8
		}

		var displayName string
		if p.MatchedIndices != nil && len(p.MatchedIndices[i]) > 0 {
			displayName = highlightMatches(e.Name, p.MatchedIndices[i], p.Theme)
		} else {
			displayName = e.Name
		}
		truncatedName := TruncateString(displayName, nameWidth)
		paddedName := PadRight(truncatedName, nameWidth)

		// Style name
		if e.IsDir {
			paddedName = p.Theme.DirName.Render(paddedName)
		} else if e.IsSymlink {
			paddedName = p.Theme.SymlinkName.Render(paddedName)
		} else {
			paddedName = p.Theme.FileName.Render(paddedName)
		}

		// Row content assembly
		rowContent := fmt.Sprintf("%s%s %s %s %s %s%s", cursorMark, mark, typeBadge, paddedName, sizeFormatted, timeStr, usageBar)

		if isCursor {
			sb.WriteString(p.Theme.RowActive.Render(rowContent))
		} else {
			sb.WriteString(p.Theme.RowNormal.Render(rowContent))
		}

		if i < end-1 {
			sb.WriteByte('\n')
		}
	}

	return sb.String()
}

func highlightMatches(name string, indices []int, theme Theme) string {
	idxMap := make(map[int]bool, len(indices))
	for _, idx := range indices {
		idxMap[idx] = true
	}

	var sb strings.Builder
	for i, r := range name {
		if idxMap[i] {
			sb.WriteString(theme.MatchHighlight.Render(string(r)))
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}
