package theme

import "github.com/charmbracelet/lipgloss"

// Styles holds all the lipgloss styles for the TUI
type Styles struct {
	// Text styles
	Text       lipgloss.Style // Primary text
	Muted      lipgloss.Style // Secondary/dimmed text
	Title      lipgloss.Style // App title
	Label      lipgloss.Style // Key labels in detail pane
	Value      lipgloss.Style // Values in detail pane

	// Layout styles
	App        lipgloss.Style // Outer app container
	Header     lipgloss.Style // Header bar
	Footer     lipgloss.Style // Footer/shortcut bar
	Panel      lipgloss.Style // Generic panel with border
	List       lipgloss.Style // Session list panel
	Detail     lipgloss.Style // Detail pane panel
	Divider    lipgloss.Style // Project group divider

	// Status dots
	DotActive     lipgloss.Style // Green dot
	DotIdle       lipgloss.Style // Blue dot
	DotNeedsInput lipgloss.Style // Peach dot

	// Badges
	BadgeNeedsInput lipgloss.Style // "N need input" badge in header

	// Selection
	Selected lipgloss.Style // Arrow indicator style
	Row      lipgloss.Style // Normal row
	RowDim   lipgloss.Style // Ended/dimmed row

	// Filter
	FilterPrompt lipgloss.Style // "/ " prompt
	FilterText   lipgloss.Style // Filter text input

	// Help overlay
	HelpOverlay lipgloss.Style // Help modal container
	HelpTitle   lipgloss.Style // Help modal title
	HelpKey     lipgloss.Style // Shortcut key
	HelpDesc    lipgloss.Style // Shortcut description

	// Provider
	ProviderClaude lipgloss.Style
	ProviderCodex  lipgloss.Style
}

// NewStyles creates styles from the theme
func NewStyles(t Theme) Styles {
	s := Styles{}

	// Text styles
	s.Text = lipgloss.NewStyle().Foreground(t.Text)
	s.Muted = lipgloss.NewStyle().Foreground(t.Muted)
	s.Title = lipgloss.NewStyle().Foreground(t.Text).Bold(true)
	s.Label = lipgloss.NewStyle().Foreground(t.Muted).Width(12)
	s.Value = lipgloss.NewStyle().Foreground(t.Text)

	// Layout styles
	s.App = lipgloss.NewStyle().
		Background(t.Background)

	s.Header = lipgloss.NewStyle().
		Foreground(t.Text).
		Padding(0, 1)

	s.Footer = lipgloss.NewStyle().
		Foreground(t.Muted).
		Padding(0, 1)

	s.Panel = lipgloss.NewStyle().
		Border(BorderRounded).
		BorderForeground(t.Border).
		Padding(0, 1)

	s.List = lipgloss.NewStyle().
		Border(BorderRounded).
		BorderForeground(t.Border)

	s.Detail = lipgloss.NewStyle().
		Border(BorderRounded).
		BorderForeground(t.Border).
		Padding(0, 1)

	s.Divider = lipgloss.NewStyle().
		Foreground(t.Faint)

	// Status dots (just colored text)
	s.DotActive = lipgloss.NewStyle().Foreground(t.Active)
	s.DotIdle = lipgloss.NewStyle().Foreground(t.Idle)
	s.DotNeedsInput = lipgloss.NewStyle().Foreground(t.NeedsInput)

	// Badge for header "N need input"
	s.BadgeNeedsInput = lipgloss.NewStyle().
		Foreground(t.NeedsInput).
		Bold(true)

	// Selection
	s.Selected = lipgloss.NewStyle().
		Foreground(t.NeedsInput).
		Bold(true)

	s.Row = lipgloss.NewStyle().
		Foreground(t.Text).
		Padding(0, 1)

	s.RowDim = lipgloss.NewStyle().
		Foreground(t.Muted).
		Padding(0, 1)

	// Filter
	s.FilterPrompt = lipgloss.NewStyle().
		Foreground(t.Muted)

	s.FilterText = lipgloss.NewStyle().
		Foreground(t.Text)

	// Help overlay
	s.HelpOverlay = lipgloss.NewStyle().
		Border(BorderRounded).
		BorderForeground(t.Border).
		Padding(1, 2).
		Background(t.Surface)

	s.HelpTitle = lipgloss.NewStyle().
		Foreground(t.Text).
		Bold(true).
		MarginBottom(1)

	s.HelpKey = lipgloss.NewStyle().
		Foreground(t.NeedsInput).
		Width(12)

	s.HelpDesc = lipgloss.NewStyle().
		Foreground(t.Muted)

	// Provider styles
	s.ProviderClaude = lipgloss.NewStyle().Foreground(t.Claude)
	s.ProviderCodex = lipgloss.NewStyle().Foreground(t.Codex)

	return s
}

// DefaultStyles returns styles using the default theme
func DefaultStyles() Styles {
	return NewStyles(Default)
}
