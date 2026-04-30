package ui

import "github.com/charmbracelet/lipgloss"

// ── Palette ───────────────────────────────────────────────────────────────────
// All colours are defined as typed constants so they can be referenced
// safely in both style definitions and inline Render calls.

const (
	clrOrange lipgloss.Color = "#FF5500" // primary accent — SoundCloud brand orange
	clrGreen  lipgloss.Color = "#04B575" // success state
	clrRed    lipgloss.Color = "#FF4E4E" // error state
	clrCyan   lipgloss.Color = "#00B4D8" // cached / informational state
	clrGray   lipgloss.Color = "#888888" // secondary text
	clrDim    lipgloss.Color = "#444444" // very subtle UI chrome
	clrBorder lipgloss.Color = "#333333" // inactive border
	clrBright lipgloss.Color = "#E0E0E0" // primary text
	clrNormal lipgloss.Color = "#BBBBBB" // secondary text
)

// ── Text styles ───────────────────────────────────────────────────────────────

var (
	titleStyle   = lipgloss.NewStyle().Foreground(clrOrange).Bold(true)
	successStyle = lipgloss.NewStyle().Foreground(clrGreen).Bold(true)
	errorStyle   = lipgloss.NewStyle().Foreground(clrRed).Bold(true)
	dimStyle     = lipgloss.NewStyle().Foreground(clrDim)
	normalStyle  = lipgloss.NewStyle().Foreground(clrNormal)
)

// ── Download status styles ────────────────────────────────────────────────────

var (
	colorDown   = lipgloss.NewStyle().Foreground(clrOrange) // actively downloading
	colorCached = lipgloss.NewStyle().Foreground(clrCyan)   // already downloaded
)

// ── URL input box ─────────────────────────────────────────────────────────────

var (
	inputLabelStyle = lipgloss.NewStyle().Foreground(clrNormal)
	inputBoxStyle   = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(clrOrange).
				Padding(0, 1)
)

// ── Download progress table ───────────────────────────────────────────────────

var (
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(clrGray)

	// tableBorderStyle is used when no download is in progress (idle / done).
	tableBorderStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(clrBorder).
				Padding(0, 1)

	// tableBorderActiveStyle highlights the table border while downloading.
	tableBorderActiveStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(clrOrange).
				Padding(0, 1)
)

// ── Progress bar ──────────────────────────────────────────────────────────────

var (
	barFilledStyle = lipgloss.NewStyle().Foreground(clrOrange) // downloaded portion
	barEmptyStyle  = lipgloss.NewStyle().Foreground(clrDim)    // remaining portion
	barPctStyle    = lipgloss.NewStyle().Foreground(clrGray)   // percentage label
)

// ── Help overlay ──────────────────────────────────────────────────────────────

var (
	helpOverlayStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(clrOrange).
				Padding(1, 4)
	helpTitleStyle = lipgloss.NewStyle().Foreground(clrOrange).Bold(true)
	helpKeyStyle   = lipgloss.NewStyle().Foreground(clrOrange).Bold(true)
	helpDescStyle  = lipgloss.NewStyle().Foreground(clrGray)
	helpGroupStyle = lipgloss.NewStyle().Foreground(clrDim)
)

// ── Footer / keybinding bar ───────────────────────────────────────────────────

var footerStyle = lipgloss.NewStyle().Foreground(clrDim)

// ── Separator line ────────────────────────────────────────────────────────────

var sepStyle = lipgloss.NewStyle().Foreground(clrDim)

// ── Done state ────────────────────────────────────────────────────────────────

var donePathStyle = lipgloss.NewStyle().Foreground(clrCyan)
