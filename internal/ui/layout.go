package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// LayoutGeometry tracks dimensions and responsive view splits.
type LayoutGeometry struct {
	Width        int
	Height       int
	IsNarrow     bool
	HeaderHeight int
	FooterHeight int
	MainHeight   int
	ListWidth    int
	DetailsWidth int
}

// ComputeLayout calculates dimensions based on current terminal dimensions.
func ComputeLayout(width, height int) LayoutGeometry {
	if width < 30 {
		width = 30
	}
	if height < 10 {
		height = 10
	}

	isNarrow := width < 80
	headerH := 3
	footerH := 2
	mainH := height - headerH - footerH
	if mainH < 4 {
		mainH = 4
	}

	var listW, detailsW int
	if isNarrow {
		listW = width - 2
		detailsW = 0
	} else {
		listW = int(float64(width)*0.64) - 2
		detailsW = width - listW - 4
		if detailsW < 24 {
			detailsW = 24
			listW = width - detailsW - 4
		}
	}

	return LayoutGeometry{
		Width:        width,
		Height:       height,
		IsNarrow:     isNarrow,
		HeaderHeight: headerH,
		FooterHeight: footerH,
		MainHeight:   mainH,
		ListWidth:    listW,
		DetailsWidth: detailsW,
	}
}

// TruncateString cuts string to maxWidth visual columns, appending "…" if shortened.
// It is ANSI-aware and preserves escape sequences without splitting them.
func TruncateString(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	w := ansi.StringWidth(s)
	if w <= maxWidth {
		return s
	}
	return ansi.Truncate(s, maxWidth, "…")
}

// PadRight pads s with spaces up to visual targetWidth columns.
func PadRight(s string, targetWidth int) string {
	w := ansi.StringWidth(s)
	if w >= targetWidth {
		return s
	}
	return s + strings.Repeat(" ", targetWidth-w)
}

// RenderDiskBar generates a gdu/ncdu style usage visual bar [████░░░░] 50%.
func RenderDiskBar(current, total int64, barWidth int, theme Theme) string {
	if barWidth <= 6 {
		return ""
	}

	graphWidth := barWidth - 6 // reserve room for " 100%"
	if graphWidth <= 2 {
		graphWidth = 2
	}

	var ratio float64
	if total > 0 {
		ratio = float64(current) / float64(total)
		if ratio > 1.0 {
			ratio = 1.0
		}
		if ratio < 0 {
			ratio = 0
		}
	}

	filledLen := int(ratio * float64(graphWidth))
	if filledLen > graphWidth {
		filledLen = graphWidth
	}
	emptyLen := graphWidth - filledLen

	filledStr := theme.DiskBarFull.Render(strings.Repeat("█", filledLen))
	emptyStr := theme.DiskBarEmpty.Render(strings.Repeat("░", emptyLen))
	pctStr := fmt.Sprintf(" %2d%%", int(ratio*100))

	return "[" + filledStr + emptyStr + "]" + pctStr
}
