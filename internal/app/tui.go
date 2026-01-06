package app

import (
	"fmt"
	"io"
	"os/exec"
	"regexp"
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

type tuiKeyMap struct {
	UpDown    key.Binding
	Quit      key.Binding
	Refresh   key.Binding
	Filter    key.Binding
	ToggleRed key.Binding
	CopyID    key.Binding
	OpenFile  key.Binding
}

func (k tuiKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.Refresh, k.Filter, k.ToggleRed, k.CopyID, k.OpenFile}
}
func (k tuiKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Quit, k.Refresh, k.Filter},
		{k.ToggleRed, k.CopyID, k.OpenFile},
	}
}

type tuiModel struct {
	cfg Config

	table  table.Model
	filter textinput.Model
	help   help.Model
	keys   tuiKeyMap

	width  int
	height int

	allSessions []SessionView
	err         error
	lastRefresh time.Time
}

var (
	styleTitle     = lipgloss.NewStyle().Bold(true)
	styleMuted     = lipgloss.NewStyle().Faint(true)
	styleBox       = lipgloss.NewStyle().Padding(0, 1)
	styleDetailBox = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	styleHeaderBox = lipgloss.NewStyle().Padding(0, 1)
	styleBadge     = lipgloss.NewStyle().Bold(true).Padding(0, 1)

	styleBadgeRun   = styleBadge.Copy().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("2"))
	styleBadgeWait  = styleBadge.Copy().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("3"))
	styleBadgeAppr  = styleBadge.Copy().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("1"))
	styleBadgeStale = styleBadge.Copy().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("8"))
)

func runTUI(cfg Config) error {
	cols := []table.Column{
		{Title: " ", Width: 2},
		{Title: "STATUS", Width: 12},
		{Title: "ID", Width: 14},
		{Title: "PROJECT", Width: 18},
		{Title: "DIR", Width: 28},
		{Title: "MODEL", Width: 14},
		{Title: "AGE", Width: 5},
		{Title: "COST", Width: 8},
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
	)

	t.SetStyles(table.DefaultStyles())

	f := textinput.New()
	f.Placeholder = "filter (/, esc)"
	f.Prompt = "🔎 "
	f.CharLimit = 128
	f.Width = 40

	km := tuiKeyMap{
		UpDown:    key.NewBinding(key.WithKeys("up", "down", "j", "k"), key.WithHelp("↑/↓", "move")),
		Quit:      key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Refresh:   key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Filter:    key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		ToggleRed: key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "toggle redact")),
		CopyID:    key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy id")),
		OpenFile:  key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open transcript/log")),
	}

	m := tuiModel{
		cfg:    cfg,
		table:  t,
		filter: f,
		help:   help.New(),
		keys:   km,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m tuiModel) Init() tea.Cmd {
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

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.filter.Width = minInt(60, maxInt(20, m.width-20))
		tableHeight := maxInt(5, m.height-2-1-9-1)
		m.table.SetHeight(tableHeight)

	case sessionsMsg:
		m.err = msg.Err
		if msg.Err == nil {
			m.allSessions = msg.Sessions
			m.lastRefresh = time.Now().UTC()
			m.applyFilterAndUpdateRows()
		}
	case tickMsg:
		cmds = append(cmds, fetchSessionsCmd(m.cfg), tickCmd(m.cfg.RefreshEvery))

	case tea.KeyMsg:
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
		case "c":
			m.cfg.Redact = !m.cfg.Redact
			m.applyFilterAndUpdateRows()
		case "y":
			if s, ok := m.selectedSession(); ok {
				_ = copyToClipboard(stripANSI(s.ID))
			}
		case "o":
			if s, ok := m.selectedSession(); ok {
				_ = openSourceForSession(s.Provider, stripANSI(s.ID))
			}
		}
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *tuiModel) applyFilterAndUpdateRows() {
	query := strings.TrimSpace(strings.ToLower(m.filter.Value()))
	var filtered []SessionView
	if query == "" {
		filtered = m.allSessions
	} else {
		for _, s := range m.allSessions {
			hay := strings.ToLower(fmt.Sprintf("%s %s %s %s %s", s.Provider, s.ID, s.Project, s.Dir, s.Model))
			if strings.Contains(hay, query) {
				filtered = append(filtered, s)
			}
		}
	}

	rows := make([]table.Row, 0, len(filtered))
	for _, s := range filtered {
		rows = append(rows, table.Row{
			providerIcon(s.Provider),
			statusBadge(s.Status),
			s.ID,
			s.Project,
			s.Dir,
			s.Model,
			fmtAgo(s.Age),
			formatCost(s.Cost),
		})
	}
	m.table.SetRows(rows)
}

func (m tuiModel) selectedSession() (SessionView, bool) {
	row := m.table.SelectedRow()
	if len(row) < 3 {
		return SessionView{}, false
	}
	id := stripANSI(row[2])
	for _, s := range m.allSessions {
		if stripANSI(s.ID) == id {
			return s, true
		}
	}
	return SessionView{}, false
}

func (m tuiModel) View() string {
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
	b.WriteString(styleBox.Render(filterLine))
	b.WriteString("\n")

	b.WriteString(m.table.View())
	b.WriteString("\n")

	if s, ok := m.selectedSession(); ok {
		b.WriteString(styleDetailBox.Render(s.Detail))
	} else {
		if m.err != nil {
			b.WriteString(styleDetailBox.Render("Error: " + m.err.Error()))
		} else {
			b.WriteString(styleDetailBox.Render("No session selected."))
		}
	}
	b.WriteString("\n")

	b.WriteString(styleMuted.Render(m.help.View(m.keys)))

	if m.err != nil && !m.filter.Focused() {
		b.WriteString("\n")
		b.WriteString(styleMuted.Render("⚠ " + m.err.Error()))
	}
	return b.String()
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
	switch s {
	case StatusRunning:
		return styleBadgeRun.Render(" RUN ")
	case StatusApproval:
		return styleBadgeAppr.Render(" APPR ")
	case StatusStale:
		return styleBadgeStale.Render(" STALE ")
	case StatusEnded:
		return styleBadgeStale.Render(" DONE ")
	case StatusNeedsAttn:
		return styleBadgeWait.Render(" ATTN ")
	default:
		return styleBadgeWait.Render(" WAIT ")
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
