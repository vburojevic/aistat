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

// StatusDot renders a colored dot for the UI status
func StatusDot(s state.Status, styles theme.Styles) string {
	ui := ToUIStatus(s)
	switch ui {
	case UIStatusNeedsInput:
		return styles.DotNeedsInput.Render("●")
	case UIStatusActive:
		return styles.DotActive.Render("●")
	default:
		return styles.DotIdle.Render("●")
	}
}

// StatusDotDimmed renders a dimmed dot for ended sessions
func StatusDotDimmed(styles theme.Styles) string {
	return styles.Muted.Render("●")
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
