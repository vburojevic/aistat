package tui

import (
	"math"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/harmonica"
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
	pinned           map[string]bool // Pinned/bookmarked session IDs

	// Filter
	filter       textinput.Model
	filterActive bool
	filterQuery  string

	// UI State
	showHelp     bool
	showEnded    bool
	refreshing   bool
	spinnerFrame int
	err          error

	// Cursor animation (spring physics)
	cursorSpring   harmonica.Spring
	cursorY        float64 // Current animated Y position
	cursorVelocity float64 // Current velocity
	targetCursor   int     // Target cursor position
	animating      bool    // Whether animation is in progress

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
		cfg:          cfg,
		filter:       f,
		styles:       styles,
		showEnded:    cfg.ShowEnded,
		pinned:       make(map[string]bool),
		cursorSpring: harmonica.NewSpring(harmonica.FPS(60), 6.0, 0.5),
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

	case SpinnerTickMsg:
		if m.refreshing {
			m.spinnerFrame = (m.spinnerFrame + 1) % len(SpinnerFrames)
			cmds = append(cmds, m.spinnerTickCmd())
		}

	case AnimationTickMsg:
		if m.animating {
			m.cursorY, m.cursorVelocity = m.cursorSpring.Update(m.cursorY, m.cursorVelocity, float64(m.targetCursor))

			// Check if animation has settled
			distance := math.Abs(m.cursorY - float64(m.targetCursor))
			if distance < 0.5 && math.Abs(m.cursorVelocity) < 0.01 {
				m.animating = false
				m.cursor = m.targetCursor
				m.cursorY = float64(m.targetCursor)
				m.cursorVelocity = 0
			} else {
				// Update display cursor to nearest row
				m.cursor = int(math.Round(m.cursorY))
				cmds = append(cmds, m.animationTickCmd())
			}
		}

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
		return m.moveCursor(1)

	case "k", "up":
		return m.moveCursor(-1)

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

	case "b":
		// Toggle bookmark/pin on selected session
		if s := m.selectedSession(); s != nil {
			m.togglePin(s.ID)
			m.applyFilter() // Re-sort to move pinned to top
		}

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
	b.WriteString(components.RenderHeader(m.filteredSessions, m.styles, m.width))
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
	b.WriteString(components.RenderFooter(m.filterActive, m.refreshing, m.spinnerFrame, SpinnerFrames, m.styles, m.width))

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

// renderSessionList renders the session list grouped by project and branch
func (m *Model) renderSessionList(width int) string {
	// Show error state if there was a fetch error
	if m.err != nil {
		return components.RenderError(m.err, m.styles)
	}

	if len(m.filteredSessions) == 0 {
		if len(m.sessions) == 0 {
			return components.RenderEmptyState(m.styles)
		}
		return components.RenderFilteredEmpty(m.filterQuery, m.styles)
	}

	var lines []string

	// Group by project and branch
	groups := m.groupByProjectAndBranch()

	rowIdx := 0
	for _, g := range groups {
		// Count total sessions in this project
		projectCount := 0
		for _, b := range g.Branches {
			projectCount += len(b.Sessions)
		}

		// Project divider with count
		divider := m.renderDivider(g.Project, projectCount, width-4)
		lines = append(lines, divider)

		// Branches in this project
		for _, b := range g.Branches {
			// Branch sub-header with count
			branchHeader := m.renderBranchHeader(b.Branch, len(b.Sessions))
			lines = append(lines, branchHeader)

			// Sessions in this branch
			for _, s := range b.Sessions {
				row := m.renderSessionRow(s, rowIdx == m.cursor, width-4)
				lines = append(lines, row)
				rowIdx++
			}
		}
	}

	return strings.Join(lines, "\n")
}

// renderBranchHeader renders a branch sub-header with session count
func (m *Model) renderBranchHeader(branch string, count int) string {
	if branch == "" {
		branch = "unknown"
	}
	countStr := ""
	if count > 0 {
		countStr = " (" + widgets.FormatInt(count) + ")"
	}
	return m.styles.Muted.Render("  ├─ " + branch + countStr)
}

// renderDivider renders a project group divider with session count
func (m *Model) renderDivider(project string, count int, width int) string {
	if project == "" {
		project = "unknown"
	}

	// ─────────── project (3) ───────────
	countStr := ""
	if count > 0 {
		countStr = " (" + widgets.FormatInt(count) + ")"
	}
	label := " " + project + countStr + " "
	labelLen := len(label)
	lineLen := (width - labelLen) / 2
	if lineLen < 3 {
		lineLen = 3
	}

	line := strings.Repeat("─", lineLen)
	return m.styles.Divider.Render(line + label + line)
}

// renderSessionRow renders a single session row (under a branch header)
func (m *Model) renderSessionRow(s state.SessionView, selected bool, width int) string {
	// Indentation for sessions under branch
	indent := "    "

	// Arrow indicator for selection
	indicator := "  "
	if selected {
		indicator = m.styles.Selected.Render("❯ ")
	}

	// Pin indicator (★) for bookmarked sessions
	pinIndicator := " "
	if m.isPinned(s.ID) {
		pinIndicator = m.styles.DotNeedsInput.Render("★")
	}

	// Urgency indicator for needs-input sessions
	urgency := " "
	if widgets.ToUIStatus(s.Status) == widgets.UIStatusNeedsInput {
		urgency = m.styles.DotNeedsInput.Render("⚡")
	}

	// Status icon (keeps its color regardless of age)
	var icon string
	if widgets.IsEnded(s.Status) {
		icon = widgets.StatusIconDimmed(m.styles)
	} else {
		icon = widgets.StatusIcon(s.Status, m.styles)
	}

	// Model name (truncated) with model-specific color - fixed width 14 chars
	model := widgets.TruncateString(s.Model, 14)
	if model == "" {
		model = "-"
	}
	modelStyle := m.modelStyle(s.Model, s.Age)
	modelText := modelStyle.Render(widgets.PadRight(model, 14))

	// Status-aware age display with status-specific color
	statusAge := m.formatStatusAgeStyled(s)

	// Build row content: indent + indicator + pin + urgency + icon + model + status-age
	rowContent := indent + indicator + pinIndicator + urgency + icon + " " + modelText + " " + statusAge

	// Apply background highlight for selected row (no fixed width)
	if selected {
		return m.styles.RowSelected.Render(rowContent)
	}

	return rowContent
}

// modelStyle returns the appropriate style for a model name with age fading
func (m *Model) modelStyle(model string, age time.Duration) lipgloss.Style {
	modelLower := strings.ToLower(model)

	// Get base color from model
	var baseStyle lipgloss.Style
	switch {
	case strings.Contains(modelLower, "opus"):
		baseStyle = m.styles.ModelOpus
	case strings.Contains(modelLower, "sonnet"):
		baseStyle = m.styles.ModelSonnet
	case strings.Contains(modelLower, "haiku"):
		baseStyle = m.styles.ModelHaiku
	default:
		// Use age-based dimming for unknown models
		return m.ageTextStyle(age)
	}

	// Apply age-based dimming to the model color
	if age >= 24*time.Hour {
		return m.styles.RowDim40
	} else if age >= 6*time.Hour {
		return m.styles.RowDim60
	} else if age >= 1*time.Hour {
		return m.styles.RowDim80
	}
	return baseStyle
}

// formatStatusAgeStyled returns status-aware age with color styling
func (m *Model) formatStatusAgeStyled(s state.SessionView) string {
	age := widgets.FormatAge(s.Age)
	var text string
	var style lipgloss.Style

	switch {
	case s.Status == state.StatusRunning:
		text = "running " + age
		style = m.styles.StatusRunning
	case s.Status == state.StatusApproval || s.Status == state.StatusNeedsAttn:
		text = "waiting " + age
		style = m.styles.StatusWaiting
	case widgets.IsEnded(s.Status):
		text = "ended " + age + " ago"
		style = m.styles.StatusEnded
	default:
		text = "idle " + age
		style = m.styles.StatusIdle
	}

	// Apply age-based dimming for older sessions
	if s.Age >= 24*time.Hour {
		style = m.styles.RowDim40
	} else if s.Age >= 6*time.Hour {
		style = m.styles.RowDim60
	} else if s.Age >= 1*time.Hour {
		style = m.styles.RowDim80
	}

	return style.Render(widgets.PadLeft(text, 15))
}

// formatStatusAge returns age with status context (e.g., "running 2m", "idle 15m")
func formatStatusAge(s state.SessionView) string {
	age := widgets.FormatAge(s.Age)

	switch {
	case s.Status == state.StatusRunning:
		return "running " + age
	case s.Status == state.StatusApproval || s.Status == state.StatusNeedsAttn:
		return "waiting " + age
	case widgets.IsEnded(s.Status):
		return "ended " + age + " ago"
	default:
		return "idle " + age
	}
}

// ageTextStyle returns a style based on session age for fade effect
func (m *Model) ageTextStyle(age time.Duration) lipgloss.Style {
	switch {
	case age < 1*time.Hour:
		return m.styles.Row
	case age < 6*time.Hour:
		return m.styles.RowDim80
	case age < 24*time.Hour:
		return m.styles.RowDim60
	default:
		return m.styles.RowDim40
	}
}

// Helper methods

func (m *Model) fetchSessionsCmd() tea.Cmd {
	if m.sessionFetcher == nil {
		return nil
	}
	m.refreshing = true
	fetcher := m.sessionFetcher
	return tea.Batch(
		func() tea.Msg {
			sessions, err := fetcher()
			return SessionsMsg{Sessions: sessions, Err: err}
		},
		m.spinnerTickCmd(),
	)
}

func (m *Model) spinnerTickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg { return SpinnerTickMsg{} })
}

func (m *Model) animationTickCmd() tea.Cmd {
	return tea.Tick(16*time.Millisecond, func(t time.Time) tea.Msg { return AnimationTickMsg{} }) // ~60fps
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

func (m *Model) moveCursor(delta int) tea.Cmd {
	// Calculate new target position
	newTarget := m.targetCursor + delta
	if newTarget < 0 {
		newTarget = 0
	}
	if newTarget >= len(m.filteredSessions) {
		newTarget = len(m.filteredSessions) - 1
	}
	if newTarget < 0 {
		newTarget = 0
	}

	// If target hasn't changed, no animation needed
	if newTarget == m.targetCursor && !m.animating {
		return nil
	}

	m.targetCursor = newTarget

	// Initialize animation if not already running
	if !m.animating {
		m.cursorY = float64(m.cursor)
		m.cursorVelocity = 0
	}
	m.animating = true

	return m.animationTickCmd()
}

func (m *Model) togglePin(id string) {
	// Strip ANSI codes from ID for consistent key
	cleanID := stripANSI(id)
	if m.pinned[cleanID] {
		delete(m.pinned, cleanID)
	} else {
		m.pinned[cleanID] = true
	}
}

func (m *Model) isPinned(id string) bool {
	return m.pinned[stripANSI(id)]
}

func (m *Model) applyPinnedFirst(list []state.SessionView) []state.SessionView {
	if len(m.pinned) == 0 {
		return list
	}
	pinned := make([]state.SessionView, 0, len(list))
	rest := make([]state.SessionView, 0, len(list))
	for _, sess := range list {
		if m.isPinned(sess.ID) {
			pinned = append(pinned, sess)
		} else {
			rest = append(rest, sess)
		}
	}
	return append(pinned, rest...)
}

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	result := strings.Builder{}
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}

func (m *Model) applyFilter() {
	filtered := make([]state.SessionView, 0, len(m.sessions))
	recentWindow := 24 * time.Hour

	for _, s := range m.sessions {
		// Recent sessions (last 24h) are always shown
		// Older ended sessions are hidden unless showEnded is true
		isRecent := s.Age < recentWindow
		isEnded := widgets.IsEnded(s.Status)

		if !isRecent && isEnded && !m.showEnded {
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

	// Move pinned sessions to the top
	m.filteredSessions = m.applyPinnedFirst(filtered)

	// Adjust cursor if out of bounds
	if m.cursor >= len(filtered) {
		m.cursor = len(filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

type branchGroup struct {
	Branch   string
	Sessions []state.SessionView
}

type projectGroup struct {
	Project  string
	Branches []branchGroup
}

func (m *Model) groupByProjectAndBranch() []projectGroup {
	// First level: group by project
	projectOrder := []string{}
	projectMap := make(map[string]map[string][]state.SessionView) // project -> branch -> sessions

	for _, s := range m.filteredSessions {
		proj := s.Project
		if proj == "" {
			proj = "unknown"
		}
		branch := s.Branch
		if branch == "" {
			branch = "unknown"
		}

		if _, ok := projectMap[proj]; !ok {
			projectOrder = append(projectOrder, proj)
			projectMap[proj] = make(map[string][]state.SessionView)
		}
		projectMap[proj][branch] = append(projectMap[proj][branch], s)
	}

	// Build result with branch ordering preserved
	result := make([]projectGroup, 0, len(projectOrder))
	for _, proj := range projectOrder {
		branches := projectMap[proj]
		branchOrder := []string{}
		for branch := range branches {
			branchOrder = append(branchOrder, branch)
		}

		branchGroups := make([]branchGroup, 0, len(branchOrder))
		for _, branch := range branchOrder {
			branchGroups = append(branchGroups, branchGroup{
				Branch:   branch,
				Sessions: branches[branch],
			})
		}

		result = append(result, projectGroup{
			Project:  proj,
			Branches: branchGroups,
		})
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
