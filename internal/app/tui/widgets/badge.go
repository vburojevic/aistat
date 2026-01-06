package widgets

import (
	"github.com/vburojevic/aistat/internal/app/tui/state"
	"github.com/vburojevic/aistat/internal/app/tui/theme"
)

// UIStatus represents the simplified 3-state UI status
type UIStatus int

const (
	UIStatusActive UIStatus = iota
	UIStatusIdle
	UIStatusNeedsInput
)

// ToUIStatus converts backend status to simplified UI status
func ToUIStatus(s state.Status) UIStatus {
	switch s {
	case state.StatusApproval, state.StatusNeedsAttn:
		return UIStatusNeedsInput
	case state.StatusRunning:
		return UIStatusActive
	default:
		// StatusWaiting, StatusStale, StatusEnded, StatusUnknown → Idle
		return UIStatusIdle
	}
}

// IsEnded returns true if the status represents an ended session
func IsEnded(s state.Status) bool {
	return s == state.StatusEnded || s == state.StatusStale
}

// StatusIcon renders a colored icon for the UI status
// Icons: ▶ running, ⏸ idle, ⚡ needs input, ⏹ ended
func StatusIcon(s state.Status, styles theme.Styles) string {
	if IsEnded(s) {
		return styles.Muted.Render("⏹")
	}

	ui := ToUIStatus(s)
	switch ui {
	case UIStatusNeedsInput:
		return styles.DotNeedsInput.Render("⚡")
	case UIStatusActive:
		return styles.DotActive.Render("▶")
	default:
		return styles.DotIdle.Render("⏸")
	}
}

// StatusIconDimmed renders a dimmed icon for old/ended sessions
func StatusIconDimmed(styles theme.Styles) string {
	return styles.Muted.Render("⏹")
}

// StatusDot renders a colored dot for the UI status (legacy, use StatusIcon)
func StatusDot(s state.Status, styles theme.Styles) string {
	return StatusIcon(s, styles)
}

// StatusDotDimmed renders a dimmed dot for ended sessions (legacy, use StatusIconDimmed)
func StatusDotDimmed(styles theme.Styles) string {
	return StatusIconDimmed(styles)
}

// ProviderLetter returns a single letter for the provider (subtle indicator)
func ProviderLetter(p state.Provider) string {
	switch p {
	case state.ProviderClaude:
		return "C"
	case state.ProviderCodex:
		return "O"
	default:
		return "?"
	}
}

// ProviderLetterStyled renders the provider letter with color
func ProviderLetterStyled(p state.Provider, styles theme.Styles) string {
	letter := ProviderLetter(p)
	switch p {
	case state.ProviderClaude:
		return styles.ProviderClaude.Render(letter)
	case state.ProviderCodex:
		return styles.ProviderCodex.Render(letter)
	default:
		return styles.Muted.Render(letter)
	}
}

// NeedsInputCount returns count of sessions needing input
func NeedsInputCount(sessions []state.SessionView) int {
	count := 0
	for _, s := range sessions {
		if ToUIStatus(s.Status) == UIStatusNeedsInput {
			count++
		}
	}
	return count
}
