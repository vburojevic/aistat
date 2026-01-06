package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/vburojevic/aistat/internal/app/tui/theme"
)

// RenderHeader renders the minimal header: title + urgent badge
func RenderHeader(needsInputCount int, styles theme.Styles, width int) string {
	title := styles.Title.Render("aistat")

	// Urgent badge (if any sessions need input)
	var badge string
	if needsInputCount > 0 {
		badge = styles.BadgeNeedsInput.Render(fmt.Sprintf("● %d need input", needsInputCount))
	}

	if badge == "" {
		return styles.Header.Width(width).Render(title)
	}

	// Layout: title on left, badge on right
	titleWidth := lipgloss.Width(title)
	badgeWidth := lipgloss.Width(badge)
	gap := width - titleWidth - badgeWidth - 4

	if gap < 1 {
		// Too narrow - just show title
		return styles.Header.Width(width).Render(title)
	}

	spacer := lipgloss.NewStyle().Width(gap).Render("")
	row := lipgloss.JoinHorizontal(lipgloss.Center, title, spacer, badge)
	return styles.Header.Width(width).Render(row)
}
