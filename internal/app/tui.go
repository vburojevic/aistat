package app

import (
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// -------------------------
// TUI
// -------------------------

type sessionsMsg struct {
	Sessions []SessionView
	Err      error
}

type tickMsg time.Time

type rowKind int

const (
	rowSession rowKind = iota
	rowGroup
)

type rowMeta struct {
	kind    rowKind
	session *SessionView
	group   string
}

type tuiKeyMap struct {
	UpDown        key.Binding
	Quit          key.Binding
	Refresh       key.Binding
	Filter        key.Binding
	Palette       key.Binding
	ToggleRedact  key.Binding
	CopyID        key.Binding
	CopyDetail    key.Binding
	OpenFile      key.Binding
	ToggleSort    key.Binding
	ToggleGroup   key.Binding
	ToggleView    key.Binding
	ToggleLast    key.Binding
	TogglePin     key.Binding
	ToggleSelect  key.Binding
	JumpApproval  key.Binding
	JumpRunning   key.Binding
	ToggleTheme   key.Binding
	ToggleAccess  key.Binding
	ToggleSidebar key.Binding
	ToggleDetail  key.Binding
	ToggleClaude  key.Binding
	ToggleCodex   key.Binding
	ToggleProj    key.Binding
	ToggleRun     key.Binding
	ToggleWait    key.Binding
	ToggleAppr    key.Binding
	ToggleStale   key.Binding
	ToggleEnded   key.Binding
	ToggleAttn    key.Binding
}

func (k tuiKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.Refresh, k.Filter, k.Palette, k.ToggleGroup, k.ToggleSort}
}
func (k tuiKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Quit, k.Refresh, k.Filter, k.Palette, k.ToggleGroup},
		{k.ToggleSort, k.ToggleView, k.ToggleLast, k.ToggleDetail},
		{k.CopyID, k.CopyDetail, k.OpenFile, k.TogglePin, k.ToggleSelect},
		{k.JumpApproval, k.JumpRunning, k.ToggleTheme, k.ToggleAccess, k.ToggleSidebar},
	}
}

type tuiTheme struct {
	name   string
	run    string
	wait   string
	appr   string
	stale  string
	muted  string
	accent string
}

var themes = []tuiTheme{
	{name: "dark", run: "2", wait: "3", appr: "1", stale: "8", muted: "8", accent: "6"},
	{name: "light", run: "10", wait: "11", appr: "9", stale: "7", muted: "7", accent: "4"},
	{name: "hc", run: "2", wait: "3", appr: "1", stale: "15", muted: "15", accent: "5"},
}

type detailMode int

const (
	detailSplit detailMode = iota
	detailFull
)

type tuiModel struct {
	cfg Config

	table   table.Model
	filter  textinput.Model
	palette textinput.Model
	help    help.Model
	keys    tuiKeyMap

	width  int
	height int

	allSessions []SessionView
	rowMeta     []rowMeta
	idColumn    int

	columnMode    string
	showLastCol   bool
	showSidebar   bool
	modeDetail    detailMode
	helpVisible   bool
	paletteOpen   bool
	paletteMsg    string
	effectiveMode string

	selected map[string]bool
	pinned   map[string]bool

	providerFilter map[Provider]bool
	statusFilter   map[Status]bool
	projectFilter  map[string]bool

	filteredSessions []SessionView
	filterCounts     map[Status]int
	filterCost       float64
	filterTotal      int

	changedAt    map[string]time.Time
	lastSnapshot map[string]SessionView
	history      map[string][]time.Time
	lastOrder    map[string]int
	moveDir      map[string]int

	err         error
	lastRefresh time.Time
	themeIndex  int
	accessible  bool
}

var (
	styleTitle       = lipgloss.NewStyle().Bold(true)
	styleMuted       = lipgloss.NewStyle().Faint(true)
	styleBox         = lipgloss.NewStyle().Padding(0, 1)
	styleDetailBox   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	styleHeaderBox   = lipgloss.NewStyle().Padding(0, 1)
	styleBadge       = lipgloss.NewStyle().Bold(true).Padding(0, 1)
	styleBadgeRun    = lipgloss.NewStyle()
	styleBadgeWait   = lipgloss.NewStyle()
	styleBadgeAppr   = lipgloss.NewStyle()
	styleBadgeStale  = lipgloss.NewStyle()
	styleGroupHeader = lipgloss.NewStyle().Bold(true)
	styleChanged     = lipgloss.NewStyle().Bold(true)
	styleCard        = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
)

func applyTheme(t tuiTheme, accessible bool) {
	styleTitle = lipgloss.NewStyle().Bold(true)
	styleMuted = lipgloss.NewStyle().Foreground(lipgloss.Color(t.muted))
	styleBox = lipgloss.NewStyle().Padding(0, 1)
	styleDetailBox = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	styleHeaderBox = lipgloss.NewStyle().Padding(0, 1)
	styleBadge = lipgloss.NewStyle().Bold(true).Padding(0, 1)
	styleGroupHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(t.accent))
	styleChanged = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(t.accent))

	styleBadgeRun = styleBadge.Copy().Foreground(lipgloss.Color("0")).Background(lipgloss.Color(t.run))
	styleBadgeWait = styleBadge.Copy().Foreground(lipgloss.Color("0")).Background(lipgloss.Color(t.wait))
	styleBadgeAppr = styleBadge.Copy().Foreground(lipgloss.Color("0")).Background(lipgloss.Color(t.appr))
	styleBadgeStale = styleBadge.Copy().Foreground(lipgloss.Color("0")).Background(lipgloss.Color(t.stale))

	if accessible {
		styleMuted = lipgloss.NewStyle()
		styleHeaderBox = lipgloss.NewStyle().Padding(1, 2)
		styleDetailBox = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(1, 2)
		styleBadgeRun = styleBadge.Copy().Underline(true)
		styleBadgeWait = styleBadge.Copy().Underline(true)
		styleBadgeAppr = styleBadge.Copy().Underline(true)
		styleBadgeStale = styleBadge.Copy().Underline(true)
		styleGroupHeader = lipgloss.NewStyle().Bold(true)
		styleChanged = lipgloss.NewStyle().Bold(true)
		styleCard = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1)
	}
}

func (m *tuiModel) applyTableStyles() {
	styles := table.DefaultStyles()
	accent := themes[m.themeIndex].accent
	styles.Selected = styles.Selected.Foreground(lipgloss.Color(accent)).Bold(true)
	if !m.accessible {
		styles.Selected = styles.Selected.Background(lipgloss.Color("0"))
	}
	m.table.SetStyles(styles)
}

func runTUI(cfg Config) error {
	cols, idCol := columnsFor(120, "full", false)
	applyTheme(themes[0], false)

	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
	)
	_ = idCol

	// styles set after model init to honor theme/accessibility

	f := textinput.New()
	f.Placeholder = "filter (/, esc)"
	f.Prompt = "🔎 "
	f.CharLimit = 128
	f.Width = 40

	pal := textinput.New()
	pal.Placeholder = "command (help)"
	pal.Prompt = ": "
	pal.CharLimit = 128
	pal.Width = 50

	km := tuiKeyMap{
		UpDown:        key.NewBinding(key.WithKeys("up", "down", "j", "k"), key.WithHelp("↑/↓", "move")),
		Quit:          key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Refresh:       key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Filter:        key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Palette:       key.NewBinding(key.WithKeys(":"), key.WithHelp(":", "palette")),
		ToggleRedact:  key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "toggle redact")),
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
		ToggleProj:    key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "toggle project")),
		ToggleRun:     key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "running")),
		ToggleWait:    key.NewBinding(key.WithKeys("W"), key.WithHelp("W", "waiting")),
		ToggleAppr:    key.NewBinding(key.WithKeys("E"), key.WithHelp("E", "approval")),
		ToggleStale:   key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "stale")),
		ToggleEnded:   key.NewBinding(key.WithKeys("Z"), key.WithHelp("Z", "ended")),
		ToggleAttn:    key.NewBinding(key.WithKeys("N"), key.WithHelp("N", "attn")),
	}

	m := &tuiModel{
		cfg:            cfg,
		table:          t,
		filter:         f,
		palette:        pal,
		help:           help.New(),
		keys:           km,
		columnMode:     "full",
		showLastCol:    false,
		showSidebar:    true,
		modeDetail:     detailSplit,
		selected:       map[string]bool{},
		pinned:         map[string]bool{},
		providerFilter: map[Provider]bool{},
		statusFilter:   map[Status]bool{},
		projectFilter:  map[string]bool{},
		filterCounts:   map[Status]int{},
		changedAt:      map[string]time.Time{},
		lastSnapshot:   map[string]SessionView{},
		history:        map[string][]time.Time{},
		lastOrder:      map[string]int{},
		moveDir:        map[string]int{},
		themeIndex:     0,
		accessible:     false,
	}
	m.idColumn = idCol
	m.applyTableStyles()

	// Initialize filters from config.
	if cfg.ProviderFilter != "" {
		m.providerFilter[Provider(cfg.ProviderFilter)] = true
	}
	for _, s := range cfg.StatusFilters {
		m.statusFilter[s] = true
	}
	for _, p := range cfg.ProjectFilters {
		m.projectFilter[strings.ToLower(p)] = true
	}

	prog := tea.NewProgram(m, tea.WithAltScreen())
	_, err := prog.Run()
	return err
}

func (m *tuiModel) Init() tea.Cmd {
	return tea.Batch(fetchSessionsCmd(m.cfg), tickCmd(m.cfg.RefreshEvery))
}

func fetchSessionsCmd(cfg Config) tea.Cmd {
	return func() tea.Msg {
		s, err := gatherSessions(cfg)
		return sessionsMsg{Sessions: s, Err: err}
	}
}

func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m *tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.filter.Width = minInt(60, maxInt(20, m.width-20))
		m.applyColumns()
		tableHeight := maxInt(5, m.height-2-1-9-1)
		m.table.SetHeight(tableHeight)

	case sessionsMsg:
		m.err = msg.Err
		if msg.Err == nil {
			m.markChanges(msg.Sessions)
			m.allSessions = msg.Sessions
			m.lastRefresh = time.Now().UTC()
			m.applyFilterAndUpdateRows()
		}
	case tickMsg:
		cmds = append(cmds, fetchSessionsCmd(m.cfg), tickCmd(m.cfg.RefreshEvery))

	case tea.KeyMsg:
		if m.paletteOpen {
			switch msg.String() {
			case "esc":
				m.paletteOpen = false
				m.palette.Blur()
				return m, nil
			case "enter":
				m.executePaletteCommand(m.palette.Value())
				m.palette.SetValue("")
				m.paletteOpen = false
				m.palette.Blur()
				return m, nil
			default:
				var cmd tea.Cmd
				m.palette, cmd = m.palette.Update(msg)
				m.paletteMsg = palettePreview(m.palette.Value())
				return m, cmd
			}
		}

		if m.filter.Focused() {
			switch msg.String() {
			case "esc":
				m.filter.Blur()
				m.applyFilterAndUpdateRows()
				return m, nil
			case "enter":
				m.filter.Blur()
				m.applyFilterAndUpdateRows()
				return m, nil
			default:
				var cmd tea.Cmd
				m.filter, cmd = m.filter.Update(msg)
				cmds = append(cmds, cmd)
				m.applyFilterAndUpdateRows()
				return m, tea.Batch(cmds...)
			}
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			cmds = append(cmds, fetchSessionsCmd(m.cfg))
		case "/":
			m.filter.Focus()
		case ":":
			m.paletteOpen = true
			m.palette.Focus()
			m.paletteMsg = palettePreview("")
		case "c":
			m.cfg.Redact = !m.cfg.Redact
			m.applyFilterAndUpdateRows()
		case "y":
			ids := m.selectedIDs()
			if len(ids) > 0 {
				_ = copyToClipboard(strings.Join(ids, "\n"))
			} else if s, ok := m.selectedSession(); ok {
				_ = copyToClipboard(stripANSI(s.ID))
			}
		case "D":
			if s, ok := m.selectedSession(); ok {
				_ = copyToClipboard(stripANSI(s.Detail))
			}
		case "o":
			if s, ok := m.selectedSession(); ok {
				_ = openSourceForSession(s.Provider, stripANSI(s.ID))
			}
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
			cmds = append(cmds, fetchSessionsCmd(m.cfg))
			m.applyFilterAndUpdateRows()
		case "P":
			m.togglePin()
			m.applyFilterAndUpdateRows()
		case " ":
			m.toggleSelect()
			m.applyFilterAndUpdateRows()
		case "a":
			m.jumpToStatus(StatusApproval)
		case "u":
			m.jumpToStatus(StatusRunning)
		case "t":
			m.themeIndex = (m.themeIndex + 1) % len(themes)
			applyTheme(themes[m.themeIndex], m.accessible)
			m.applyTableStyles()
		case "A":
			m.accessible = !m.accessible
			applyTheme(themes[m.themeIndex], m.accessible)
			m.applyTableStyles()
		case "b":
			m.showSidebar = !m.showSidebar
		case "d":
			if m.modeDetail == detailSplit {
				m.modeDetail = detailFull
			} else {
				m.modeDetail = detailSplit
			}
		case "1":
			m.toggleProviderFilter(ProviderClaude)
			m.applyFilterAndUpdateRows()
		case "2":
			m.toggleProviderFilter(ProviderCodex)
			m.applyFilterAndUpdateRows()
		case "p":
			m.toggleProjectFilter()
			m.applyFilterAndUpdateRows()
		case "R":
			m.toggleStatusFilter(StatusRunning)
			m.applyFilterAndUpdateRows()
		case "W":
			m.toggleStatusFilter(StatusWaiting)
			m.applyFilterAndUpdateRows()
		case "E":
			m.toggleStatusFilter(StatusApproval)
			m.applyFilterAndUpdateRows()
		case "S":
			m.toggleStatusFilter(StatusStale)
			m.applyFilterAndUpdateRows()
		case "Z":
			m.toggleStatusFilter(StatusEnded)
			m.applyFilterAndUpdateRows()
		case "N":
			m.toggleStatusFilter(StatusNeedsAttn)
			m.applyFilterAndUpdateRows()
		}
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *tuiModel) applyColumns() {
	mode := m.columnMode
	if mode != "card" {
		if m.width < 80 {
			mode = "ultra"
		} else if m.width < 100 && mode == "full" {
			mode = "compact"
		}
	}
	m.effectiveMode = mode
	cols, idCol := columnsFor(m.width, mode, m.showLastCol)
	m.table.SetColumns(cols)
	m.idColumn = idCol
}

func (m *tuiModel) executePaletteCommand(raw string) {
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
		m.paletteMsg = "Commands: show, open, copy-id, copy-detail, sort <key>, group <key>, view <full|compact>, theme, last-msg <on|off>"
	case "show", "detail":
		m.modeDetail = detailFull
		m.paletteMsg = "Detail view: full"
	case "open":
		if s, ok := m.selectedSession(); ok {
			_ = openSourceForSession(s.Provider, stripANSI(s.ID))
			m.paletteMsg = "Opened log"
		}
	case "copy-id":
		ids := m.selectedIDs()
		if len(ids) > 0 {
			_ = copyToClipboard(strings.Join(ids, "\n"))
		} else if s, ok := m.selectedSession(); ok {
			_ = copyToClipboard(stripANSI(s.ID))
		}
		m.paletteMsg = "Copied id"
	case "copy-detail":
		if s, ok := m.selectedSession(); ok {
			_ = copyToClipboard(stripANSI(s.Detail))
			m.paletteMsg = "Copied detail"
		}
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
		m.paletteMsg = "Group: " + safe(m.cfg.GroupBy, "none")
		m.applyFilterAndUpdateRows()
	case "view":
		if arg == "compact" {
			m.columnMode = "compact"
		} else if arg == "full" {
			m.columnMode = "full"
		} else {
			if m.columnMode == "full" {
				m.columnMode = "compact"
			} else {
				m.columnMode = "full"
			}
		}
		m.applyColumns()
		m.applyFilterAndUpdateRows()
	case "theme":
		m.themeIndex = (m.themeIndex + 1) % len(themes)
		applyTheme(themes[m.themeIndex], m.accessible)
		m.paletteMsg = "Theme: " + themes[m.themeIndex].name
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
	commands := []string{"show", "detail", "open", "copy-id", "copy-detail", "sort", "group", "view", "theme", "last-msg", "help", "?"}
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
	for _, c := range commands {
		if fuzzyMatch(input, c) {
			return c
		}
	}
	return input
}

func palettePreview(raw string) string {
	cmdLine := strings.TrimSpace(raw)
	if cmdLine == "" {
		return "Palette: show, open, copy-id, copy-detail, sort <key>, group <key>, view <full|compact|ultra|card>, theme, last-msg <on|off>"
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
	}
	return "Press Enter to run"
}

func columnsFor(width int, mode string, showLast bool) ([]table.Column, int) {
	if width <= 0 {
		width = 120
	}
	if mode == "card" {
		return []table.Column{{Title: "SESSION", Width: maxInt(20, width-4)}}, 0
	}

	if mode == "ultra" || width < 80 {
		cols := []table.Column{
			{Title: " ", Width: 6},
			{Title: "STATUS", Width: 8},
			{Title: "PROJECT", Width: 16},
			{Title: "AGE", Width: 5},
		}
		return cols, 2
	}

	compact := mode != "full" || width < 100

	iconCol := table.Column{Title: " ", Width: 4}
	statusCol := table.Column{Title: "STATUS", Width: 8}
	idCol := table.Column{Title: "ID", Width: 14}
	projectCol := table.Column{Title: "PROJECT", Width: 16}
	ageCol := table.Column{Title: "AGE", Width: 5}
	costCol := table.Column{Title: "COST", Width: 8}

	if compact {
		cols := []table.Column{iconCol, statusCol, idCol, projectCol, ageCol, costCol}
		return cols, 2
	}

	dirCol := table.Column{Title: "DIR", Width: 26}
	modelCol := table.Column{Title: "MODEL", Width: 12}
	sinceCol := table.Column{Title: "SINCE", Width: 14}

	cols := []table.Column{iconCol, statusCol, idCol, projectCol, dirCol, modelCol, ageCol, sinceCol, costCol}
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

func (m *tuiModel) applyFilterAndUpdateRows() {
	query := strings.TrimSpace(m.filter.Value())
	mode, q := parseQueryMode(query)
	filtered := make([]SessionView, 0, len(m.allSessions))

	for _, s := range m.allSessions {
		if !m.matchesProvider(s.Provider) {
			continue
		}
		if !m.matchesStatus(s.Status) {
			continue
		}
		if !m.matchesProject(s.Project) {
			continue
		}
		if !matchesQuery(s, mode, q) {
			continue
		}
		filtered = append(filtered, s)
	}

	sortSessions(filtered, m.cfg.SortBy)
	filtered = m.applyPinnedFirst(filtered)

	m.filteredSessions = filtered
	m.filterCounts = map[Status]int{}
	m.filterCost = 0
	m.filterTotal = len(filtered)
	for _, s := range filtered {
		m.filterCounts[s.Status]++
		m.filterCost += s.Cost
	}

	m.moveDir = map[string]int{}
	for idx, s := range filtered {
		id := stripANSI(s.ID)
		if prev, ok := m.lastOrder[id]; ok {
			if idx < prev {
				m.moveDir[id] = -1
			} else if idx > prev {
				m.moveDir[id] = 1
			}
		}
		m.lastOrder[id] = idx
	}

	rows := make([]table.Row, 0, len(filtered))
	meta := make([]rowMeta, 0, len(filtered))

	if m.cfg.GroupBy == "" {
		for i := range filtered {
			s := filtered[i]
			rows = append(rows, m.rowForSession(&s))
			meta = append(meta, rowMeta{kind: rowSession, session: &s})
		}
	} else {
		groups := groupSessions(filtered, m.cfg.GroupBy)
		for _, g := range groups {
			groupLabel := g.Group
			if strings.TrimSpace(groupLabel) == "" {
				groupLabel = "unknown"
			}
			rows = append(rows, m.groupRow(groupLabel))
			meta = append(meta, rowMeta{kind: rowGroup, group: groupLabel})
			for i := range g.Sessions {
				s := g.Sessions[i]
				rows = append(rows, m.rowForSession(&s))
				meta = append(meta, rowMeta{kind: rowSession, session: &s})
			}
		}
	}

	m.rowMeta = meta
	m.table.SetRows(rows)
	m.ensureCursorOnSession()
}

func parseQueryMode(raw string) (string, string) {
	q := strings.TrimSpace(raw)
	if strings.HasPrefix(strings.ToLower(q), "p:") {
		return "project", strings.TrimSpace(q[2:])
	}
	if strings.HasPrefix(strings.ToLower(q), "s:") {
		return "status", strings.TrimSpace(q[2:])
	}
	return "all", q
}

func matchesQuery(s SessionView, mode string, q string) bool {
	if strings.TrimSpace(q) == "" {
		return true
	}
	needle := strings.ToLower(q)
	var hay string
	switch mode {
	case "project":
		hay = strings.ToLower(s.Project)
	case "status":
		hay = strings.ToLower(string(s.Status))
	default:
		hay = strings.ToLower(fmt.Sprintf("%s %s %s %s %s", s.Provider, s.ID, s.Project, s.Dir, s.Model))
	}
	return fuzzyMatch(needle, hay)
}

func fuzzyMatch(needle, hay string) bool {
	if needle == "" {
		return true
	}
	n := []rune(needle)
	h := []rune(hay)
	idx := 0
	for _, r := range h {
		if r == n[idx] {
			idx++
			if idx == len(n) {
				return true
			}
		}
	}
	return false
}

func (m *tuiModel) rowForSession(s *SessionView) table.Row {
	id := s.ID
	idKey := stripANSI(id)
	if m.isRecentlyChanged(idKey) {
		id = styleChanged.Render(id)
	}

	prefix := m.prefixForSession(*s)
	status := statusBadge(s.Status)
	age := fmtAgo(s.Age)
	cost := formatCost(s.Cost)
	since := formatSince(s.LastSeen)
	mode := m.effectiveMode
	if mode == "" {
		mode = m.columnMode
	}

	if mode == "card" {
		return table.Row{m.renderCard(*s)}
	}

	if mode == "ultra" || m.width < 80 {
		return table.Row{prefix, status, s.Project, age}
	}

	if mode != "full" || m.width < 100 {
		return table.Row{prefix, status, id, s.Project, age, cost}
	}

	row := table.Row{prefix, status, id, s.Project, s.Dir, s.Model, age, since, cost}
	if m.showLastCol {
		row = append(row, lastSnippet(*s))
	}
	return row
}

func (m *tuiModel) groupRow(label string) table.Row {
	cols := m.table.Columns()
	row := make(table.Row, len(cols))
	if len(cols) > 0 {
		row[0] = styleGroupHeader.Render("==")
	}
	if m.idColumn >= 0 && m.idColumn < len(cols) {
		row[m.idColumn] = styleGroupHeader.Render(label)
	}
	return row
}

func (m *tuiModel) renderCard(s SessionView) string {
	status := chip(statusIcon(s.Status)+" "+strings.ToUpper(string(s.Status)), themes[m.themeIndex].accent, "0")
	provider := chip(strings.ToUpper(string(s.Provider)), themes[m.themeIndex].accent, "0")
	project := s.Project
	if project == "" {
		project = "unknown"
	}
	projectChip := chip(project, themes[m.themeIndex].muted, "0")

	title := fmt.Sprintf("%s %s %s", provider, status, projectChip)
	meta := fmt.Sprintf("%s  %s  %s", s.Model, fmtAgo(s.Age), formatSince(s.LastSeen))
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
	return styleCard.Render(strings.Join(lines, "\n"))
}

func chip(text, bg, fg string) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}
	return lipgloss.NewStyle().
		Background(lipgloss.Color(bg)).
		Foreground(lipgloss.Color(fg)).
		Padding(0, 1).
		Render(text)
}

func (m *tuiModel) prefixForSession(s SessionView) string {
	idKey := stripANSI(s.ID)
	pinned := m.pinned[idKey]
	selected := m.selected[idKey]

	pin := " "
	if pinned {
		pin = "★"
	}
	box := "□"
	if selected {
		box = "☑"
	}
	spark := sparkForAge(s.Age, m.cfg.ActiveWindow)
	move := " "
	if dir, ok := m.moveDir[idKey]; ok {
		if dir < 0 {
			move = "↑"
		} else if dir > 0 {
			move = "↓"
		}
	}
	timeline := miniTimeline(m.history[idKey], m.cfg.RefreshEvery)
	icon := providerIcon(s.Provider)
	return fmt.Sprintf("%s%s%s%s%s", pin, box, move, timeline, spark+icon)
}

func sparkForAge(age time.Duration, window time.Duration) string {
	levels := []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}
	if window <= 0 {
		return levels[len(levels)-1]
	}
	ratio := float64(age) / float64(window)
	idx := int((1.0 - ratio) * float64(len(levels)-1))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(levels) {
		idx = len(levels) - 1
	}
	return levels[idx]
}

func miniTimeline(times []time.Time, refresh time.Duration) string {
	if refresh <= 0 {
		refresh = time.Second
	}
	buckets := []string{"·", "·", "·", "·"}
	if len(times) == 0 {
		return strings.Join(buckets, "")
	}
	now := time.Now().UTC()
	window := refresh * 8
	for _, t := range times {
		age := now.Sub(t)
		if age < 0 {
			age = 0
		}
		if age > window {
			continue
		}
		ratio := float64(age) / float64(window)
		idx := int((1.0 - ratio) * float64(len(buckets)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(buckets) {
			idx = len(buckets) - 1
		}
		buckets[idx] = "•"
	}
	return strings.Join(buckets, "")
}

func (m *tuiModel) selectedSession() (SessionView, bool) {
	if len(m.rowMeta) == 0 {
		return SessionView{}, false
	}
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.rowMeta) {
		return SessionView{}, false
	}
	meta := m.rowMeta[idx]
	if meta.kind != rowSession || meta.session == nil {
		return SessionView{}, false
	}
	return *meta.session, true
}

func (m *tuiModel) selectedIDs() []string {
	if len(m.selected) == 0 {
		return nil
	}
	ids := make([]string, 0, len(m.selected))
	for id := range m.selected {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (m *tuiModel) ensureCursorOnSession() {
	if len(m.rowMeta) == 0 {
		return
	}
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.rowMeta) || m.rowMeta[idx].kind == rowSession {
		return
	}
	for i, meta := range m.rowMeta {
		if meta.kind == rowSession {
			m.table.SetCursor(i)
			return
		}
	}
}

func (m *tuiModel) View() string {
	var b strings.Builder

	title := styleTitle.Render("Active AI sessions")
	meta := styleMuted.Render(fmt.Sprintf("refresh %s • window %s • redact %v",
		m.cfg.RefreshEvery, m.cfg.ActiveWindow, m.cfg.Redact))
	header := styleHeaderBox.Render(fmt.Sprintf("%s  %s", title, meta))
	b.WriteString(header)
	b.WriteString("\n")

	filterLine := m.filter.View()
	if !m.filter.Focused() && m.filter.Value() == "" {
		filterLine = styleMuted.Render(m.filter.Prompt + m.filter.Placeholder)
	}
	mode, _ := parseQueryMode(m.filter.Value())
	modeLabel := strings.ToUpper(mode)
	modeChip := styleGroupHeader.Render("[" + modeLabel + "]")
	b.WriteString(styleBox.Render(modeChip + " " + filterLine))
	b.WriteString("\n")

	if m.paletteOpen {
		pLine := m.palette.View()
		if m.palette.Value() == "" {
			pLine = styleMuted.Render(m.palette.Prompt + m.palette.Placeholder)
		}
		b.WriteString(styleBox.Render(pLine))
		b.WriteString("\n")
		b.WriteString(styleMuted.Render(m.paletteMsg))
		b.WriteString("\n")
	} else if m.paletteMsg != "" {
		b.WriteString(styleMuted.Render(m.paletteMsg))
		b.WriteString("\n")
	}

	b.WriteString(m.legendView())
	b.WriteString("\n")
	b.WriteString(styleMuted.Render(m.actionsView()))
	b.WriteString("\n")

	content := m.table.View()
	if m.showSidebar && m.width >= 100 {
		content = lipgloss.JoinHorizontal(lipgloss.Top, m.sidebarView(), content)
	}

	if m.modeDetail == detailFull {
		b.WriteString(styleDetailBox.Render(m.detailView()))
		b.WriteString("\n")
		b.WriteString(styleMuted.Render(m.help.View(m.keys)))
		return b.String()
	}

	b.WriteString(content)
	b.WriteString("\n")
	b.WriteString(styleDetailBox.Render(m.detailView()))
	b.WriteString("\n")
	b.WriteString(styleMuted.Render(m.help.View(m.keys)))

	if m.err != nil && !m.filter.Focused() {
		b.WriteString("\n")
		b.WriteString(styleMuted.Render("⚠ " + m.err.Error()))
	}
	return b.String()
}

func (m *tuiModel) detailView() string {
	if s, ok := m.selectedSession(); ok {
		return formatDetailBlock(s)
	}
	if m.err != nil {
		return "Error: " + m.err.Error()
	}
	return "No session selected."
}

func formatDetailBlock(s SessionView) string {
	lines := strings.Split(strings.TrimSpace(s.Detail), "\n")
	maxKey := 0
	for _, line := range lines {
		if idx := strings.Index(line, ":"); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			if len(key) > maxKey {
				maxKey = len(key)
			}
		}
	}

	var b strings.Builder
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if idx := strings.Index(line, ":"); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			val := strings.TrimSpace(line[idx+1:])
			pad := strings.Repeat(" ", maxInt(0, maxKey-len(key)))
			b.WriteString(styleMuted.Render(key + pad + " : "))
			b.WriteString(val)
			b.WriteString("\n")
			continue
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString(styleMuted.Render("Seen at: "))
	b.WriteString(s.LastSeen.In(time.Local).Format("2006-01-02 15:04:05"))
	b.WriteString("\n")
	return b.String()
}

func (m *tuiModel) legendView() string {
	legend := fmt.Sprintf("%s %s %s %s %s", statusBadge(StatusRunning), statusBadge(StatusWaiting), statusBadge(StatusApproval), statusBadge(StatusStale), statusBadge(StatusEnded))
	group := safe(m.cfg.GroupBy, "none")
	mode := m.effectiveMode
	if mode == "" {
		mode = m.columnMode
	}
	filters := fmt.Sprintf("group:%s sort:%s view:%s theme:%s", group, m.cfg.SortBy, mode, themes[m.themeIndex].name)
	counts := fmt.Sprintf("total:%d run:%d wait:%d appr:%d stale:%d end:%d cost:%s",
		m.filterTotal,
		m.filterCounts[StatusRunning],
		m.filterCounts[StatusWaiting],
		m.filterCounts[StatusApproval],
		m.filterCounts[StatusStale],
		m.filterCounts[StatusEnded],
		formatCost(m.filterCost),
	)
	selected := ""
	if len(m.selected) > 0 {
		selected = fmt.Sprintf("selected:%d", len(m.selected))
	}
	parts := []string{legend, filters, counts}
	if selected != "" {
		parts = append(parts, selected)
	}
	return styleMuted.Render(strings.Join(parts, "  "))
}

func (m *tuiModel) actionsView() string {
	if len(m.selected) > 0 {
		return fmt.Sprintf("Selected %d: y copy ids • D copy detail • o open", len(m.selected))
	}
	if m.paletteOpen {
		return "Palette: enter to run • esc to cancel"
	}
	return "Actions: / filter • : palette • s sort • g group • v view • m last-msg"
}

func (m *tuiModel) sidebarView() string {
	var b strings.Builder
	b.WriteString(styleMuted.Render("Filters"))
	b.WriteString("\n")

	providerCounts := map[Provider]int{}
	statusCounts := map[Status]int{}
	projectCounts := map[string]int{}
	for _, s := range m.allSessions {
		providerCounts[s.Provider]++
		statusCounts[s.Status]++
		if s.Project != "" {
			projectCounts[strings.ToLower(s.Project)]++
		}
	}

	b.WriteString(m.sidebarLine("1", "claude", providerCounts[ProviderClaude], m.providerFilter[ProviderClaude]))
	b.WriteString("\n")
	b.WriteString(m.sidebarLine("2", "codex", providerCounts[ProviderCodex], m.providerFilter[ProviderCodex]))
	b.WriteString("\n\n")

	b.WriteString(styleMuted.Render("Status"))
	b.WriteString("\n")
	b.WriteString(m.sidebarLine("R", "running", statusCounts[StatusRunning], m.statusFilter[StatusRunning]))
	b.WriteString("\n")
	b.WriteString(m.sidebarLine("W", "waiting", statusCounts[StatusWaiting], m.statusFilter[StatusWaiting]))
	b.WriteString("\n")
	b.WriteString(m.sidebarLine("E", "approval", statusCounts[StatusApproval], m.statusFilter[StatusApproval]))
	b.WriteString("\n")
	b.WriteString(m.sidebarLine("S", "stale", statusCounts[StatusStale], m.statusFilter[StatusStale]))
	b.WriteString("\n")
	b.WriteString(m.sidebarLine("Z", "ended", statusCounts[StatusEnded], m.statusFilter[StatusEnded]))
	b.WriteString("\n")
	b.WriteString(m.sidebarLine("N", "attn", statusCounts[StatusNeedsAttn], m.statusFilter[StatusNeedsAttn]))
	b.WriteString("\n\n")

	b.WriteString(styleMuted.Render("Projects"))
	b.WriteString("\n")

	projects := topProjects(projectCounts, 6)
	for _, p := range projects {
		b.WriteString(m.sidebarLine("p", p.name, p.count, m.projectFilter[p.name]))
		b.WriteString("\n")
	}

	return styleBox.Render(b.String())
}

func (m *tuiModel) sidebarLine(key, label string, count int, active bool) string {
	check := " "
	if active {
		check = "*"
	}
	return fmt.Sprintf("[%s] %s %-10s %3d", key, check, label, count)
}

func (m *tuiModel) togglePin() {
	if s, ok := m.selectedSession(); ok {
		id := stripANSI(s.ID)
		m.pinned[id] = !m.pinned[id]
	}
}

func (m *tuiModel) toggleSelect() {
	if s, ok := m.selectedSession(); ok {
		id := stripANSI(s.ID)
		m.selected[id] = !m.selected[id]
	}
}

func (m *tuiModel) toggleProviderFilter(p Provider) {
	if m.providerFilter[p] {
		delete(m.providerFilter, p)
	} else {
		m.providerFilter[p] = true
	}
}

func (m *tuiModel) toggleStatusFilter(s Status) {
	if m.statusFilter[s] {
		delete(m.statusFilter, s)
	} else {
		m.statusFilter[s] = true
	}
}

func (m *tuiModel) toggleProjectFilter() {
	if s, ok := m.selectedSession(); ok {
		key := strings.ToLower(s.Project)
		if key == "" {
			return
		}
		if m.projectFilter[key] {
			delete(m.projectFilter, key)
		} else {
			m.projectFilter[key] = true
		}
	}
}

func (m *tuiModel) matchesProvider(p Provider) bool {
	if len(m.providerFilter) == 0 {
		return true
	}
	return m.providerFilter[p]
}

func (m *tuiModel) matchesStatus(s Status) bool {
	if len(m.statusFilter) == 0 {
		return true
	}
	return m.statusFilter[s]
}

func (m *tuiModel) matchesProject(project string) bool {
	if len(m.projectFilter) == 0 {
		return true
	}
	return m.projectFilter[strings.ToLower(project)]
}

func (m *tuiModel) cycleSort() {
	order := []string{"last_seen", "status", "provider", "cost", "project"}
	idx := indexOf(order, m.cfg.SortBy)
	if idx < 0 {
		idx = 0
	}
	m.cfg.SortBy = order[(idx+1)%len(order)]
}

func (m *tuiModel) cycleGroup() {
	order := []string{"", "provider", "project", "status", "day", "hour"}
	idx := indexOf(order, m.cfg.GroupBy)
	if idx < 0 {
		idx = 0
	}
	m.cfg.GroupBy = order[(idx+1)%len(order)]
}

func indexOf(list []string, val string) int {
	for i, v := range list {
		if v == val {
			return i
		}
	}
	return -1
}

func (m *tuiModel) cycleViewMode() {
	order := []string{"full", "compact", "ultra", "card"}
	idx := indexOf(order, m.columnMode)
	if idx < 0 {
		idx = 0
	}
	m.columnMode = order[(idx+1)%len(order)]
}

func (m *tuiModel) jumpToStatus(status Status) {
	for i, meta := range m.rowMeta {
		if meta.kind == rowSession && meta.session != nil && meta.session.Status == status {
			m.table.SetCursor(i)
			return
		}
	}
}

func (m *tuiModel) applyPinnedFirst(list []SessionView) []SessionView {
	if len(m.pinned) == 0 {
		return list
	}
	pinned := make([]SessionView, 0, len(list))
	rest := make([]SessionView, 0, len(list))
	for _, s := range list {
		id := stripANSI(s.ID)
		if m.pinned[id] {
			pinned = append(pinned, s)
		} else {
			rest = append(rest, s)
		}
	}
	return append(pinned, rest...)
}

func (m *tuiModel) markChanges(sessions []SessionView) {
	now := time.Now().UTC()
	for _, s := range sessions {
		id := stripANSI(s.ID)
		prev, ok := m.lastSnapshot[id]
		if !ok || prev.Status != s.Status || !prev.LastSeen.Equal(s.LastSeen) || prev.Cost != s.Cost {
			m.changedAt[id] = now
			m.history[id] = append(m.history[id], now)
			if len(m.history[id]) > 8 {
				m.history[id] = m.history[id][len(m.history[id])-8:]
			}
		}
		m.lastSnapshot[id] = s
	}
}

func (m *tuiModel) isRecentlyChanged(id string) bool {
	when, ok := m.changedAt[id]
	if !ok {
		return false
	}
	if time.Since(when) > 2*m.cfg.RefreshEvery {
		delete(m.changedAt, id)
		return false
	}
	return true
}

func lastSnippet(s SessionView) string {
	if s.LastUser != "" {
		return truncateString("u: "+s.LastUser, 22)
	}
	if s.LastAssist != "" {
		return truncateString("a: "+s.LastAssist, 22)
	}
	return ""
}

func truncateString(s string, max int) string {
	if len([]rune(s)) <= max {
		return s
	}
	parts := []rune(s)
	return string(parts[:max-1]) + "…"
}

func formatSince(ts time.Time) string {
	if ts.IsZero() {
		return ""
	}
	return ts.In(time.Local).Format("01-02 15:04")
}

func topProjects(counts map[string]int, limit int) []struct {
	name  string
	count int
} {
	type item struct {
		name  string
		count int
	}
	var items []item
	for name, count := range counts {
		items = append(items, item{name: name, count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].count == items[j].count {
			return items[i].name < items[j].name
		}
		return items[i].count > items[j].count
	})
	if len(items) > limit {
		items = items[:limit]
	}
	out := make([]struct {
		name  string
		count int
	}, len(items))
	for i, it := range items {
		out[i] = struct {
			name  string
			count int
		}{name: it.name, count: it.count}
	}
	return out
}

func providerIcon(p Provider) string {
	switch p {
	case ProviderClaude:
		return "🧠"
	case ProviderCodex:
		return "⚡"
	default:
		return "?"
	}
}

func statusBadge(s Status) string {
	icon := statusIcon(s)
	switch s {
	case StatusRunning:
		return styleBadgeRun.Render(icon + " RUN")
	case StatusApproval:
		return styleBadgeAppr.Render(icon + " APPR")
	case StatusStale:
		return styleBadgeStale.Render(icon + " STALE")
	case StatusEnded:
		return styleBadgeStale.Render(icon + " DONE")
	case StatusNeedsAttn:
		return styleBadgeWait.Render(icon + " ATTN")
	default:
		return styleBadgeWait.Render(icon + " WAIT")
	}
}

func statusIcon(s Status) string {
	switch s {
	case StatusRunning:
		return "▶"
	case StatusApproval:
		return "⚠"
	case StatusStale:
		return "…"
	case StatusEnded:
		return "✓"
	case StatusNeedsAttn:
		return "‼"
	default:
		return "⏸"
	}
}

func formatCost(c float64) string {
	if c <= 0 {
		return ""
	}
	return fmt.Sprintf("$%.3f", c)
}

func stripANSI(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return re.ReplaceAllString(s, "")
}

func copyToClipboard(text string) error {
	cmd := exec.Command("pbcopy")
	in, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		_ = in.Close()
		return err
	}
	_, _ = io.WriteString(in, text)
	_ = in.Close()
	return cmd.Wait()
}

func openSourceForSession(provider Provider, id string) error {
	p, _, err := resolveSourcePath(string(provider), id)
	if err != nil {
		return err
	}
	return exec.Command("open", p).Run()
}
