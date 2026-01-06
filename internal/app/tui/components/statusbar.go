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

// ShortcutSets for different contexts
var (
	ShortcutsDashboard = []Shortcut{
		{"tab", "list"},
		{"enter", "focus"},
		{"/", "search"},
		{"p", "projects"},
		{"?", "help"},
		{"q", "quit"},
	}

	ShortcutsSessionList = []Shortcut{
		{"/", "filter"},
		{":", "palette"},
		{"tab", "dashboard"},
		{"p", "projects"},
		{"r", "refresh"},
		{"?", "help"},
		{"q", "quit"},
		{"s", "sort"},
		{"g", "group"},
		{"v", "view"},
		{"d", "detail"},
		{"b", "sidebar"},
		{"m", "last-msg"},
		{"c", "redact"},
		{"space", "select"},
		{"P", "pin"},
		{"y", "copy ids"},
		{"D", "copy detail"},
		{"o", "open"},
		{"a", "jump approval"},
		{"u", "jump running"},
		{"1/2", "providers"},
		{"R/W/E/S/Z/N", "status"},
		{"t", "theme"},
		{"A", "access"},
	}

	ShortcutsProjects = []Shortcut{
		{"enter", "focus"},
		{"space", "toggle"},
		{"a", "clear"},
		{"j/k", "nav"},
		{"esc", "close"},
	}

	ShortcutsHelp = []Shortcut{
		{"?", "close"},
		{"esc", "close"},
	}
)

// RenderShortcutBar renders the shortcut bar at the bottom
func RenderShortcutBar(shortcuts []Shortcut, styles theme.Styles, width int) string {
	sep := "  •  "
	var lines []string
	line := ""

	for _, s := range shortcuts {
		entry := styles.Accent.Render(s.Key) + " " + s.Desc

		if line == "" {
			line = entry
			continue
		}

		// Check if adding this entry would exceed width
		testLine := line + sep + entry
		if lipgloss.Width(testLine) <= width-4 {
			line = testLine
			continue
		}

		// Start new line
		lines = append(lines, line)
		line = entry
	}

	if line != "" {
		lines = append(lines, line)
	}

	// Style each line
	result := make([]string, len(lines))
	for i, l := range lines {
		result[i] = styles.ShortcutBar.Width(width).Render(l)
	}

	return strings.Join(result, "\n")
}

// RenderContextBar renders context-specific actions
func RenderContextBar(context string, selectedCount int, styles theme.Styles) string {
	switch context {
	case "selected":
		return styles.Muted.Render(
			lipgloss.NewStyle().Render(
				strings.Join([]string{
					styles.Accent.Render("Selected ") + string(rune('0'+selectedCount)),
					"y copy ids",
					"D copy detail",
					"o open",
				}, " • "),
			),
		)
	case "palette":
		return styles.Muted.Render("Palette: enter to run • esc to cancel")
	case "projects":
		return styles.Muted.Render("Projects: enter/space toggle • a clear • esc close")
	case "dashboard":
		return styles.Muted.Render("Dashboard: enter focus • space toggle • a clear • tab back")
	default:
		return ""
	}
}
