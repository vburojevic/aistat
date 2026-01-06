package components

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/vburojevic/aistat/internal/app/tui/theme"
)

// HeaderConfig holds header configuration
type HeaderConfig struct {
	RefreshInterval time.Duration
	ActiveWindow    time.Duration
	Redact          bool
	ThemeName       string
	UrgentCount     int // Count of sessions needing input (approval + attention)
}

// RenderHeader renders the application header with urgent notification
func RenderHeader(styles theme.Styles, cfg HeaderConfig, width int) string {
	title := styles.Title.Render("◆ aistat")

	// Urgent badge (if any sessions need input)
	var urgentBadge string
	if cfg.UrgentCount > 0 {
		urgentBadge = styles.BadgeAttn.Render(fmt.Sprintf("🚨 %d need input", cfg.UrgentCount))
	}

	// Meta info - simplified
	meta := styles.Muted.Render(fmt.Sprintf("↻ %s • %s", cfg.RefreshInterval, cfg.ThemeName))

	// Calculate spacing
	titleWidth := lipgloss.Width(title)
	urgentWidth := lipgloss.Width(urgentBadge)
	metaWidth := lipgloss.Width(meta)
	totalContent := titleWidth + urgentWidth + metaWidth
	if urgentWidth > 0 {
		totalContent += 4 // Extra spacing around urgent badge
	}
	gap := width - totalContent - 4

	if gap < 1 {
		// Narrow mode - stack vertically
		parts := []string{title}
		if urgentBadge != "" {
			parts = append(parts, urgentBadge)
		}
		parts = append(parts, meta)
		return lipgloss.JoinVertical(lipgloss.Left, parts...)
	}

	// Wide mode - side by side with urgent badge prominent
	var parts []string
	parts = append(parts, title)
	if urgentBadge != "" {
		parts = append(parts, "  "+urgentBadge)
	}
	spacer := lipgloss.NewStyle().Width(gap).Render("")
	parts = append(parts, spacer, meta)
	return lipgloss.JoinHorizontal(lipgloss.Center, parts...)
}

// RenderTitleOnly renders just the title
func RenderTitleOnly(styles theme.Styles) string {
	return styles.Title.Render("◆ aistat")
}

// RenderBanner renders the onboarding banner
func RenderBanner(styles theme.Styles) string {
	return styles.PillActive.Render("Press ? for help • p projects • / search • : commands")
}
