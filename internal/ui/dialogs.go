package ui

import (
	"fmt"
	"strings"

	"github.com/AgentNex/delvo/internal/scanner"
)

// DeleteConfirmParams supplies parameters for the confirmation modal.
type DeleteConfirmParams struct {
	Paths     []string
	TotalSize int64
	ItemCount int
	Width     int
	Height    int
	Theme     Theme
}

// RenderDeleteConfirm builds the safety confirmation dialog.
func RenderDeleteConfirm(p DeleteConfirmParams) string {
	boxWidth := p.Width - 4
	if boxWidth > 64 {
		boxWidth = 64
	}
	if boxWidth < 30 {
		boxWidth = 30
	}

	var lines []string
	lines = append(lines, p.Theme.DangerRed.Render("⚠ CONFIRM DELETION"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("You are about to delete %d item(s).", p.ItemCount))
	lines = append(lines, fmt.Sprintf("Total size: %s", p.Theme.Title.Render(scanner.FormatSize(p.TotalSize))))
	lines = append(lines, "")

	// Sample of paths
	lines = append(lines, "Affected items:")
	maxPreview := 4
	for i, path := range p.Paths {
		if i >= maxPreview {
			lines = append(lines, fmt.Sprintf("  ... and %d more", len(p.Paths)-maxPreview))
			break
		}
		lines = append(lines, "  • "+TruncateString(path, boxWidth-8))
	}

	lines = append(lines, "")
	lines = append(lines, p.Theme.Mint.Render("[t] Move to Trash (Recommended)"))
	lines = append(lines, p.Theme.DangerRed.Render("[y] Permanently Delete"))
	lines = append(lines, p.Theme.HelpDesc.Render("[n / Esc] Cancel"))

	content := strings.Join(lines, "\n")
	return p.Theme.DialogBox.Width(boxWidth).Render(content)
}

// RenderHelpBar renders the bottom keyboard instructions adapted to width.
func RenderHelpBar(width int, searchActive bool, theme Theme) string {
	if searchActive {
		return theme.HelpDesc.Render(" [Enter] Apply Search │ [Esc] Cancel │ [↑/↓] Navigate ")
	}

	if width < 55 {
		// Ultra compact for narrow phone screens
		return theme.HelpDesc.Render(" [j/k] Nav │ [↵] Open │ [Spc] Sel │ [/] Search │ [f] Filter │ [d] Del │ [q] Quit ")
	}

	if width < 85 {
		// Medium compact
		return theme.HelpDesc.Render(" [↑/↓] Nav │ [Enter] Open │ [Space] Sel │ [/] Search │ [s] Sort │ [f] Filter │ [d] Delete │ [q] Quit ")
	}

	// Full desktop/landscape width
	return theme.HelpDesc.Render(" [↑/k ↓/j] Navigate │ [Enter] Open │ [Space] Select │ [a] All │ [/] Search │ [s] Sort │ [f] Filter │ [d] Delete │ [r] Rescan │ [q] Quit ")
}
