package widgets

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/vburojevic/aistat/internal/app/tui/state"
	"github.com/vburojevic/aistat/internal/app/tui/theme"
)

// StatusIcon returns the icon for a status
func StatusIcon(s state.Status) string {
	switch s {
	case state.StatusRunning:
		return "●" // Solid circle - active
	case state.StatusWaiting:
		return "◐" // Half circle - idle
	case state.StatusApproval:
		return "◉" // Bullseye - needs attention
	case state.StatusNeedsAttn:
		return "◈" // Diamond - urgent
	case state.StatusStale:
		return "◌" // Dotted circle - fading
	case state.StatusEnded:
		return "◇" // Empty diamond - completed
	default:
		return "?"
	}
}

// StatusIconAlt returns an alternative icon set
func StatusIconAlt(s state.Status) string {
	switch s {
	case state.StatusRunning:
		return "▶" // Play
	case state.StatusWaiting:
		return "⏸" // Pause
	case state.StatusApproval:
		return "⚡" // Lightning
	case state.StatusNeedsAttn:
		return "⚠" // Warning
	case state.StatusStale:
		return "⏳" // Hourglass
	case state.StatusEnded:
		return "✓" // Check
	default:
		return "?"
	}
}

// StatusLabel returns the short label for a status
func StatusLabel(s state.Status) string {
	switch s {
	case state.StatusRunning:
		return "RUN"
	case state.StatusWaiting:
		return "IDL"
	case state.StatusApproval:
		return "APR"
	case state.StatusNeedsAttn:
		return "ATN"
	case state.StatusStale:
		return "OLD"
	case state.StatusEnded:
		return "END"
	default:
		return "???"
	}
}

// StatusBadge renders a full status badge with icon and label
func StatusBadge(s state.Status, styles theme.Styles) string {
	icon := StatusIcon(s)
	label := StatusLabel(s)

	var style lipgloss.Style
	switch s {
	case state.StatusRunning:
		style = styles.BadgeRun
	case state.StatusWaiting:
		style = styles.BadgeWait
	case state.StatusApproval:
		style = styles.BadgeAppr
	case state.StatusNeedsAttn:
		style = styles.BadgeAttn
	case state.StatusStale:
		style = styles.BadgeStale
	case state.StatusEnded:
		style = styles.BadgeEnded
	default:
		style = styles.BadgeWait
	}

	return style.Render(icon + " " + label)
}

// StatusBadgeCompact renders a compact status badge (icon only)
func StatusBadgeCompact(s state.Status, styles theme.Styles) string {
	icon := StatusIcon(s)

	var style lipgloss.Style
	switch s {
	case state.StatusRunning:
		style = styles.BadgeRun
	case state.StatusWaiting:
		style = styles.BadgeWait
	case state.StatusApproval:
		style = styles.BadgeAppr
	case state.StatusNeedsAttn:
		style = styles.BadgeAttn
	case state.StatusStale:
		style = styles.BadgeStale
	case state.StatusEnded:
		style = styles.BadgeEnded
	default:
		style = styles.BadgeWait
	}

	return style.Render(" " + icon + " ")
}

// StatusDot renders just a colored dot for a status
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

// ProviderIcon returns the icon for a provider
func ProviderIcon(p state.Provider) string {
	switch p {
	case state.ProviderClaude:
		return "◆" // Diamond for Claude
	case state.ProviderCodex:
		return "◇" // Empty diamond for Codex
	default:
		return "?"
	}
}

// ProviderIconEmoji returns the emoji icon for a provider
func ProviderIconEmoji(p state.Provider) string {
	switch p {
	case state.ProviderClaude:
		return "🧠"
	case state.ProviderCodex:
		return "⚡"
	default:
		return "?"
	}
}

// ProviderBadge renders a provider badge
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

// StatusChipHuman returns a human-readable status chip for urgent states
// Used in dashboard cards to show only action-needed statuses
func StatusChipHuman(s state.Status, count int, styles theme.Styles) string {
	switch s {
	case state.StatusApproval:
		return styles.BadgeAppr.Render(fmt.Sprintf("👋 %d NEEDS YOU", count))
	case state.StatusNeedsAttn:
		return styles.BadgeAttn.Render(fmt.Sprintf("🚨 %d URGENT", count))
	default:
		return ""
	}
}

// StatusBadgeHuman returns a human-readable status badge for table rows
// Used in session list for clear status indication
func StatusBadgeHuman(s state.Status, styles theme.Styles) string {
	switch s {
	case state.StatusRunning:
		return styles.BadgeRun.Render("▶ ACTIVE")
	case state.StatusWaiting:
		return styles.BadgeWait.Render("⏸ IDLE")
	case state.StatusApproval:
		return styles.BadgeAppr.Render("👋 NEEDS YOU")
	case state.StatusNeedsAttn:
		return styles.BadgeAttn.Render("🚨 URGENT")
	case state.StatusStale:
		return styles.BadgeStale.Render("💤 STALE")
	case state.StatusEnded:
		return styles.BadgeEnded.Render("✓ DONE")
	default:
		return styles.BadgeWait.Render("? UNKNOWN")
	}
}
