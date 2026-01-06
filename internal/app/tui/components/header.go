package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/vburojevic/aistat/internal/app/tui/state"
	"github.com/vburojevic/aistat/internal/app/tui/theme"
	"github.com/vburojevic/aistat/internal/app/tui/widgets"
)

// StatusCounts holds counts for each status category
type StatusCounts struct {
	Running    int
	Idle       int
	NeedsInput int
}

// CountStatuses counts sessions by status category
func CountStatuses(sessions []state.SessionView) StatusCounts {
	var c StatusCounts
	for _, s := range sessions {
		switch widgets.ToUIStatus(s.Status) {
		case widgets.UIStatusActive:
			c.Running++
		case widgets.UIStatusNeedsInput:
			c.NeedsInput++
		default:
			c.Idle++
		}
	}
	return c
}

// RenderHeader renders the header: title + status counts + urgent badge
func RenderHeader(sessions []state.SessionView, styles theme.Styles, width int) string {
	title := styles.Title.Render("aistat")
	counts := CountStatuses(sessions)

	// Status counts with icons: ▶3 ⏸2 ⚡1
	var statusParts []string
	if counts.Running > 0 {
		statusParts = append(statusParts, styles.DotActive.Render("▶")+fmt.Sprintf("%d", counts.Running))
	}
	if counts.Idle > 0 {
		statusParts = append(statusParts, styles.DotIdle.Render("⏸")+fmt.Sprintf("%d", counts.Idle))
	}
	if counts.NeedsInput > 0 {
		statusParts = append(statusParts, styles.DotNeedsInput.Render("⚡")+fmt.Sprintf("%d", counts.NeedsInput))
	}

	statusStr := ""
	for i, p := range statusParts {
		if i > 0 {
			statusStr += " "
		}
		statusStr += p
	}

	// Urgent badge (if any sessions need input)
	var badge string
	if counts.NeedsInput > 0 {
		badge = styles.BadgeNeedsInput.Render(fmt.Sprintf("● %d need input", counts.NeedsInput))
	}

	// Layout: title | status counts | badge
	titleWidth := lipgloss.Width(title)
	statusWidth := lipgloss.Width(statusStr)
	badgeWidth := lipgloss.Width(badge)

	totalContent := titleWidth + statusWidth + badgeWidth
	gap := width - totalContent - 6 // 6 for padding

	if gap < 2 {
		// Too narrow - just show title
		return styles.Header.Width(width).Render(title)
	}

	// Distribute gaps
	gap1 := gap / 2
	gap2 := gap - gap1

	spacer1 := lipgloss.NewStyle().Width(gap1).Render("")
	spacer2 := lipgloss.NewStyle().Width(gap2).Render("")

	row := lipgloss.JoinHorizontal(lipgloss.Center, title, spacer1, statusStr, spacer2, badge)
	return styles.Header.Width(width).Render(row)
}
