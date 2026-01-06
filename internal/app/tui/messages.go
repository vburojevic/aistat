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

// AnimMsg is sent for layout animations
type AnimMsg time.Time

// ViewMode represents the current view mode
type ViewMode int

const (
	ViewDashboard ViewMode = iota
	ViewSessions
	ViewProjects
	ViewHelp
)

// DetailMode represents the detail pane mode
type DetailMode int

const (
	DetailSplit DetailMode = iota
	DetailFull
)

// ColumnMode represents the table column mode
type ColumnMode string

const (
	ColumnModeFull    ColumnMode = "full"
	ColumnModeCompact ColumnMode = "compact"
	ColumnModeUltra   ColumnMode = "ultra"
	ColumnModeCard    ColumnMode = "card"
)

// Layout constants
const (
	SplitMinWidth     = 120
	SplitMinListWidth = 60
	SidebarMaxWidth   = 28
	SplitGap          = 1
	AnimStep          = 4
	AnimInterval      = 30 * time.Millisecond
)
