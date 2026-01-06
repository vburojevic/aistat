package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap holds all key bindings for the TUI
type KeyMap struct {
	// Navigation
	UpDown key.Binding
	Quit   key.Binding

	// Search and Filter
	Filter  key.Binding
	Palette key.Binding

	// Views
	Help      key.Binding
	Projects  key.Binding
	Dashboard key.Binding

	// Display toggles
	ToggleRedact  key.Binding
	ToggleSort    key.Binding
	ToggleGroup   key.Binding
	ToggleView    key.Binding
	ToggleLast    key.Binding
	ToggleSidebar key.Binding
	ToggleDetail  key.Binding
	ToggleTheme   key.Binding
	ToggleAccess  key.Binding

	// Actions
	Refresh      key.Binding
	CopyID       key.Binding
	CopyDetail   key.Binding
	OpenFile     key.Binding
	TogglePin    key.Binding
	ToggleSelect key.Binding

	// Jump commands
	JumpApproval key.Binding
	JumpRunning  key.Binding

	// Provider filters
	ToggleClaude key.Binding
	ToggleCodex  key.Binding

	// Status filters
	ToggleRun   key.Binding
	ToggleWait  key.Binding
	ToggleAppr  key.Binding
	ToggleStale key.Binding
	ToggleEnded key.Binding
	ToggleAttn  key.Binding
}

// DefaultKeyMap returns the default key bindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		UpDown:        key.NewBinding(key.WithKeys("up", "down", "j", "k"), key.WithHelp("↑/↓", "move")),
		Quit:          key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Refresh:       key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Filter:        key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Palette:       key.NewBinding(key.WithKeys(":"), key.WithHelp(":", "palette")),
		Help:          key.NewBinding(key.WithKeys("?", "h"), key.WithHelp("?", "help")),
		Projects:      key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "projects")),
		Dashboard:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "dashboard")),
		ToggleRedact:  key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "redact")),
		CopyID:        key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy id(s)")),
		CopyDetail:    key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "copy detail")),
		OpenFile:      key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open log")),
		ToggleSort:    key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort")),
		ToggleGroup:   key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "group")),
		ToggleView:    key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "view")),
		ToggleLast:    key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "last msg")),
		TogglePin:     key.NewBinding(key.WithKeys("P"), key.WithHelp("P", "pin")),
		ToggleSelect:  key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "select")),
		JumpApproval:  key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "jump approval")),
		JumpRunning:   key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "jump running")),
		ToggleTheme:   key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "theme")),
		ToggleAccess:  key.NewBinding(key.WithKeys("A"), key.WithHelp("A", "access")),
		ToggleSidebar: key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "sidebar")),
		ToggleDetail:  key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "detail")),
		ToggleClaude:  key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "claude")),
		ToggleCodex:   key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "codex")),
		ToggleRun:     key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "running")),
		ToggleWait:    key.NewBinding(key.WithKeys("W"), key.WithHelp("W", "waiting")),
		ToggleAppr:    key.NewBinding(key.WithKeys("E"), key.WithHelp("E", "approval")),
		ToggleStale:   key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "stale")),
		ToggleEnded:   key.NewBinding(key.WithKeys("Z"), key.WithHelp("Z", "ended")),
		ToggleAttn:    key.NewBinding(key.WithKeys("N"), key.WithHelp("N", "attn")),
	}
}

// ShortHelp returns the short help
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.Refresh, k.Filter, k.Palette, k.Projects, k.Dashboard, k.Help}
}

// FullHelp returns the full help
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Quit, k.Refresh, k.Filter, k.Palette, k.Projects},
		{k.ToggleSort, k.ToggleGroup, k.ToggleView, k.ToggleLast, k.ToggleDetail},
		{k.CopyID, k.CopyDetail, k.OpenFile, k.TogglePin, k.ToggleSelect},
		{k.JumpApproval, k.JumpRunning, k.ToggleTheme, k.ToggleAccess, k.ToggleSidebar},
		{k.Dashboard},
		{k.Help},
	}
}
