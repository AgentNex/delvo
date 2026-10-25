package ui

import (
	"fmt"
	"strings"

	"github.com/AgentNex/delvo/internal/scanner"
)

// DetailsParams provides data to render the right-side or modal metadata panel.
type DetailsParams struct {
	Entry      *scanner.Entry
	ParentSize int64
	Width      int
	Height     int
	Theme      Theme
}

// RenderDetailsPanel builds the inspection panel for a selected entry.
func RenderDetailsPanel(p DetailsParams) string {
	if p.Entry == nil {
		return p.Theme.PanelBorderDim.Render("No entry selected")
	}

	e := p.Entry
	var lines []string

	lines = append(lines, p.Theme.Title.Render("── ITEM DETAILS ──"))
	lines = append(lines, fmt.Sprintf("%s %s", p.Theme.HelpKey.Render("Name:"), p.Theme.FileName.Render(TruncateString(e.Name, p.Width-8))))
	lines = append(lines, fmt.Sprintf("%s %s", p.Theme.HelpKey.Render("Path:"), p.Theme.HelpDesc.Render(TruncateString(e.Path, p.Width-8))))

	kind := "Regular File"
	if e.IsSymlink {
		kind = "Symbolic Link"
		if e.SymlinkTarget != "" {
			kind = fmt.Sprintf("Symlink -> %s", e.SymlinkTarget)
		}
	} else if e.IsDir {
		kind = "Directory"
	}
	lines = append(lines, fmt.Sprintf("%s %s", p.Theme.HelpKey.Render("Type:"), p.Theme.DirName.Render(kind)))

	sizeStr := scanner.FormatSize(e.Size)
	if e.AllocatedSize > 0 && e.AllocatedSize != e.Size {
		sizeStr += fmt.Sprintf(" (alloc %s)", scanner.FormatSize(e.AllocatedSize))
	}
	lines = append(lines, fmt.Sprintf("%s %s", p.Theme.HelpKey.Render("Size:"), p.Theme.RowActive.Render(sizeStr)))

	if p.ParentSize > 0 {
		bar := RenderDiskBar(e.Size, p.ParentSize, p.Width-10, p.Theme)
		lines = append(lines, fmt.Sprintf("%s %s", p.Theme.HelpKey.Render("Usage:"), bar))
	}

	lines = append(lines, fmt.Sprintf("%s %s", p.Theme.HelpKey.Render("Modified:"), p.Theme.HelpDesc.Render(e.ModTime.Format("2006-01-02 15:04:05"))))

	if e.Dev != 0 || e.Ino != 0 {
		lines = append(lines, fmt.Sprintf("%s %s", p.Theme.HelpKey.Render("Inode:"), p.Theme.HelpDesc.Render(fmt.Sprintf("%d (dev %d)", e.Ino, e.Dev))))
	}

	content := strings.Join(lines, "\n")
	box := p.Theme.PanelBorderActive.
		Width(p.Width).
		Height(p.Height).
		Render(content)

	return box
}
