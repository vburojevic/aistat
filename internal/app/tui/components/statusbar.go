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
	{"enter", "select"},
	{"/", "filter"},
	{"?", "help"},
	{"q", "quit"},
}

// FilterShortcuts - shown when filter is active
var FilterShortcuts = []Shortcut{
	{"esc", "clear"},
	{"enter", "apply"},
}

// RenderFooter renders the context-aware footer with shortcuts
func RenderFooter(filterActive bool, refreshing bool, styles theme.Styles, width int) string {
	var shortcuts []Shortcut
	if filterActive {
		shortcuts = FilterShortcuts
	} else {
		shortcuts = MainShortcuts
	}

	// Build shortcut string
	var parts []string
	for _, s := range shortcuts {
		parts = append(parts, styles.HelpKey.Render(s.Key)+" "+s.Desc)
	}
	shortcutStr := strings.Join(parts, "  ")

	// Add refresh indicator on right
	var indicator string
	if refreshing {
		indicator = styles.Muted.Render("↻")
	}

	if indicator == "" {
		return styles.Footer.Width(width).Render(shortcutStr)
	}

	// Layout with indicator on right
	scWidth := lipgloss.Width(shortcutStr)
	indWidth := lipgloss.Width(indicator)
	gap := width - scWidth - indWidth - 4

	if gap < 1 {
		return styles.Footer.Width(width).Render(shortcutStr)
	}

	spacer := lipgloss.NewStyle().Width(gap).Render("")
	row := lipgloss.JoinHorizontal(lipgloss.Center, shortcutStr, spacer, indicator)
	return styles.Footer.Width(width).Render(row)
}
