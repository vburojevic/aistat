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
	Selected    lipgloss.Style // Arrow indicator style
	RowSelected lipgloss.Style // Selected row background
	Row         lipgloss.Style // Normal row (< 1h)
	RowDim80    lipgloss.Style // Slightly dimmed (1-6h)
	RowDim60 lipgloss.Style // More dimmed (6-24h)
	RowDim   lipgloss.Style // Ended/dimmed row
	RowDim40 lipgloss.Style // Very dimmed (>24h)

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

	// Model styles
	ModelOpus   lipgloss.Style
	ModelSonnet lipgloss.Style
	ModelHaiku  lipgloss.Style

	// Status text styles (for status-aware age labels)
	StatusRunning lipgloss.Style
	StatusIdle    lipgloss.Style
	StatusWaiting lipgloss.Style
	StatusEnded   lipgloss.Style

	// Error styles
	ErrorText   lipgloss.Style
	ErrorBorder lipgloss.Style
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

	s.RowSelected = lipgloss.NewStyle().
		Background(t.Highlight)

	s.Row = lipgloss.NewStyle().
		Foreground(t.Text).
		Padding(0, 1)

	s.RowDim80 = lipgloss.NewStyle().
		Foreground(t.Dim80).
		Padding(0, 1)

	s.RowDim60 = lipgloss.NewStyle().
		Foreground(t.Dim60).
		Padding(0, 1)

	s.RowDim = lipgloss.NewStyle().
		Foreground(t.Muted).
		Padding(0, 1)

	s.RowDim40 = lipgloss.NewStyle().
		Foreground(t.Dim40).
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

	// Model styles
	s.ModelOpus = lipgloss.NewStyle().Foreground(t.ModelOpus)
	s.ModelSonnet = lipgloss.NewStyle().Foreground(t.ModelSonnet)
	s.ModelHaiku = lipgloss.NewStyle().Foreground(t.ModelHaiku)

	// Status text styles
	s.StatusRunning = lipgloss.NewStyle().Foreground(t.Active)
	s.StatusIdle = lipgloss.NewStyle().Foreground(t.Idle)
	s.StatusWaiting = lipgloss.NewStyle().Foreground(t.NeedsInput)
	s.StatusEnded = lipgloss.NewStyle().Foreground(t.Muted)

	// Error styles
	s.ErrorText = lipgloss.NewStyle().Foreground(t.Error)
	s.ErrorBorder = lipgloss.NewStyle().BorderForeground(t.Error)

	return s
}

// DefaultStyles returns styles using the default theme
func DefaultStyles() Styles {
	return NewStyles(Default)
}
