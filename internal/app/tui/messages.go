package tui

import (
	"time"

	"github.com/vburojevic/aistat/internal/app/tui/state"
)

// SessionsMsg is sent when sessions are fetched
type SessionsMsg struct {
	Sessions []state.SessionView
	Err      error
}

// TickMsg is sent on each refresh tick
type TickMsg time.Time

// SpinnerTickMsg is sent to animate the spinner
type SpinnerTickMsg struct{}

// SpinnerFrames are braille spinner characters
var SpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
