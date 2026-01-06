package theme

import "github.com/charmbracelet/lipgloss"

// Theme defines a complete color scheme for the TUI
type Theme struct {
	Name string

	// Background layers
	Crust    lipgloss.Color
	Mantle   lipgloss.Color
	Base     lipgloss.Color
	Surface0 lipgloss.Color
	Surface1 lipgloss.Color
	Surface2 lipgloss.Color

	// Overlay layers
	Overlay0 lipgloss.Color
	Overlay1 lipgloss.Color
	Overlay2 lipgloss.Color

	// Text hierarchy
	Text     lipgloss.Color
	Subtext1 lipgloss.Color
	Subtext0 lipgloss.Color

	// Accent colors
	Accent    lipgloss.Color
	Secondary lipgloss.Color
	Tertiary  lipgloss.Color

	// Status colors
	Running   lipgloss.Color
	Waiting   lipgloss.Color
	Approval  lipgloss.Color
	NeedsAttn lipgloss.Color
	Stale     lipgloss.Color
	Ended     lipgloss.Color

	// Provider colors
	Claude lipgloss.Color
	Codex  lipgloss.Color
}

// Mocha is the default dark theme (Catppuccin Mocha)
var Mocha = Theme{
	Name:      "mocha",
	Crust:     "#11111b",
	Mantle:    "#181825",
	Base:      "#1e1e2e",
	Surface0:  "#313244",
	Surface1:  "#45475a",
	Surface2:  "#585b70",
	Overlay0:  "#6c7086",
	Overlay1:  "#7f849c",
	Overlay2:  "#9399b2",
	Text:      "#cdd6f4",
	Subtext1:  "#bac2de",
	Subtext0:  "#a6adc8",
	Accent:    "#89b4fa", // Blue
	Secondary: "#cba6f7", // Mauve
	Tertiary:  "#94e2d5", // Teal
	Running:   "#a6e3a1", // Green
	Waiting:   "#89b4fa", // Blue
	Approval:  "#fab387", // Peach
	NeedsAttn: "#f38ba8", // Red
	Stale:     "#6c7086", // Overlay0
	Ended:     "#585b70", // Surface2
	Claude:    "#cba6f7", // Mauve
	Codex:     "#94e2d5", // Teal
}

// Frappe is a slightly lighter dark theme (Catppuccin Frappe)
var Frappe = Theme{
	Name:      "frappe",
	Crust:     "#232634",
	Mantle:    "#292c3c",
	Base:      "#303446",
	Surface0:  "#414559",
	Surface1:  "#51576d",
	Surface2:  "#626880",
	Overlay0:  "#737994",
	Overlay1:  "#838ba7",
	Overlay2:  "#949cbb",
	Text:      "#c6d0f5",
	Subtext1:  "#b5bfe2",
	Subtext0:  "#a5adce",
	Accent:    "#8caaee", // Blue
	Secondary: "#ca9ee6", // Mauve
	Tertiary:  "#81c8be", // Teal
	Running:   "#a6d189", // Green
	Waiting:   "#8caaee", // Blue
	Approval:  "#ef9f76", // Peach
	NeedsAttn: "#e78284", // Red
	Stale:     "#737994", // Overlay0
	Ended:     "#626880", // Surface2
	Claude:    "#ca9ee6", // Mauve
	Codex:     "#81c8be", // Teal
}

// Latte is the light theme (Catppuccin Latte)
var Latte = Theme{
	Name:      "latte",
	Crust:     "#dce0e8",
	Mantle:    "#e6e9ef",
	Base:      "#eff1f5",
	Surface0:  "#ccd0da",
	Surface1:  "#bcc0cc",
	Surface2:  "#acb0be",
	Overlay0:  "#9ca0b0",
	Overlay1:  "#8c8fa1",
	Overlay2:  "#7c7f93",
	Text:      "#4c4f69",
	Subtext1:  "#5c5f77",
	Subtext0:  "#6c6f85",
	Accent:    "#1e66f5", // Blue
	Secondary: "#8839ef", // Mauve
	Tertiary:  "#179299", // Teal
	Running:   "#40a02b", // Green
	Waiting:   "#1e66f5", // Blue
	Approval:  "#fe640b", // Peach
	NeedsAttn: "#d20f39", // Red
	Stale:     "#9ca0b0", // Overlay0
	Ended:     "#acb0be", // Surface2
	Claude:    "#8839ef", // Mauve
	Codex:     "#179299", // Teal
}

// Themes is the list of available themes
var Themes = []Theme{Mocha, Frappe, Latte}

// ThemeByName returns a theme by name, defaulting to Mocha
func ThemeByName(name string) Theme {
	for _, t := range Themes {
		if t.Name == name {
			return t
		}
	}
	return Mocha
}
