package tests

import (
	"strings"
	"testing"

	"github.com/AgentNex/delvo/internal/ui"
	"github.com/charmbracelet/x/ansi"
)

func TestAnsiSafeTruncationAndPadding(t *testing.T) {
	theme := ui.NewDarkNavyTheme()

	// 1. Plain string truncation
	plain := "Hello World! This is a long path description"
	truncPlain := ui.TruncateString(plain, 10)
	if ansi.StringWidth(truncPlain) > 10 {
		t.Errorf("expected visual width <= 10, got %d for %q", ansi.StringWidth(truncPlain), truncPlain)
	}
	if !strings.HasSuffix(truncPlain, "…") {
		t.Errorf("expected truncated string to end with …, got %q", truncPlain)
	}

	// 2. ANSI-styled string truncation
	// Styled string contains dozens of escape bytes
	styled := theme.Title.Render("DELVO CLI") + " " + theme.Subtitle.Render("Terminal File Manager")
	truncStyled := ui.TruncateString(styled, 15)
	if ansi.StringWidth(truncStyled) > 15 {
		t.Errorf("expected visual width <= 15, got %d for %q", ansi.StringWidth(truncStyled), truncStyled)
	}
	// Verify no broken escape sequence
	if strings.Contains(truncStyled, "\x1b[") && !strings.Contains(truncStyled, "m") {
		t.Errorf("unterminated escape sequence in: %q", truncStyled)
	}

	// 3. ANSI-styled string padding
	shortStyled := theme.Cyan.Render("Test") // visual width = 4
	padded := ui.PadRight(shortStyled, 10)
	if ansi.StringWidth(padded) != 10 {
		t.Errorf("expected visual width exactly 10, got %d for %q", ansi.StringWidth(padded), padded)
	}

	// 4. Unicode & Emoji handling
	emojiStr := "📂 Folder with 🚀 rocket"
	truncEmoji := ui.TruncateString(emojiStr, 12)
	if ansi.StringWidth(truncEmoji) > 12 {
		t.Errorf("expected visual width <= 12, got %d for %q", ansi.StringWidth(truncEmoji), truncEmoji)
	}
}

func TestRenderDiskBar(t *testing.T) {
	theme := ui.NewDarkNavyTheme()

	// 50% usage
	bar := ui.RenderDiskBar(500, 1000, 16, theme)
	if !strings.Contains(bar, "50%") {
		t.Errorf("expected 50%% in disk bar, got %s", bar)
	}
	if !strings.HasPrefix(bar, "[") {
		t.Errorf("expected bar to start with [, got %s", bar)
	}

	// 0% usage
	barZero := ui.RenderDiskBar(0, 1000, 16, theme)
	if !strings.Contains(barZero, "0%") {
		t.Errorf("expected 0%% in disk bar, got %s", barZero)
	}

	// 100% usage
	barFull := ui.RenderDiskBar(1000, 1000, 16, theme)
	if !strings.Contains(barFull, "100%") {
		t.Errorf("expected 100%% in disk bar, got %s", barFull)
	}
}

func TestComputeLayoutResponsive(t *testing.T) {
	// Narrow portrait phone screen (e.g. 50 cols x 24 rows)
	geomPhone := ui.ComputeLayout(50, 24)
	if !geomPhone.IsNarrow {
		t.Errorf("expected 50 cols to be narrow")
	}
	if geomPhone.DetailsWidth != 0 {
		t.Errorf("expected 0 details width on narrow screen, got %d", geomPhone.DetailsWidth)
	}
	if geomPhone.ListWidth <= 0 || geomPhone.ListWidth > 50 {
		t.Errorf("invalid list width on phone: %d", geomPhone.ListWidth)
	}

	// Wide landscape/desktop terminal (e.g. 120 cols x 40 rows)
	geomWide := ui.ComputeLayout(120, 40)
	if geomWide.IsNarrow {
		t.Errorf("expected 120 cols to be wide")
	}
	if geomWide.DetailsWidth < 24 {
		t.Errorf("expected details width >= 24, got %d", geomWide.DetailsWidth)
	}
	if geomWide.ListWidth+geomWide.DetailsWidth+4 > 120 {
		t.Errorf("split width exceeds total width: %d + %d + 4 > 120", geomWide.ListWidth, geomWide.DetailsWidth)
	}
}
