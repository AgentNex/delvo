package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Theme holds the lipgloss styles for Delvo's TUI.
type Theme struct {
	BgColor          color.Color
	SurfaceColor     color.Color
	ActiveRowBgColor color.Color

	// Color styles (can be used directly with .Render())
	Cyan          lipgloss.Style
	Teal          lipgloss.Style
	Mint          lipgloss.Style
	SkyBlue       lipgloss.Style
	TextPrimary   lipgloss.Style
	TextMuted     lipgloss.Style
	TextDim       lipgloss.Style
	DangerRed     lipgloss.Style
	WarningYellow lipgloss.Style

	// Component Styles
	Title             lipgloss.Style
	Subtitle          lipgloss.Style
	PanelBorderActive lipgloss.Style
	PanelBorderDim    lipgloss.Style
	RowNormal         lipgloss.Style
	RowActive         lipgloss.Style
	DirName           lipgloss.Style
	FileName          lipgloss.Style
	SymlinkName       lipgloss.Style
	SelectedMark      lipgloss.Style
	UnselectedMark    lipgloss.Style
	MatchHighlight    lipgloss.Style
	Badge             lipgloss.Style
	StatusBar         lipgloss.Style
	HelpKey           lipgloss.Style
	HelpDesc          lipgloss.Style
	DialogBox         lipgloss.Style
	DialogTitle       lipgloss.Style
	DiskBarFull       lipgloss.Style
	DiskBarEmpty      lipgloss.Style
}

// NewDarkNavyTheme builds Delvo's signature dark-navy, cyan, and teal theme.
func NewDarkNavyTheme() Theme {
	bg := lipgloss.Color("#0b0f19")
	surface := lipgloss.Color("#111827")
	activeRowBg := lipgloss.Color("#152438")
	cyanColor := lipgloss.Color("#00f0ff")
	tealColor := lipgloss.Color("#2dd4bf")
	mintColor := lipgloss.Color("#34d399")
	skyColor := lipgloss.Color("#38bdf8")
	textPrimaryColor := lipgloss.Color("#e2e8f0")
	textMutedColor := lipgloss.Color("#64748b")
	textDimColor := lipgloss.Color("#334155")
	dangerColor := lipgloss.Color("#f87171")
	warningColor := lipgloss.Color("#fbbf24")

	return Theme{
		BgColor:          bg,
		SurfaceColor:     surface,
		ActiveRowBgColor: activeRowBg,

		Cyan:          lipgloss.NewStyle().Foreground(cyanColor),
		Teal:          lipgloss.NewStyle().Foreground(tealColor),
		Mint:          lipgloss.NewStyle().Foreground(mintColor),
		SkyBlue:       lipgloss.NewStyle().Foreground(skyColor),
		TextPrimary:   lipgloss.NewStyle().Foreground(textPrimaryColor),
		TextMuted:     lipgloss.NewStyle().Foreground(textMutedColor),
		TextDim:       lipgloss.NewStyle().Foreground(textDimColor),
		DangerRed:     lipgloss.NewStyle().Foreground(dangerColor),
		WarningYellow: lipgloss.NewStyle().Foreground(warningColor),

		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(cyanColor),

		Subtitle: lipgloss.NewStyle().
			Foreground(tealColor),

		PanelBorderActive: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(cyanColor),

		PanelBorderDim: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(textDimColor),

		RowNormal: lipgloss.NewStyle().
			Foreground(textPrimaryColor),

		RowActive: lipgloss.NewStyle().
			Bold(true).
			Foreground(cyanColor).
			Background(activeRowBg),

		DirName: lipgloss.NewStyle().
			Bold(true).
			Foreground(skyColor),

		FileName: lipgloss.NewStyle().
			Foreground(textPrimaryColor),

		SymlinkName: lipgloss.NewStyle().
			Italic(true).
			Foreground(tealColor),

		SelectedMark: lipgloss.NewStyle().
			Bold(true).
			Foreground(mintColor),

		UnselectedMark: lipgloss.NewStyle().
			Foreground(textDimColor),

		MatchHighlight: lipgloss.NewStyle().
			Bold(true).
			Foreground(mintColor),

		Badge: lipgloss.NewStyle().
			Bold(true).
			Foreground(surface).
			Background(tealColor).
			Padding(0, 1),

		StatusBar: lipgloss.NewStyle().
			Foreground(textMutedColor).
			Background(surface),

		HelpKey: lipgloss.NewStyle().
			Bold(true).
			Foreground(cyanColor),

		HelpDesc: lipgloss.NewStyle().
			Foreground(textMutedColor),

		DialogBox: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(cyanColor).
			Background(surface).
			Padding(1, 2),

		DialogTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(cyanColor),

		DiskBarFull: lipgloss.NewStyle().
			Foreground(tealColor),

		DiskBarEmpty: lipgloss.NewStyle().
			Foreground(textDimColor),
	}
}
