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
}

// RenderHeader renders the application header
func RenderHeader(styles theme.Styles, cfg HeaderConfig, width int) string {
	title := styles.Title.Render("◆ aistat")

	meta := styles.Muted.Render(fmt.Sprintf("refresh %s • window %s • redact %v • theme %s",
		cfg.RefreshInterval, cfg.ActiveWindow, cfg.Redact, cfg.ThemeName))

	// Calculate spacing
	titleWidth := lipgloss.Width(title)
	metaWidth := lipgloss.Width(meta)
	gap := width - titleWidth - metaWidth - 4

	if gap < 1 {
		// Narrow mode - stack vertically
		return lipgloss.JoinVertical(lipgloss.Left, title, meta)
	}

	// Wide mode - side by side
	spacer := lipgloss.NewStyle().Width(gap).Render("")
	return lipgloss.JoinHorizontal(lipgloss.Center, title, spacer, meta)
}

// RenderTitleOnly renders just the title
func RenderTitleOnly(styles theme.Styles) string {
	return styles.Title.Render("◆ aistat")
}

// RenderBanner renders the onboarding banner
func RenderBanner(styles theme.Styles) string {
	return styles.PillActive.Render("Press ? for help • p projects • / search • : commands")
}
