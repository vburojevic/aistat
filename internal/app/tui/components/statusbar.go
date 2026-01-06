package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vburojevic/aistat/internal/app/tui/theme"
)

// Shortcut represents a keyboard shortcut
type Shortcut struct {
	Key  string
	Desc string
}

// MainShortcuts - the ~10 essential shortcuts for normal mode
var MainShortcuts = []Shortcut{
	{"j/k", "navigate"},
	{"/", "filter"},
	{"b", "bookmark"},
	{"?", "help"},
	{"q", "quit"},
}

// FilterShortcuts - shown when filter is active
var FilterShortcuts = []Shortcut{
	{"esc", "clear"},
	{"enter", "apply"},
}

// RenderFooter renders the context-aware footer with shortcuts and icon legend
func RenderFooter(filterActive bool, refreshing bool, spinnerFrame int, spinnerFrames []string, styles theme.Styles, width int) string {
	var shortcuts []Shortcut
	if filterActive {
		shortcuts = FilterShortcuts
	} else {
		shortcuts = MainShortcuts
	}

	// Build shortcut string with fixed-width key column
	keyStyle := lipgloss.NewStyle().
		Foreground(styles.HelpKey.GetForeground()).
		Width(6).
		Align(lipgloss.Right)

	descStyle := lipgloss.NewStyle().
		Foreground(styles.Muted.GetForeground())

	var parts []string
	for _, s := range shortcuts {
		part := keyStyle.Render(s.Key) + " " + descStyle.Render(s.Desc)
		parts = append(parts, part)
	}
	shortcutStr := strings.Join(parts, "   ")

	// Icon legend: ▶run ⏸idle ⚡input
	legend := styles.DotActive.Render("▶") + descStyle.Render("run") + " " +
		styles.DotIdle.Render("⏸") + descStyle.Render("idle") + " " +
		styles.DotNeedsInput.Render("⚡") + descStyle.Render("input")

	// Add animated spinner indicator
	var indicator string
	if refreshing && len(spinnerFrames) > 0 {
		frame := spinnerFrame % len(spinnerFrames)
		indicator = " " + styles.DotActive.Render(spinnerFrames[frame])
	}

	// Calculate remaining space and right-align legend + indicator
	leftContent := shortcutStr
	rightContent := legend + indicator

	contentWidth := lipgloss.Width(leftContent) + lipgloss.Width(rightContent)
	if contentWidth >= width-4 {
		return styles.Footer.Width(width).Render(shortcutStr)
	}

	gap := width - contentWidth - 4
	spacer := strings.Repeat(" ", gap)
	row := leftContent + spacer + rightContent

	return styles.Footer.Width(width).Render(row)
}
