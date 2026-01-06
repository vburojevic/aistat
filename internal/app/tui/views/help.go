package views

import (
	"strings"

	"github.com/vburojevic/aistat/internal/app/tui/theme"
)

// RenderHelpOverlay renders the help overlay modal
func RenderHelpOverlay(styles theme.Styles) string {
	lines := []string{
		styles.HelpTitle.Render("Keyboard Shortcuts"),
		"",
	}

	// Navigation
	shortcuts := []struct {
		key  string
		desc string
	}{
		{"j / ↓", "Move down"},
		{"k / ↑", "Move up"},
		{"enter", "Select session"},
		{"/", "Filter sessions"},
		{"esc", "Clear filter"},
		{"?", "Toggle help"},
		{"q", "Quit"},
		{"", ""},
		{"b", "Toggle bookmark ★"},
		{"y", "Copy session ID"},
		{"r", "Refresh now"},
		{"a", "Toggle show all"},
	}

	for _, s := range shortcuts {
		if s.key == "" {
			lines = append(lines, "")
			continue
		}
		lines = append(lines, styles.HelpKey.Render(s.key)+styles.HelpDesc.Render(s.desc))
	}

	lines = append(lines, "")
	lines = append(lines, styles.Muted.Render("Press any key to close"))

	return styles.HelpOverlay.Render(strings.Join(lines, "\n"))
}
