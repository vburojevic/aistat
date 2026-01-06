package theme

import "github.com/charmbracelet/lipgloss"

// Muted dark palette - professional and readable
var (
	// Background layers
	ColorBackground = lipgloss.Color("#1e1e2e") // Dark charcoal
	ColorSurface    = lipgloss.Color("#2a2a3c") // Slightly lighter
	ColorBorder     = lipgloss.Color("#3a3a4c") // Subtle border

	// Text hierarchy
	ColorText   = lipgloss.Color("#cdd6f4") // Off-white primary
	ColorMuted  = lipgloss.Color("#6c7086") // Gray for secondary/dimmed
	ColorFaint  = lipgloss.Color("#45475a") // Very faint for dividers

	// Status colors (muted pastels)
	ColorActive     = lipgloss.Color("#a8c97f") // Sage green - running
	ColorIdle       = lipgloss.Color("#8ba4b4") // Slate blue - waiting
	ColorNeedsInput = lipgloss.Color("#e9b59f") // Peach - needs attention

	// Provider colors (subtle)
	ColorClaude = lipgloss.Color("#b4a7d6") // Muted lavender
	ColorCodex  = lipgloss.Color("#8fb8a8") // Muted teal
)

// Border characters for rounded subtle borders
var BorderRounded = lipgloss.Border{
	Top:         "─",
	Bottom:      "─",
	Left:        "│",
	Right:       "│",
	TopLeft:     "╭",
	TopRight:    "╮",
	BottomLeft:  "╰",
	BottomRight: "╯",
}
