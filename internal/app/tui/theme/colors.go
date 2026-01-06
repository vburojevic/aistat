package theme

import "github.com/charmbracelet/lipgloss"

// Muted dark palette - professional and readable
var (
	// Background layers
	ColorBackground = lipgloss.Color("#1e1e2e") // Dark charcoal
	ColorSurface    = lipgloss.Color("#2a2a3c") // Slightly lighter
	ColorHighlight  = lipgloss.Color("#363648") // Subtle selection highlight
	ColorBorder     = lipgloss.Color("#3a3a4c") // Subtle border

	// Text hierarchy
	ColorText   = lipgloss.Color("#cdd6f4") // Off-white primary
	ColorDim80  = lipgloss.Color("#a9b1c6") // Slightly dimmed (1-6h age)
	ColorDim60  = lipgloss.Color("#8a93a8") // More dimmed (6-24h age)
	ColorMuted  = lipgloss.Color("#6c7086") // Gray for secondary/dimmed
	ColorDim40  = lipgloss.Color("#585b6b") // Very dimmed (>24h age)
	ColorFaint  = lipgloss.Color("#45475a") // Very faint for dividers

	// Status colors (muted pastels)
	ColorActive     = lipgloss.Color("#a8c97f") // Sage green - running
	ColorIdle       = lipgloss.Color("#8ba4b4") // Slate blue - waiting
	ColorNeedsInput = lipgloss.Color("#e9b59f") // Peach - needs attention
	ColorError      = lipgloss.Color("#f38ba8") // Soft red - errors

	// Provider colors (subtle)
	ColorClaude = lipgloss.Color("#b4a7d6") // Muted lavender
	ColorCodex  = lipgloss.Color("#8fb8a8") // Muted teal

	// Model colors
	ColorModelOpus   = lipgloss.Color("#b4a7d6") // Purple - matches Claude
	ColorModelSonnet = lipgloss.Color("#89b4fa") // Blue
	ColorModelHaiku  = lipgloss.Color("#8fb8a8") // Teal - matches Codex
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
