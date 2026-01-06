package widgets

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/vburojevic/aistat/internal/app/tui/state"
	"github.com/vburojevic/aistat/internal/app/tui/theme"
)

// StatusBadge renders a human-readable status badge (emoji + short text)
// This is the unified status display used everywhere in the UI
func StatusBadge(s state.Status, styles theme.Styles) string {
	switch s {
	case state.StatusRunning:
		return styles.BadgeRun.Render("▶ ACTIVE")
	case state.StatusWaiting:
		return styles.BadgeWait.Render("⏸ IDLE")
	case state.StatusApproval:
		return styles.BadgeAppr.Render("👋 NEED YOU")
	case state.StatusNeedsAttn:
		return styles.BadgeAttn.Render("🚨 URGENT")
	case state.StatusStale:
		return styles.BadgeStale.Render("💤 STALE")
	case state.StatusEnded:
		return styles.BadgeEnded.Render("✓ DONE")
	default:
		return styles.Muted.Render("? UNKNOWN")
	}
}

// StatusChip renders a status chip with count (for dashboard cards)
func StatusChip(s state.Status, count int, styles theme.Styles) string {
	switch s {
	case state.StatusApproval:
		return styles.BadgeAppr.Render(fmt.Sprintf("👋 %d NEED YOU", count))
	case state.StatusNeedsAttn:
		return styles.BadgeAttn.Render(fmt.Sprintf("🚨 %d URGENT", count))
	case state.StatusRunning:
		return styles.BadgeRun.Render(fmt.Sprintf("▶ %d active", count))
	case state.StatusWaiting:
		return styles.BadgeWait.Render(fmt.Sprintf("⏸ %d idle", count))
	case state.StatusStale:
		return styles.BadgeStale.Render(fmt.Sprintf("💤 %d stale", count))
	case state.StatusEnded:
		return styles.BadgeEnded.Render(fmt.Sprintf("✓ %d done", count))
	default:
		return ""
	}
}

// StatusDot renders just a colored dot for a status (minimal indicator)
func StatusDot(s state.Status, t theme.Theme) string {
	var color lipgloss.Color
	switch s {
	case state.StatusRunning:
		color = t.Running
	case state.StatusWaiting:
		color = t.Waiting
	case state.StatusApproval:
		color = t.Approval
	case state.StatusNeedsAttn:
		color = t.NeedsAttn
	case state.StatusStale:
		color = t.Stale
	case state.StatusEnded:
		color = t.Ended
	default:
		color = t.Overlay0
	}
	return lipgloss.NewStyle().Foreground(color).Render("●")
}

// ProviderIcon returns the text-based icon for a provider
func ProviderIcon(p state.Provider) string {
	switch p {
	case state.ProviderClaude:
		return "[C]"
	case state.ProviderCodex:
		return "[O]"
	default:
		return "[?]"
	}
}

// ProviderBadge renders a styled provider badge
func ProviderBadge(p state.Provider, styles theme.Styles) string {
	icon := ProviderIcon(p)
	var style lipgloss.Style
	switch p {
	case state.ProviderClaude:
		style = styles.ProviderClaude
	case state.ProviderCodex:
		style = styles.ProviderCodex
	default:
		style = styles.Muted
	}
	return style.Render(icon)
}
