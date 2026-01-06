package theme

import "github.com/charmbracelet/lipgloss"

// Styles holds all the lipgloss styles for the TUI
type Styles struct {
	// Text styles
	Title  lipgloss.Style
	Muted  lipgloss.Style
	Accent lipgloss.Style
	Bold   lipgloss.Style
	Dim    lipgloss.Style

	// Container styles
	Box        lipgloss.Style
	Panel      lipgloss.Style
	DetailBox  lipgloss.Style
	Card       lipgloss.Style
	OverlayBox lipgloss.Style

	// Header and bars
	HeaderBox   lipgloss.Style
	ShortcutBar lipgloss.Style
	Section     lipgloss.Style
	GroupHeader lipgloss.Style

	// Pills and badges
	Pill       lipgloss.Style
	PillActive lipgloss.Style

	// Status badges
	BadgeRun   lipgloss.Style
	BadgeWait  lipgloss.Style
	BadgeAppr  lipgloss.Style
	BadgeAttn  lipgloss.Style
	BadgeStale lipgloss.Style
	BadgeEnded lipgloss.Style

	// Selection and highlighting
	Selected   lipgloss.Style
	OverlaySel lipgloss.Style
	Changed    lipgloss.Style

	// Provider styles
	ProviderClaude lipgloss.Style
	ProviderCodex  lipgloss.Style
}

// NewStyles creates a new Styles instance from a theme
func NewStyles(t Theme, accessible bool) Styles {
	s := Styles{}

	// Text styles
	s.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Accent)

	s.Muted = lipgloss.NewStyle().
		Foreground(t.Subtext0)

	s.Accent = lipgloss.NewStyle().
		Foreground(t.Accent)

	s.Bold = lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Text)

	s.Dim = lipgloss.NewStyle().
		Foreground(t.Overlay0).
		Faint(true)

	// Container styles
	s.Box = lipgloss.NewStyle().
		Padding(0, 1)

	s.Panel = lipgloss.NewStyle().
		Border(BorderSubtle).
		BorderForeground(t.Surface2).
		Padding(0, 1)

	s.DetailBox = lipgloss.NewStyle().
		Border(BorderSubtle).
		BorderForeground(t.Surface2).
		Padding(0, 1)

	s.Card = lipgloss.NewStyle().
		Border(BorderSubtle).
		BorderForeground(t.Surface1).
		Padding(0, 1)

	s.OverlayBox = lipgloss.NewStyle().
		Border(BorderSubtle).
		BorderForeground(t.Accent).
		Padding(1, 2)

	// Header and bars
	s.HeaderBox = lipgloss.NewStyle().
		Padding(0, 1)

	s.ShortcutBar = lipgloss.NewStyle().
		Foreground(t.Subtext0).
		Background(t.Surface0).
		Padding(0, 1)

	s.Section = lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Accent)

	s.GroupHeader = lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Secondary)

	// Pills and badges
	s.Pill = lipgloss.NewStyle().
		Foreground(t.Subtext1).
		Background(t.Surface0).
		Padding(0, 1)

	s.PillActive = lipgloss.NewStyle().
		Foreground(t.Crust).
		Background(t.Accent).
		Bold(true).
		Padding(0, 1)

	// Status badges - dark background with colored text
	baseBadge := lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1)

	s.BadgeRun = baseBadge.Copy().
		Foreground(t.Crust).
		Background(t.Running)

	s.BadgeWait = baseBadge.Copy().
		Foreground(t.Crust).
		Background(t.Waiting)

	s.BadgeAppr = baseBadge.Copy().
		Foreground(t.Crust).
		Background(t.Approval)

	s.BadgeAttn = baseBadge.Copy().
		Foreground(t.Crust).
		Background(t.NeedsAttn)

	s.BadgeStale = baseBadge.Copy().
		Foreground(t.Text).
		Background(t.Surface1)

	s.BadgeEnded = baseBadge.Copy().
		Foreground(t.Subtext0).
		Background(t.Surface0).
		Faint(true)

	// Selection and highlighting
	s.Selected = lipgloss.NewStyle().
		Foreground(t.Accent).
		Bold(true)

	s.OverlaySel = lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Accent)

	s.Changed = lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Accent)

	// Provider styles
	s.ProviderClaude = lipgloss.NewStyle().
		Foreground(t.Claude).
		Bold(true)

	s.ProviderCodex = lipgloss.NewStyle().
		Foreground(t.Codex).
		Bold(true)

	// Apply accessibility overrides
	if accessible {
		s = applyAccessibleOverrides(s, t)
	}

	return s
}

// applyAccessibleOverrides modifies styles for better accessibility
func applyAccessibleOverrides(s Styles, t Theme) Styles {
	// Remove faint styling
	s.Muted = lipgloss.NewStyle().Foreground(t.Subtext1)
	s.Dim = lipgloss.NewStyle().Foreground(t.Overlay1)

	// Use normal borders instead of rounded
	s.Panel = s.Panel.Border(BorderSharp)
	s.DetailBox = s.DetailBox.Border(BorderSharp)
	s.Card = s.Card.Border(BorderSharp)
	s.OverlayBox = s.OverlayBox.Border(BorderSharp)

	// Add underlines to status badges instead of relying on color alone
	s.BadgeRun = s.BadgeRun.Underline(true)
	s.BadgeWait = s.BadgeWait.Underline(true)
	s.BadgeAppr = s.BadgeAppr.Underline(true)
	s.BadgeAttn = s.BadgeAttn.Underline(true)
	s.BadgeStale = s.BadgeStale.Underline(true)
	s.BadgeEnded = s.BadgeEnded.Underline(true).UnsetFaint()

	// Pills use borders instead of background colors
	s.Pill = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Padding(0, 1)
	s.PillActive = s.Pill.Copy().Bold(true)

	return s
}
