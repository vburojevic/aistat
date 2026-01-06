package tui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/table"
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

// Config holds the TUI configuration (mirrors app.Config relevant fields)
type Config struct {
	Redact         bool
	ActiveWindow   time.Duration
	RunningWindow  time.Duration
	RefreshEvery   time.Duration
	MaxSessions    int
	IncludeEnded   bool
	ProviderFilter string
	ProjectFilters []string
	StatusFilters  []state.Status
	SortBy         string
	GroupBy        string
	IncludeLastMsg bool
}

// Model is the main TUI model
type Model struct {
	cfg Config

	// Session fetcher - injected by caller
	sessionFetcher SessionFetcher

	// Core components
	table   table.Model
	filter  textinput.Model
	palette textinput.Model
	help    help.Model
	keys    KeyMap

	// Layout
	width  int
	height int

	// State
	appState *state.AppState
	filters  *state.FilterState

	// Views
	dashboardView *views.DashboardView
	projectsView  *views.ProjectsView

	// UI State
	viewMode      ViewMode
	detailMode    DetailMode
	columnMode    ColumnMode
	showLastCol   bool
	showSidebar   bool
	showBanner    bool
	paletteOpen   bool
	paletteMsg    string
	effectiveMode ColumnMode

	// Theme
	themeIndex int
	theme      theme.Theme
	styles     theme.Styles
	accessible bool

	// Layout animation
	sidebarWidth  int
	sidebarTarget int
	detailWidth   int
	detailTarget  int

	// Table metadata
	idColumn int

	// Error state
	err error
}

// New creates a new TUI model
func New(cfg Config) *Model {
	// Initialize theme
	th := theme.Mocha
	s := theme.NewStyles(th, false)

	// Initialize filter input
	f := textinput.New()
	f.Placeholder = "filter (/, esc)"
	f.Prompt = "◆ "
	f.CharLimit = 128
	f.Width = 40

	// Initialize palette input
	pal := textinput.New()
	pal.Placeholder = "command (help)"
	pal.Prompt = ": "
	pal.CharLimit = 128
	pal.Width = 50

	// Initialize table
	cols, idCol := columnsFor(120, ColumnModeFull, false)
	tbl := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
	)

	// Initialize state
	appState := state.NewAppState()
	filters := state.NewFilterState()

	// Apply config filters
	if cfg.ProviderFilter != "" {
		filters.ProviderFilter[state.Provider(cfg.ProviderFilter)] = true
	}
	for _, s := range cfg.StatusFilters {
		filters.StatusFilter[s] = true
	}
	for _, p := range cfg.ProjectFilters {
		filters.ProjectFilter[strings.ToLower(p)] = true
	}

	m := &Model{
		cfg:         cfg,
		table:       tbl,
		filter:      f,
		palette:     pal,
		help:        help.New(),
		keys:        DefaultKeyMap(),
		appState:    appState,
		filters:     filters,
		viewMode:    ViewDashboard, // Dashboard-first!
		detailMode:  DetailSplit,
		columnMode:  ColumnModeFull,
		showLastCol: false,
		showSidebar: true,
		showBanner:  false,
		theme:       th,
		styles:      s,
		themeIndex:  0,
		accessible:  false,
		idColumn:    idCol,
	}

	// Initialize views
	m.dashboardView = views.NewDashboardView(m.styles, m.filters)
	m.projectsView = views.NewProjectsView(m.styles, m.filters)

	m.applyTableStyles()

	return m
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	// Start with dashboard focused
	m.dashboardView.Focus()
	return tea.Batch(m.fetchSessionsCmd(), m.tickCmd())
}

// Update handles messages
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.filter.Width = widgets.MinInt(60, widgets.MaxInt(20, m.width-20))
		m.dashboardView.SetSize(m.width, m.height)
		m.projectsView.SetSize(m.width, m.height)
		cmds = append(cmds, m.updateLayoutTargets())
		m.applyColumns()
		tableHeight := widgets.MaxInt(5, m.height-2-1-9-1)
		m.table.SetHeight(tableHeight)

	case SessionsMsg:
		m.err = msg.Err
		if msg.Err == nil {
			m.appState.UpdateSessions(msg.Sessions, m.cfg.RefreshEvery)
			m.applyFilterAndUpdateRows()
		}

	case TickMsg:
		cmds = append(cmds, m.fetchSessionsCmd(), m.tickCmd())

	case AnimMsg:
		if m.stepLayoutAnimation() {
			cmds = append(cmds, m.animCmd())
		}

	case tea.KeyMsg:
		cmd := m.handleKeyMsg(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Update table
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// handleKeyMsg handles keyboard input
func (m *Model) handleKeyMsg(msg tea.KeyMsg) tea.Cmd {
	// Help view
	if m.viewMode == ViewHelp {
		switch msg.String() {
		case "?", "h", "esc":
			m.viewMode = ViewSessions
			return nil
		case "ctrl+c":
			return tea.Quit
		}
		return nil
	}

	// Dashboard view
	if m.viewMode == ViewDashboard {
		return m.handleDashboardKeys(msg)
	}

	// Projects view
	if m.viewMode == ViewProjects {
		return m.handleProjectsKeys(msg)
	}

	// Palette open
	if m.paletteOpen {
		return m.handlePaletteKeys(msg)
	}

	// Filter focused
	if m.filter.Focused() {
		return m.handleFilterKeys(msg)
	}

	// Default: session list view
	return m.handleSessionListKeys(msg)
}

// handleDashboardKeys handles keys in dashboard mode
func (m *Model) handleDashboardKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "tab", "esc":
		m.viewMode = ViewSessions
		m.dashboardView.Blur()
		return nil
	case "ctrl+c":
		return tea.Quit
	case "up", "k":
		items := state.FilterDashboardItems(m.appState.AllSessions, m.filters)
		m.dashboardView.MoveCursor(-1, len(items))
		return nil
	case "down", "j":
		items := state.FilterDashboardItems(m.appState.AllSessions, m.filters)
		m.dashboardView.MoveCursor(1, len(items))
		return nil
	case "enter":
		if proj := m.dashboardView.SelectedProject(m.appState.AllSessions); proj != nil {
			m.filters.SetProject(proj.Name)
			m.viewMode = ViewSessions
			m.dashboardView.Blur()
			m.applyFilterAndUpdateRows()
		}
		return nil
	case " ":
		if proj := m.dashboardView.SelectedProject(m.appState.AllSessions); proj != nil {
			m.filters.ToggleProject(proj.Name)
			m.applyFilterAndUpdateRows()
		}
		return nil
	case "a":
		m.filters.ProjectFilter = make(map[string]bool)
		m.applyFilterAndUpdateRows()
		return nil
	default:
		// Forward to filter input
		var cmd tea.Cmd
		input := m.dashboardView.FilterInput()
		*input, cmd = input.Update(msg)
		return cmd
	}
}

// handleProjectsKeys handles keys in projects picker mode
func (m *Model) handleProjectsKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.viewMode = ViewSessions
		m.projectsView.Blur()
		return nil
	case "ctrl+c":
		return tea.Quit
	case "up", "k":
		m.projectsView.MoveCursor(-1, len(m.appState.ProjectItems))
		return nil
	case "down", "j":
		m.projectsView.MoveCursor(1, len(m.appState.ProjectItems))
		return nil
	case "enter", " ":
		if proj := m.projectsView.SelectedProject(m.appState.ProjectItems); proj != nil {
			m.filters.ToggleProject(proj.Name)
			m.applyFilterAndUpdateRows()
		}
		return nil
	case "a":
		m.filters.ProjectFilter = make(map[string]bool)
		m.applyFilterAndUpdateRows()
		return nil
	default:
		var cmd tea.Cmd
		input := m.projectsView.FilterInput()
		*input, cmd = input.Update(msg)
		return cmd
	}
}

// handlePaletteKeys handles keys when palette is open
func (m *Model) handlePaletteKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.paletteOpen = false
		m.palette.Blur()
		return nil
	case "enter":
		m.executePaletteCommand(m.palette.Value())
		m.palette.SetValue("")
		m.paletteOpen = false
		m.palette.Blur()
		return nil
	default:
		var cmd tea.Cmd
		m.palette, cmd = m.palette.Update(msg)
		m.paletteMsg = palettePreview(m.palette.Value())
		return cmd
	}
}

// handleFilterKeys handles keys when filter is focused
func (m *Model) handleFilterKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "enter":
		m.filter.Blur()
		m.applyFilterAndUpdateRows()
		return nil
	default:
		var cmd tea.Cmd
		m.filter, cmd = m.filter.Update(msg)
		m.applyFilterAndUpdateRows()
		return cmd
	}
}

// handleSessionListKeys handles keys in the main session list view
func (m *Model) handleSessionListKeys(msg tea.KeyMsg) tea.Cmd {
	var cmds []tea.Cmd

	switch msg.String() {
	case "q", "ctrl+c":
		return tea.Quit
	case "r":
		cmds = append(cmds, m.fetchSessionsCmd())
	case "/":
		m.filter.Focus()
	case ":":
		m.paletteOpen = true
		m.palette.Focus()
		m.paletteMsg = palettePreview("")
	case "tab":
		m.viewMode = ViewDashboard
		m.dashboardView.Focus()
		m.dashboardView.SetFilterValue("")
		m.dashboardView.SetCursor(0)
	case "?", "h":
		m.viewMode = ViewHelp
	case "p":
		m.viewMode = ViewProjects
		m.projectsView.Focus()
		m.projectsView.SetFilterValue("")
		m.projectsView.SetCursor(0)
	case "c":
		m.cfg.Redact = !m.cfg.Redact
		m.applyFilterAndUpdateRows()
	case "y":
		// Copy IDs
		ids := m.appState.SelectedIDs()
		if len(ids) > 0 {
			copyToClipboard(strings.Join(ids, "\n"))
		} else if s, ok := m.selectedSession(); ok {
			copyToClipboard(stripANSI(s.ID))
		}
	case "D":
		if s, ok := m.selectedSession(); ok {
			copyToClipboard(stripANSI(s.Detail))
		}
	case "o":
		// Open file - would need integration with app package
	case "s":
		m.cycleSort()
		m.applyFilterAndUpdateRows()
	case "g":
		m.cycleGroup()
		m.applyFilterAndUpdateRows()
	case "v":
		m.cycleViewMode()
		m.applyColumns()
		m.applyFilterAndUpdateRows()
	case "m":
		m.showLastCol = !m.showLastCol
		m.cfg.IncludeLastMsg = m.showLastCol
		m.applyColumns()
		cmds = append(cmds, m.fetchSessionsCmd())
	case "P":
		if s, ok := m.selectedSession(); ok {
			m.appState.TogglePin(stripANSI(s.ID))
			m.applyFilterAndUpdateRows()
		}
	case " ":
		if s, ok := m.selectedSession(); ok {
			m.appState.ToggleSelect(stripANSI(s.ID))
			m.applyFilterAndUpdateRows()
		}
	case "a":
		m.jumpToStatus(state.StatusApproval)
	case "u":
		m.jumpToStatus(state.StatusRunning)
	case "t":
		m.themeIndex = (m.themeIndex + 1) % len(theme.Themes)
		m.theme = theme.Themes[m.themeIndex]
		m.styles = theme.NewStyles(m.theme, m.accessible)
		m.dashboardView.SetStyles(m.styles)
		m.projectsView.SetStyles(m.styles)
		m.applyTableStyles()
	case "A":
		m.accessible = !m.accessible
		m.styles = theme.NewStyles(m.theme, m.accessible)
		m.dashboardView.SetStyles(m.styles)
		m.projectsView.SetStyles(m.styles)
		m.applyTableStyles()
	case "b":
		m.showSidebar = !m.showSidebar
		cmds = append(cmds, m.updateLayoutTargets())
	case "d":
		if m.detailMode == DetailSplit {
			m.detailMode = DetailFull
		} else {
			m.detailMode = DetailSplit
		}
		cmds = append(cmds, m.updateLayoutTargets())
	case "1":
		m.filters.ToggleProvider(state.ProviderClaude)
		m.applyFilterAndUpdateRows()
	case "2":
		m.filters.ToggleProvider(state.ProviderCodex)
		m.applyFilterAndUpdateRows()
	case "R":
		m.filters.ToggleStatus(state.StatusRunning)
		m.applyFilterAndUpdateRows()
	case "W":
		m.filters.ToggleStatus(state.StatusWaiting)
		m.applyFilterAndUpdateRows()
	case "E":
		m.filters.ToggleStatus(state.StatusApproval)
		m.applyFilterAndUpdateRows()
	case "S":
		m.filters.ToggleStatus(state.StatusStale)
		m.applyFilterAndUpdateRows()
	case "Z":
		m.filters.ToggleStatus(state.StatusEnded)
		m.applyFilterAndUpdateRows()
	case "N":
		m.filters.ToggleStatus(state.StatusNeedsAttn)
		m.applyFilterAndUpdateRows()
	}

	return tea.Batch(cmds...)
}

// View renders the UI
func (m *Model) View() string {
	var b strings.Builder

	// Header
	headerCfg := components.HeaderConfig{
		RefreshInterval: m.cfg.RefreshEvery,
		ActiveWindow:    m.cfg.ActiveWindow,
		Redact:          m.cfg.Redact,
		ThemeName:       m.theme.Name,
	}
	b.WriteString(components.RenderHeader(m.styles, headerCfg, m.width))
	b.WriteString("\n")

	// Filter bar
	filterLine := m.filter.View()
	if !m.filter.Focused() && m.filter.Value() == "" {
		filterLine = m.styles.Muted.Render(m.filter.Prompt + m.filter.Placeholder)
	}
	m.filters.TextQuery = m.filter.Value()
	m.filters.ParseQueryMode()
	filterInput := components.RenderFilterInput(filterLine, m.filters.QueryMode, m.filter.Focused(), m.styles)
	b.WriteString(filterInput)
	b.WriteString("\n")

	// Active filters
	b.WriteString(components.RenderActiveFilters(m.filters, len(m.appState.Selected), m.styles))
	b.WriteString("\n")

	// Palette
	if m.paletteOpen {
		pLine := m.palette.View()
		if m.palette.Value() == "" {
			pLine = m.styles.Muted.Render(m.palette.Prompt + m.palette.Placeholder)
		}
		b.WriteString(m.styles.Box.Render(pLine))
		b.WriteString("\n")
		b.WriteString(m.styles.Muted.Render(m.paletteMsg))
		b.WriteString("\n")
	}

	// Legend
	b.WriteString(components.RenderLegend(
		m.appState.FilterCounts,
		m.appState.FilterTotal,
		m.appState.FilterCost,
		string(m.effectiveMode),
		m.cfg.SortBy,
		m.cfg.GroupBy,
		m.theme.Name,
		m.styles,
	))
	b.WriteString("\n")

	// Context actions
	var context string
	if len(m.appState.Selected) > 0 {
		context = "selected"
	} else if m.paletteOpen {
		context = "palette"
	} else if m.viewMode == ViewProjects {
		context = "projects"
	} else if m.viewMode == ViewDashboard {
		context = "dashboard"
	}
	if actions := components.RenderContextBar(context, len(m.appState.Selected), m.styles); actions != "" {
		b.WriteString(actions)
		b.WriteString("\n")
	}

	// Main content area
	var listContent string

	switch m.viewMode {
	case ViewDashboard:
		listContent = m.dashboardView.Render(m.appState.AllSessions, m.appState.FilterCounts, m.appState.FilterCost)
	case ViewProjects:
		listContent = m.projectsView.Render(m.appState.ProjectItems)
	case ViewHelp:
		listContent = views.RenderHelpOverlay(m.styles)
	default:
		// Session list
		if m.appState.FilterTotal == 0 {
			listContent = components.RenderEmptyState(len(m.appState.AllSessions) > 0, m.filter.Value(), m.styles)
		} else {
			listContent = m.table.View()
		}

		// Add sidebar
		if m.showSidebar && m.sidebarWidth > 0 {
			sidebarData := components.BuildSidebarData(m.appState.AllSessions, m.filters)
			sidebar := components.RenderSidebar(sidebarData, m.styles, m.sidebarWidth)
			listContent = lipgloss.JoinHorizontal(lipgloss.Top, sidebar, listContent)
		}
	}

	// Detail mode handling
	if m.detailMode == DetailFull && m.viewMode == ViewSessions {
		b.WriteString(m.styles.DetailBox.Render(m.detailView()))
		b.WriteString("\n")
		b.WriteString(m.shortcutsBar())
		return b.String()
	}

	// Split view
	if m.splitActive() && m.detailWidth > 0 && m.viewMode == ViewSessions {
		listPane := lipgloss.NewStyle().Width(m.listPaneWidth()).Render(listContent)
		detailPane := m.styles.DetailBox.Render(m.detailView())
		detailPane = lipgloss.NewStyle().Width(m.detailWidth).Render(detailPane)
		gap := strings.Repeat(" ", SplitGap)
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, listPane, gap, detailPane))
		b.WriteString("\n")
		b.WriteString(m.shortcutsBar())
		if m.err != nil {
			b.WriteString("\n")
			b.WriteString(components.RenderError(m.err, m.styles))
		}
		return b.String()
	}

	// Standard layout
	b.WriteString(listContent)
	b.WriteString("\n")

	if m.viewMode == ViewSessions {
		b.WriteString(m.styles.DetailBox.Render(m.detailView()))
		b.WriteString("\n")
	}

	b.WriteString(m.shortcutsBar())

	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(components.RenderError(m.err, m.styles))
	}

	return b.String()
}

// Helper methods

func (m *Model) fetchSessionsCmd() tea.Cmd {
	if m.sessionFetcher == nil {
		return nil
	}
	fetcher := m.sessionFetcher
	return func() tea.Msg {
		sessions, err := fetcher()
		return SessionsMsg{Sessions: sessions, Err: err}
	}
}

func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(m.cfg.RefreshEvery, func(t time.Time) tea.Msg { return TickMsg(t) })
}

func (m *Model) animCmd() tea.Cmd {
	return tea.Tick(AnimInterval, func(t time.Time) tea.Msg { return AnimMsg(t) })
}

func (m *Model) selectedSession() (state.SessionView, bool) {
	if len(m.appState.RowMeta) == 0 {
		return state.SessionView{}, false
	}
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.appState.RowMeta) {
		return state.SessionView{}, false
	}
	meta := m.appState.RowMeta[idx]
	if meta.Kind != state.RowSession || meta.Session == nil {
		return state.SessionView{}, false
	}
	return *meta.Session, true
}

func (m *Model) detailView() string {
	if m.viewMode == ViewDashboard {
		if proj := m.dashboardView.SelectedProject(m.appState.AllSessions); proj != nil {
			return components.RenderProjectDetail(*proj, m.styles)
		}
		return "No project selected."
	}
	if s, ok := m.selectedSession(); ok {
		return components.RenderSessionDetail(s, m.styles)
	}
	if m.err != nil {
		return "Error: " + m.err.Error()
	}
	return "No session selected."
}

func (m *Model) shortcutsBar() string {
	var shortcuts []components.Shortcut
	switch m.viewMode {
	case ViewDashboard:
		shortcuts = components.ShortcutsDashboard
	case ViewProjects:
		shortcuts = components.ShortcutsProjects
	case ViewHelp:
		shortcuts = components.ShortcutsHelp
	default:
		shortcuts = components.ShortcutsSessionList
	}
	return components.RenderShortcutBar(shortcuts, m.styles, m.width)
}

func (m *Model) splitActive() bool {
	if m.detailMode != DetailSplit {
		return false
	}
	return m.width >= SplitMinWidth && (m.detailTarget > 0 || m.detailWidth > 0)
}

func (m *Model) listPaneWidth() int {
	width := m.width
	if m.splitActive() && m.detailWidth > 0 {
		width -= m.detailWidth + SplitGap
	}
	if width < 20 {
		return 20
	}
	return width
}

func (m *Model) tableWidth() int {
	width := m.listPaneWidth()
	if m.sidebarWidth > 0 {
		width -= m.sidebarWidth
	}
	if width < 20 {
		return 20
	}
	return width
}

func (m *Model) updateLayoutTargets() tea.Cmd {
	sidebarTarget := 0
	if m.showSidebar && m.width >= 100 {
		sidebarTarget = SidebarMaxWidth
	}

	detailTarget := 0
	if m.detailMode == DetailSplit && m.width >= SplitMinWidth {
		available := m.width - SplitGap
		detailTarget = widgets.ClampInt(available/3, 32, 60)
		if available-detailTarget-sidebarTarget < SplitMinListWidth {
			detailTarget = 0
		}
	}

	m.sidebarTarget = sidebarTarget
	m.detailTarget = detailTarget

	if m.sidebarWidth != m.sidebarTarget || m.detailWidth != m.detailTarget {
		return m.animCmd()
	}
	return nil
}

func (m *Model) stepLayoutAnimation() bool {
	changed := false
	if m.sidebarWidth != m.sidebarTarget {
		m.sidebarWidth = stepToward(m.sidebarWidth, m.sidebarTarget, AnimStep)
		changed = true
	}
	if m.detailWidth != m.detailTarget {
		m.detailWidth = stepToward(m.detailWidth, m.detailTarget, AnimStep)
		changed = true
	}
	if changed {
		m.applyColumns()
	}
	return m.sidebarWidth != m.sidebarTarget || m.detailWidth != m.detailTarget
}

func stepToward(cur, target, step int) int {
	if cur == target || step <= 0 {
		return target
	}
	if cur < target {
		cur += step
		if cur > target {
			cur = target
		}
		return cur
	}
	cur -= step
	if cur < target {
		cur = target
	}
	return cur
}

func (m *Model) applyColumns() {
	mode := m.columnMode
	if mode != ColumnModeCard {
		if m.tableWidth() < 80 {
			mode = ColumnModeUltra
		} else if m.tableWidth() < 100 && mode == ColumnModeFull {
			mode = ColumnModeCompact
		}
	}
	m.effectiveMode = mode
	cols, idCol := columnsFor(m.tableWidth(), mode, m.showLastCol)
	m.table.SetColumns(cols)
	m.idColumn = idCol
}

func (m *Model) applyTableStyles() {
	styles := table.DefaultStyles()
	styles.Selected = styles.Selected.Foreground(lipgloss.Color(m.theme.Accent)).Bold(true)
	if !m.accessible {
		styles.Selected = styles.Selected.Background(lipgloss.Color(m.theme.Crust))
	}
	m.table.SetStyles(styles)
}

func (m *Model) applyFilterAndUpdateRows() {
	// Apply filters
	m.filters.TextQuery = m.filter.Value()
	m.filters.ParseQueryMode()
	filtered := m.filters.ApplyToSessions(m.appState.AllSessions)

	// Sort
	sortSessions(filtered, m.cfg.SortBy)

	// Apply pinned first
	filtered = m.appState.ApplyPinnedFirst(filtered)

	m.appState.FilteredSessions = filtered
	m.appState.FilterCounts = make(map[state.Status]int)
	m.appState.FilterCost = 0
	m.appState.FilterTotal = len(filtered)
	for _, s := range filtered {
		m.appState.FilterCounts[s.Status]++
		m.appState.FilterCost += s.Cost
	}

	// Build rows
	rows := make([]table.Row, 0, len(filtered))
	m.appState.RowMeta = nil

	if m.cfg.GroupBy == "" {
		for i := range filtered {
			s := &filtered[i]
			rows = append(rows, m.rowForSession(s))
			m.appState.RowMeta = append(m.appState.RowMeta, state.RowMeta{Kind: state.RowSession, Session: s})
		}
	} else {
		groups := groupSessions(filtered, m.cfg.GroupBy)
		for _, g := range groups {
			groupLabel := g.Group
			if strings.TrimSpace(groupLabel) == "" {
				groupLabel = "unknown"
			}
			rows = append(rows, m.groupRow(groupLabel))
			m.appState.RowMeta = append(m.appState.RowMeta, state.RowMeta{Kind: state.RowGroup, Group: groupLabel})
			for i := range g.Sessions {
				s := g.Sessions[i]
				rows = append(rows, m.rowForSession(&s))
				m.appState.RowMeta = append(m.appState.RowMeta, state.RowMeta{Kind: state.RowSession, Session: &s})
			}
		}
	}

	m.table.SetRows(rows)
	m.ensureCursorOnSession()
}

func (m *Model) rowForSession(s *state.SessionView) table.Row {
	id := s.ID
	idKey := stripANSI(id)
	if m.appState.IsRecentlyChanged(idKey, m.cfg.RefreshEvery) {
		id = m.styles.Changed.Render(id)
	}

	prefix := m.prefixForSession(*s)
	status := widgets.StatusBadge(s.Status, m.styles)
	age := widgets.FormatAgo(s.Age)
	cost := widgets.FormatCost(s.Cost)
	since := widgets.FormatSince(s.LastSeen)
	mode := m.effectiveMode

	if mode == ColumnModeCard {
		return table.Row{m.renderCard(*s)}
	}

	if mode == ColumnModeUltra || m.width < 80 {
		return table.Row{prefix, status, s.Project, age}
	}

	if mode != ColumnModeFull || m.width < 100 {
		return table.Row{prefix, status, s.Project, age, cost, id}
	}

	row := table.Row{prefix, status, s.Project, age, cost, id, s.Model, s.Dir, since}
	if m.showLastCol {
		row = append(row, lastSnippet(*s))
	}
	return row
}

func (m *Model) groupRow(label string) table.Row {
	cols := m.table.Columns()
	row := make(table.Row, len(cols))
	if len(cols) > 0 {
		row[0] = m.styles.GroupHeader.Render("══")
	}
	if m.idColumn >= 0 && m.idColumn < len(cols) {
		row[m.idColumn] = m.styles.GroupHeader.Render(label)
	}
	return row
}

func (m *Model) prefixForSession(s state.SessionView) string {
	idKey := stripANSI(s.ID)
	pinned := m.appState.Pinned[idKey]
	selected := m.appState.Selected[idKey]

	pin := " "
	if pinned {
		pin = "★"
	}
	box := " "
	if selected {
		box = "●"
	}
	icon := widgets.ProviderIconEmoji(s.Provider)
	if m.appState.IsRecentlyChanged(idKey, m.cfg.RefreshEvery) {
		icon = m.styles.Changed.Render(icon)
	}
	return fmt.Sprintf("%s%s%s", pin, box, icon)
}

func (m *Model) renderCard(s state.SessionView) string {
	status := chip(widgets.StatusIcon(s.Status)+" "+strings.ToUpper(string(s.Status)), m.theme.Accent, m.theme.Crust)
	provider := chip(strings.ToUpper(string(s.Provider)), m.theme.Accent, m.theme.Crust)
	project := s.Project
	if project == "" {
		project = "unknown"
	}
	projectChip := chip(project, m.theme.Overlay0, m.theme.Crust)

	title := fmt.Sprintf("%s %s %s", provider, status, projectChip)
	meta := fmt.Sprintf("%s  %s  %s", s.Model, widgets.FormatAgo(s.Age), widgets.FormatSince(s.LastSeen))
	dir := s.Dir
	if dir == "" {
		dir = "-"
	}
	body := fmt.Sprintf("dir: %s", dir)

	var lines []string
	lines = append(lines, title)
	lines = append(lines, meta)
	lines = append(lines, body)
	if m.showLastCol {
		last := lastSnippet(s)
		if last != "" {
			lines = append(lines, last)
		}
	}
	return m.styles.Card.Render(strings.Join(lines, "\n"))
}

func chip(text string, bg, fg lipgloss.Color) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}
	return lipgloss.NewStyle().
		Background(bg).
		Foreground(fg).
		Padding(0, 1).
		Render(text)
}

func lastSnippet(s state.SessionView) string {
	if s.LastUser != "" {
		return widgets.TruncateString("u: "+s.LastUser, 22)
	}
	if s.LastAssist != "" {
		return widgets.TruncateString("a: "+s.LastAssist, 22)
	}
	return ""
}

func (m *Model) ensureCursorOnSession() {
	if len(m.appState.RowMeta) == 0 {
		return
	}
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.appState.RowMeta) || m.appState.RowMeta[idx].Kind == state.RowSession {
		return
	}
	for i, meta := range m.appState.RowMeta {
		if meta.Kind == state.RowSession {
			m.table.SetCursor(i)
			return
		}
	}
}

func (m *Model) cycleSort() {
	order := []string{"last_seen", "status", "provider", "cost", "project"}
	idx := indexOf(order, m.cfg.SortBy)
	if idx < 0 {
		idx = 0
	}
	m.cfg.SortBy = order[(idx+1)%len(order)]
}

func (m *Model) cycleGroup() {
	order := []string{"", "provider", "project", "status", "day", "hour"}
	idx := indexOf(order, m.cfg.GroupBy)
	if idx < 0 {
		idx = 0
	}
	m.cfg.GroupBy = order[(idx+1)%len(order)]
}

func (m *Model) cycleViewMode() {
	order := []ColumnMode{ColumnModeFull, ColumnModeCompact, ColumnModeUltra, ColumnModeCard}
	idx := -1
	for i, mode := range order {
		if mode == m.columnMode {
			idx = i
			break
		}
	}
	if idx < 0 {
		idx = 0
	}
	m.columnMode = order[(idx+1)%len(order)]
}

func (m *Model) jumpToStatus(status state.Status) {
	for i, meta := range m.appState.RowMeta {
		if meta.Kind == state.RowSession && meta.Session != nil && meta.Session.Status == status {
			m.table.SetCursor(i)
			return
		}
	}
}

func indexOf(list []string, val string) int {
	for i, v := range list {
		if v == val {
			return i
		}
	}
	return -1
}

// Utility functions

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

func copyToClipboard(text string) error {
	if runtime.GOOS == "darwin" {
		cmd := exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	}
	return nil
}

func columnsFor(width int, mode ColumnMode, showLast bool) ([]table.Column, int) {
	if width <= 0 {
		width = 120
	}
	if mode == ColumnModeCard {
		return []table.Column{{Title: "SESSION", Width: widgets.MaxInt(20, width-4)}}, 0
	}

	if mode == ColumnModeUltra || width < 80 {
		cols := []table.Column{
			{Title: " ", Width: 4},
			{Title: "STATUS", Width: 8},
			{Title: "PROJECT", Width: 18},
			{Title: "AGE", Width: 5},
		}
		return cols, 2
	}

	compact := mode != ColumnModeFull || width < 100

	iconCol := table.Column{Title: " ", Width: 4}
	statusCol := table.Column{Title: "STATUS", Width: 8}
	projectCol := table.Column{Title: "PROJECT", Width: 18}
	ageCol := table.Column{Title: "AGE", Width: 5}
	costCol := table.Column{Title: "COST", Width: 8}
	idCol := table.Column{Title: "ID", Width: 14}

	if compact {
		cols := []table.Column{iconCol, statusCol, projectCol, ageCol, costCol, idCol}
		return cols, 2
	}

	modelCol := table.Column{Title: "MODEL", Width: 12}
	sinceCol := table.Column{Title: "SINCE", Width: 14}
	dirCol := table.Column{Title: "DIR", Width: 26}

	cols := []table.Column{iconCol, statusCol, projectCol, ageCol, costCol, idCol, modelCol, dirCol, sinceCol}
	idIndex := 2
	if showLast {
		lastCol := table.Column{Title: "LAST", Width: 20}
		cols = append(cols, lastCol)
	}

	fixed := 0
	for _, c := range cols {
		fixed += c.Width
	}
	extra := width - fixed
	if extra > 0 {
		for i := range cols {
			if cols[i].Title == "DIR" {
				cols[i].Width += extra
				break
			}
		}
	}
	return cols, idIndex
}

func sortSessions(views []state.SessionView, sortBy string) {
	sortKey := strings.ToLower(strings.TrimSpace(sortBy))
	if sortKey == "" {
		sortKey = "last_seen"
	}

	for i := 0; i < len(views); i++ {
		for j := i + 1; j < len(views); j++ {
			swap := false
			a := views[i]
			b := views[j]

			switch sortKey {
			case "status":
				if string(a.Status) > string(b.Status) {
					swap = true
				} else if string(a.Status) == string(b.Status) && a.LastSeen.Before(b.LastSeen) {
					swap = true
				}
			case "provider":
				if string(a.Provider) > string(b.Provider) {
					swap = true
				} else if string(a.Provider) == string(b.Provider) && a.LastSeen.Before(b.LastSeen) {
					swap = true
				}
			case "cost":
				if a.Cost < b.Cost {
					swap = true
				} else if a.Cost == b.Cost && a.LastSeen.Before(b.LastSeen) {
					swap = true
				}
			case "project":
				if strings.ToLower(a.Project) > strings.ToLower(b.Project) {
					swap = true
				} else if strings.ToLower(a.Project) == strings.ToLower(b.Project) && a.LastSeen.Before(b.LastSeen) {
					swap = true
				}
			default:
				// last_seen
				if a.LastSeen.Before(b.LastSeen) {
					swap = true
				}
			}

			if swap {
				views[i], views[j] = views[j], views[i]
			}
		}
	}
}

type sessionGroup struct {
	Group    string
	Sessions []state.SessionView
}

func groupSessions(views []state.SessionView, groupBy string) []sessionGroup {
	groupKey := strings.ToLower(strings.TrimSpace(groupBy))
	if groupKey == "" {
		return []sessionGroup{{Group: "", Sessions: views}}
	}

	order := []string{}
	groups := map[string][]state.SessionView{}
	for _, v := range views {
		key := ""
		switch groupKey {
		case "provider":
			key = string(v.Provider)
		case "project":
			key = v.Project
		case "status":
			key = string(v.Status)
		case "day":
			key = v.LastSeen.Local().Format("2006-01-02")
		case "hour":
			key = v.LastSeen.Local().Format("2006-01-02 15:00")
		}
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], v)
	}

	out := make([]sessionGroup, 0, len(order))
	for _, k := range order {
		out = append(out, sessionGroup{Group: k, Sessions: groups[k]})
	}
	return out
}

func (m *Model) executePaletteCommand(raw string) {
	cmdLine := strings.TrimSpace(raw)
	if cmdLine == "" {
		return
	}
	parts := strings.Fields(cmdLine)
	if len(parts) == 0 {
		return
	}
	cmd := resolvePaletteCommand(strings.ToLower(parts[0]))
	arg := ""
	if len(parts) > 1 {
		arg = strings.ToLower(strings.Join(parts[1:], " "))
	}

	switch cmd {
	case "help", "?":
		m.paletteMsg = "Commands: dashboard, projects, clear-filters, reset-view, show, open, copy-id, copy-detail, sort <key>, group <key>, view <full|compact>, theme, last-msg <on|off>"
	case "show", "detail":
		m.detailMode = DetailFull
		m.paletteMsg = "Detail view: full"
	case "dashboard":
		m.viewMode = ViewDashboard
		m.dashboardView.Focus()
		m.dashboardView.SetFilterValue("")
		m.dashboardView.SetCursor(0)
		m.paletteMsg = "Dashboard"
	case "projects":
		m.viewMode = ViewProjects
		m.projectsView.Focus()
		m.projectsView.SetFilterValue("")
		m.projectsView.SetCursor(0)
		m.paletteMsg = "Projects"
	case "clear-filters":
		m.filters.Clear()
		m.filter.SetValue("")
		m.paletteMsg = "Filters cleared"
		m.applyFilterAndUpdateRows()
	case "reset-view":
		m.columnMode = ColumnModeFull
		m.showLastCol = false
		m.cfg.IncludeLastMsg = false
		m.cfg.SortBy = "last_seen"
		m.cfg.GroupBy = ""
		m.showSidebar = true
		m.detailMode = DetailSplit
		m.updateLayoutTargets()
		m.sidebarWidth = m.sidebarTarget
		m.detailWidth = m.detailTarget
		m.paletteMsg = "View reset"
		m.applyFilterAndUpdateRows()
	case "sort":
		if arg != "" {
			m.cfg.SortBy = arg
		}
		m.paletteMsg = "Sort: " + m.cfg.SortBy
		m.applyFilterAndUpdateRows()
	case "group":
		if arg != "" {
			m.cfg.GroupBy = arg
		}
		g := m.cfg.GroupBy
		if g == "" {
			g = "none"
		}
		m.paletteMsg = "Group: " + g
		m.applyFilterAndUpdateRows()
	case "view":
		switch arg {
		case "compact":
			m.columnMode = ColumnModeCompact
		case "full":
			m.columnMode = ColumnModeFull
		case "ultra":
			m.columnMode = ColumnModeUltra
		case "card":
			m.columnMode = ColumnModeCard
		default:
			// Toggle
			m.cycleViewMode()
		}
		m.applyColumns()
		m.applyFilterAndUpdateRows()
	case "theme":
		m.themeIndex = (m.themeIndex + 1) % len(theme.Themes)
		m.theme = theme.Themes[m.themeIndex]
		m.styles = theme.NewStyles(m.theme, m.accessible)
		m.dashboardView.SetStyles(m.styles)
		m.projectsView.SetStyles(m.styles)
		m.applyTableStyles()
		m.paletteMsg = "Theme: " + m.theme.Name
	case "last-msg":
		if arg == "on" {
			m.showLastCol = true
		} else if arg == "off" {
			m.showLastCol = false
		} else {
			m.showLastCol = !m.showLastCol
		}
		m.cfg.IncludeLastMsg = m.showLastCol
		m.applyColumns()
		m.applyFilterAndUpdateRows()
	default:
		m.paletteMsg = "Unknown command"
	}
}

func resolvePaletteCommand(input string) string {
	commands := []string{"dashboard", "projects", "clear-filters", "reset-view", "show", "detail", "open", "copy-id", "copy-detail", "sort", "group", "view", "theme", "last-msg", "help", "?"}
	for _, c := range commands {
		if c == input {
			return c
		}
	}
	for _, c := range commands {
		if strings.HasPrefix(c, input) {
			return c
		}
	}
	return input
}

func palettePreview(raw string) string {
	cmdLine := strings.TrimSpace(raw)
	if cmdLine == "" {
		return "Palette: dashboard, projects, clear-filters, reset-view, show, open, copy-id, copy-detail, sort <key>, group <key>, view <mode>, theme, last-msg <on|off>"
	}
	parts := strings.Fields(cmdLine)
	cmd := resolvePaletteCommand(strings.ToLower(parts[0]))
	switch cmd {
	case "sort":
		return "Sort by: last_seen | status | provider | cost | project"
	case "group":
		return "Group by: provider | project | status | day | hour"
	case "view":
		return "View: full | compact | ultra | card"
	case "last-msg":
		return "Toggle last message column"
	case "show":
		return "Expand detail view"
	case "open":
		return "Open selected transcript/log"
	case "copy-id":
		return "Copy selected IDs"
	case "copy-detail":
		return "Copy detail panel"
	case "dashboard":
		return "Open projects dashboard"
	case "projects":
		return "Open project picker"
	case "clear-filters":
		return "Clear all filters"
	case "reset-view":
		return "Reset view defaults"
	}
	return "Press Enter to run"
}
