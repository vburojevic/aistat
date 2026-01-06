package theme

import "github.com/charmbracelet/lipgloss"

// Theme defines the color scheme for the TUI (single dark theme)
type Theme struct {
	// Background layers
	Background lipgloss.Color
	Surface    lipgloss.Color
	Highlight  lipgloss.Color
	Border     lipgloss.Color

	// Text hierarchy
	Text  lipgloss.Color
	Dim80 lipgloss.Color // 1-6h age
	Dim60 lipgloss.Color // 6-24h age
	Muted lipgloss.Color
	Dim40 lipgloss.Color // >24h age
	Faint lipgloss.Color

	// Status colors (3 states only)
	Active     lipgloss.Color // Running, healthy
	Idle       lipgloss.Color // Waiting
	NeedsInput lipgloss.Color // Approval/attention required
	Error      lipgloss.Color // Error state

	// Provider colors
	Claude lipgloss.Color
	Codex  lipgloss.Color

	// Model colors
	ModelOpus   lipgloss.Color
	ModelSonnet lipgloss.Color
	ModelHaiku  lipgloss.Color
}

// Default is the single dark theme
var Default = Theme{
	Background: ColorBackground,
	Surface:    ColorSurface,
	Highlight:  ColorHighlight,
	Border:     ColorBorder,
	Text:       ColorText,
	Dim80:      ColorDim80,
	Dim60:      ColorDim60,
	Muted:      ColorMuted,
	Dim40:      ColorDim40,
	Faint:      ColorFaint,
	Active:     ColorActive,
	Idle:       ColorIdle,
	NeedsInput: ColorNeedsInput,
	Error:      ColorError,
	Claude:     ColorClaude,
	Codex:      ColorCodex,

	ModelOpus:   ColorModelOpus,
	ModelSonnet: ColorModelSonnet,
	ModelHaiku:  ColorModelHaiku,
}

// Current returns the current theme (always Default since we only have one)
func Current() Theme {
	return Default
}
