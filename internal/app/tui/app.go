package tui

import (
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vburojevic/aistat/internal/app/tui/components"
	"github.com/vburojevic/aistat/internal/app/tui/state"
	"github.com/vburojevic/aistat/internal/app/tui/theme"
	"github.com/vburojevic/aistat/internal/app/tui/views"
	"github.com/vburojevic/aistat/internal/app/tui/widgets"
)

// SessionFetcher is a function type that fetches sessions
type SessionFetcher func() ([]state.SessionView, error)

// Run starts the TUI with the given config and session fetcher
func Run(cfg Config, fetcher SessionFetcher) error {
	m := New(cfg)
	m.sessionFetcher = fetcher

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// Config holds the TUI configuration
type Config struct {
	RefreshEvery time.Duration
	MaxSessions  int
	ShowEnded    bool // Toggle to show ended/stale sessions
}

// Model is the main TUI model - simplified single-view design
type Model struct {
	cfg Config

	// Session fetcher
	sessionFetcher SessionFetcher

	// Layout
	width  int
	height int

	// State
	sessions         []state.SessionView
	filteredSessions []state.SessionView
	cursor           int

	// Filter
	filter       textinput.Model
	filterActive bool
	filterQuery  string

	// UI State
	showHelp    bool
	showEnded   bool
	refreshing  bool
	err         error

	// Theme
	styles theme.Styles
}

// New creates a new TUI model
func New(cfg Config) *Model {
	styles := theme.DefaultStyles()

	// Initialize filter input
	f := textinput.New()
	f.Placeholder = "filter..."
	f.Prompt = ""
	f.CharLimit = 128
	f.Width = 40

	return &Model{
		cfg:       cfg,
		filter:    f,
		styles:    styles,
		showEnded: cfg.ShowEnded,
	}
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.fetchSessionsCmd(), m.tickCmd())
}

// Update handles messages
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.filter.Width = widgets.MinInt(60, widgets.MaxInt(20, m.width-20))

	case SessionsMsg:
		m.refreshing = false
		m.err = msg.Err
		if msg.Err == nil {
			m.sessions = msg.Sessions
			m.applyFilter()
		}

	case TickMsg:
		cmds = append(cmds, m.fetchSessionsCmd(), m.tickCmd())

	case tea.KeyMsg:
		cmd := m.handleKeyMsg(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// handleKeyMsg handles keyboard input
func (m *Model) handleKeyMsg(msg tea.KeyMsg) tea.Cmd {
	// Help view - dismiss on any key
	if m.showHelp {
		m.showHelp = false
		return nil
	}

	// Filter mode
	if m.filterActive {
		return m.handleFilterKeys(msg)
	}

	// Normal mode
	return m.handleNormalKeys(msg)
}

// handleFilterKeys handles keys when filter is active
func (m *Model) handleFilterKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.filterActive = false
		m.filter.Blur()
		m.filterQuery = ""
		m.filter.SetValue("")
		m.applyFilter()
		return nil
	case "enter":
		m.filterActive = false
		m.filter.Blur()
		m.filterQuery = m.filter.Value()
		m.applyFilter()
		return nil
	default:
		var cmd tea.Cmd
		m.filter, cmd = m.filter.Update(msg)
		m.filterQuery = m.filter.Value()
		m.applyFilter()
		return cmd
	}
}

// handleNormalKeys handles keys in normal mode (~10 essential shortcuts)
func (m *Model) handleNormalKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "q", "ctrl+c":
		return tea.Quit

	case "j", "down":
		m.moveCursor(1)

	case "k", "up":
		m.moveCursor(-1)

	case "/":
		m.filterActive = true
		m.filter.Focus()

	case "?":
		m.showHelp = true

	case "y":
		// Copy selected session ID
		if s := m.selectedSession(); s != nil {
			copyToClipboard(s.ID)
		}

	case "r":
		m.refreshing = true
		return m.fetchSessionsCmd()

	case "a":
		// Toggle show all (including ended)
		m.showEnded = !m.showEnded
		m.applyFilter()

	case "enter":
		// No-op for now, selection is just visual

	case "esc":
		// Clear filter if set
		if m.filterQuery != "" {
			m.filterQuery = ""
			m.filter.SetValue("")
			m.applyFilter()
		}
	}

	return nil
}

// View renders the UI
func (m *Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Header
	needsInputCount := widgets.NeedsInputCount(m.filteredSessions)
	b.WriteString(components.RenderHeader(needsInputCount, m.styles, m.width))
	b.WriteString("\n")

	// Filter bar
	b.WriteString(components.RenderFilterBar(m.filterQuery, m.filterActive, m.styles))
	b.WriteString("\n")

	// Main content: split view (list | detail)
	listWidth := m.width / 2
	detailWidth := m.width - listWidth - 3 // 3 for gap

	listContent := m.renderSessionList(listWidth)
	detailContent := components.RenderDetail(m.selectedSession(), m.styles, detailWidth)

	// Wrap in panels
	listPanel := m.styles.List.Width(listWidth).Height(m.height - 6).Render(listContent)
	detailPanel := m.styles.Detail.Width(detailWidth).Height(m.height - 6).Render(detailContent)

	gap := " "
	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, listPanel, gap, detailPanel)
	b.WriteString(mainContent)
	b.WriteString("\n")

	// Footer
	b.WriteString(components.RenderFooter(m.filterActive, m.refreshing, m.styles, m.width))

	// Help overlay
	if m.showHelp {
		// Render help on top (centered)
		helpContent := views.RenderHelpOverlay(m.styles)
		helpWidth := lipgloss.Width(helpContent)
		helpHeight := lipgloss.Height(helpContent)
		x := (m.width - helpWidth) / 2
		y := (m.height - helpHeight) / 2

		// This is a simple overlay - in a real app you'd use a more sophisticated approach
		return placeOverlay(x, y, helpContent, b.String())
	}

	return b.String()
}

// renderSessionList renders the session list grouped by project
func (m *Model) renderSessionList(width int) string {
	if len(m.filteredSessions) == 0 {
		if len(m.sessions) == 0 {
			return components.RenderEmptyState(m.styles)
		}
		return components.RenderFilteredEmpty(m.filterQuery, m.styles)
	}

	var lines []string

	// Group by project
	groups := m.groupByProject()

	rowIdx := 0
	for _, g := range groups {
		// Project divider
		divider := m.renderDivider(g.Project, width-4)
		lines = append(lines, divider)

		// Sessions in this project
		for _, s := range g.Sessions {
			row := m.renderSessionRow(s, rowIdx == m.cursor, width-4)
			lines = append(lines, row)
			rowIdx++
		}
	}

	return strings.Join(lines, "\n")
}

// renderDivider renders a project group divider
func (m *Model) renderDivider(project string, width int) string {
	if project == "" {
		project = "unknown"
	}

	// ─────────── project ───────────
	label := " " + project + " "
	labelLen := len(label)
	lineLen := (width - labelLen) / 2
	if lineLen < 3 {
		lineLen = 3
	}

	line := strings.Repeat("─", lineLen)
	return m.styles.Divider.Render(line + label + line)
}

// renderSessionRow renders a single session row
func (m *Model) renderSessionRow(s state.SessionView, selected bool, width int) string {
	// Arrow indicator for selection
	indicator := "  "
	if selected {
		indicator = m.styles.Selected.Render("▶ ")
	}

	// Status dot
	var dot string
	if widgets.IsEnded(s.Status) {
		dot = widgets.StatusDotDimmed(m.styles)
	} else {
		dot = widgets.StatusDot(s.Status, m.styles)
	}

	// Project name (truncated)
	project := widgets.TruncateString(s.Project, 20)
	if project == "" {
		project = "unknown"
	}

	// Age
	age := widgets.FormatAge(s.Age)

	// Build row
	row := indicator + dot + " " + widgets.PadRight(project, 20) + "  " + widgets.PadLeft(age, 6)

	// Apply dimming for ended sessions
	if widgets.IsEnded(s.Status) {
		row = m.styles.RowDim.Render(row)
	}

	return row
}

// Helper methods

func (m *Model) fetchSessionsCmd() tea.Cmd {
	if m.sessionFetcher == nil {
		return nil
	}
	m.refreshing = true
	fetcher := m.sessionFetcher
	return func() tea.Msg {
		sessions, err := fetcher()
		return SessionsMsg{Sessions: sessions, Err: err}
	}
}

func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(m.cfg.RefreshEvery, func(t time.Time) tea.Msg { return TickMsg(t) })
}

func (m *Model) selectedSession() *state.SessionView {
	if m.cursor < 0 || m.cursor >= len(m.filteredSessions) {
		return nil
	}
	return &m.filteredSessions[m.cursor]
}

func (m *Model) moveCursor(delta int) {
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.filteredSessions) {
		m.cursor = len(m.filteredSessions) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *Model) applyFilter() {
	filtered := make([]state.SessionView, 0, len(m.sessions))

	for _, s := range m.sessions {
		// Hide ended/stale unless showEnded is true
		if !m.showEnded && widgets.IsEnded(s.Status) {
			continue
		}

		// Apply text filter
		if m.filterQuery != "" {
			query := strings.ToLower(m.filterQuery)

			// Smart filter: check for provider names
			if query == "claude" && s.Provider != state.ProviderClaude {
				continue
			}
			if query == "codex" && s.Provider != state.ProviderCodex {
				continue
			}

			// Otherwise fuzzy match on project/ID
			if !strings.Contains(strings.ToLower(s.Project), query) &&
				!strings.Contains(strings.ToLower(s.ID), query) {
				continue
			}
		}

		filtered = append(filtered, s)
	}

	// Sort: needs input first, then active, then idle, then by recency
	sortByUrgency(filtered)

	m.filteredSessions = filtered

	// Adjust cursor if out of bounds
	if m.cursor >= len(filtered) {
		m.cursor = len(filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

type projectGroup struct {
	Project  string
	Sessions []state.SessionView
}

func (m *Model) groupByProject() []projectGroup {
	groups := make(map[string][]state.SessionView)
	order := []string{}

	for _, s := range m.filteredSessions {
		proj := s.Project
		if proj == "" {
			proj = ""
		}
		if _, ok := groups[proj]; !ok {
			order = append(order, proj)
		}
		groups[proj] = append(groups[proj], s)
	}

	result := make([]projectGroup, 0, len(order))
	for _, p := range order {
		result = append(result, projectGroup{Project: p, Sessions: groups[p]})
	}
	return result
}

// sortByUrgency sorts sessions: needs input first, then active, then idle
func sortByUrgency(sessions []state.SessionView) {
	for i := 0; i < len(sessions); i++ {
		for j := i + 1; j < len(sessions); j++ {
			if urgencyScore(sessions[j]) > urgencyScore(sessions[i]) {
				sessions[i], sessions[j] = sessions[j], sessions[i]
			} else if urgencyScore(sessions[j]) == urgencyScore(sessions[i]) {
				// Same urgency - sort by recency
				if sessions[j].LastSeen.After(sessions[i].LastSeen) {
					sessions[i], sessions[j] = sessions[j], sessions[i]
				}
			}
		}
	}
}

func urgencyScore(s state.SessionView) int {
	ui := widgets.ToUIStatus(s.Status)
	switch ui {
	case widgets.UIStatusNeedsInput:
		return 100
	case widgets.UIStatusActive:
		return 50
	default:
		return 10
	}
}

// Utility functions

func copyToClipboard(text string) error {
	if runtime.GOOS == "darwin" {
		cmd := exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	}
	return nil
}

// placeOverlay places content at x,y over base (simple overlay)
func placeOverlay(x, y int, overlay, base string) string {
	// Simple implementation: just return overlay centered
	// A proper implementation would merge the strings character by character
	return lipgloss.Place(
		lipgloss.Width(base),
		lipgloss.Height(base),
		lipgloss.Center,
		lipgloss.Center,
		overlay,
	)
}
