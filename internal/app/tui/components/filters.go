package components

import (
	"github.com/vburojevic/aistat/internal/app/tui/theme"
)

// RenderFilterBar renders the filter input bar
// Shows "/ " prompt followed by the current filter text
func RenderFilterBar(query string, active bool, styles theme.Styles) string {
	prompt := styles.FilterPrompt.Render("/ ")
	if query == "" {
		if active {
			return prompt + styles.Muted.Render("type to filter...")
		}
		return prompt + styles.Muted.Render("filter...")
	}
	return prompt + styles.FilterText.Render(query)
}
