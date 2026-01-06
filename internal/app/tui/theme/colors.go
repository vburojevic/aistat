package theme

import "github.com/charmbracelet/lipgloss"

// Catppuccin Mocha - Base colors
var (
	// Background layers (darkest to lightest)
	ColorCrust    = lipgloss.Color("#11111b")
	ColorMantle   = lipgloss.Color("#181825")
	ColorBase     = lipgloss.Color("#1e1e2e")
	ColorSurface0 = lipgloss.Color("#313244")
	ColorSurface1 = lipgloss.Color("#45475a")
	ColorSurface2 = lipgloss.Color("#585b70")

	// Overlay layers
	ColorOverlay0 = lipgloss.Color("#6c7086")
	ColorOverlay1 = lipgloss.Color("#7f849c")
	ColorOverlay2 = lipgloss.Color("#9399b2")

	// Text hierarchy
	ColorText     = lipgloss.Color("#cdd6f4")
	ColorSubtext1 = lipgloss.Color("#bac2de")
	ColorSubtext0 = lipgloss.Color("#a6adc8")

	// Catppuccin accent colors
	ColorRosewater = lipgloss.Color("#f5e0dc")
	ColorFlamingo  = lipgloss.Color("#f2cdcd")
	ColorPink      = lipgloss.Color("#f5c2e7")
	ColorMauve     = lipgloss.Color("#cba6f7")
	ColorRed       = lipgloss.Color("#f38ba8")
	ColorMaroon    = lipgloss.Color("#eba0ac")
	ColorPeach     = lipgloss.Color("#fab387")
	ColorYellow    = lipgloss.Color("#f9e2af")
	ColorGreen     = lipgloss.Color("#a6e3a1")
	ColorTeal      = lipgloss.Color("#94e2d5")
	ColorSky       = lipgloss.Color("#89dceb")
	ColorSapphire  = lipgloss.Color("#74c7ec")
	ColorBlue      = lipgloss.Color("#89b4fa")
	ColorLavender  = lipgloss.Color("#b4befe")
)

// Semantic status colors
var (
	ColorRunning   = ColorGreen // Active, healthy
	ColorWaiting   = ColorBlue  // Idle but ready
	ColorApproval  = ColorPeach // Needs attention
	ColorNeedsAttn = ColorRed   // Urgent
	ColorStale     = ColorOverlay0
	ColorEnded     = ColorSurface2
)

// Provider colors
var (
	ColorClaude = ColorMauve // Purple for Claude
	ColorCodex  = ColorTeal  // Teal for Codex
)

// Border characters for rounded subtle borders (Lazygit style)
var BorderSubtle = lipgloss.Border{
	Top:         "─",
	Bottom:      "─",
	Left:        "│",
	Right:       "│",
	TopLeft:     "╭",
	TopRight:    "╮",
	BottomLeft:  "╰",
	BottomRight: "╯",
}

// Sharp borders for emphasis
var BorderSharp = lipgloss.Border{
	Top:         "─",
	Bottom:      "─",
	Left:        "│",
	Right:       "│",
	TopLeft:     "┌",
	TopRight:    "┐",
	BottomLeft:  "└",
	BottomRight: "┘",
}
