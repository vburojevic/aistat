package theme

import "github.com/charmbracelet/lipgloss"

// Theme defines the color scheme for the TUI (single dark theme)
type Theme struct {
	// Background layers
	Background lipgloss.Color
	Surface    lipgloss.Color
	Border     lipgloss.Color

	// Text hierarchy
	Text  lipgloss.Color
	Muted lipgloss.Color
	Faint lipgloss.Color

	// Status colors (3 states only)
	Active     lipgloss.Color // Running, healthy
	Idle       lipgloss.Color // Waiting
	NeedsInput lipgloss.Color // Approval/attention required

	// Provider colors
	Claude lipgloss.Color
	Codex  lipgloss.Color
}

// Default is the single dark theme
var Default = Theme{
	Background: ColorBackground,
	Surface:    ColorSurface,
	Border:     ColorBorder,
	Text:       ColorText,
	Muted:      ColorMuted,
	Faint:      ColorFaint,
	Active:     ColorActive,
	Idle:       ColorIdle,
	NeedsInput: ColorNeedsInput,
	Claude:     ColorClaude,
	Codex:      ColorCodex,
}

// Current returns the current theme (always Default since we only have one)
func Current() Theme {
	return Default
}
