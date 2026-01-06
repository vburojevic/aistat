package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/vburojevic/aistat/internal/app/tui/state"
	"github.com/vburojevic/aistat/internal/app/tui/theme"
	"github.com/vburojevic/aistat/internal/app/tui/widgets"
)

// DashboardView renders the dashboard-first landing view
type DashboardView struct {
	styles      theme.Styles
	filters     *state.FilterState
	width       int
	height      int
	cursor      int
	filterInput textinput.Model
}

// NewDashboardView creates a new dashboard view
func NewDashboardView(styles theme.Styles, filters *state.FilterState) *DashboardView {
	input := textinput.New()
	input.Placeholder = "filter projects"
	input.Prompt = "◆ "
	input.CharLimit = 64
	input.Width = 32

	return &DashboardView{
		styles:      styles,
		filters:     filters,
		filterInput: input,
	}
}

// SetSize sets the view dimensions
func (v *DashboardView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.filterInput.Width = widgets.MinInt(40, widgets.MaxInt(20, width-20))
}

// Focus focuses the view
func (v *DashboardView) Focus() {
	v.filterInput.Focus()
}

// Blur blurs the view
func (v *DashboardView) Blur() {
	v.filterInput.Blur()
}

// SetStyles updates the styles
func (v *DashboardView) SetStyles(styles theme.Styles) {
	v.styles = styles
}

// Cursor returns the current cursor position
func (v *DashboardView) Cursor() int {
	return v.cursor
}

// SetCursor sets the cursor position
func (v *DashboardView) SetCursor(pos int) {
	v.cursor = pos
}

// MoveCursor moves the cursor by delta
func (v *DashboardView) MoveCursor(delta int, max int) {
	v.cursor += delta
	if v.cursor < 0 {
		v.cursor = 0
	}
	if max > 0 && v.cursor >= max {
		v.cursor = max - 1
	}
}

// FilterValue returns the current filter value
func (v *DashboardView) FilterValue() string {
	return v.filterInput.Value()
}

// SetFilterValue sets the filter value
func (v *DashboardView) SetFilterValue(val string) {
	v.filterInput.SetValue(val)
}

// FilterInput returns the filter input model
func (v *DashboardView) FilterInput() *textinput.Model {
	return &v.filterInput
}

// Render renders the dashboard view
func (v *DashboardView) Render(sessions []state.SessionView, counts map[state.Status]int, totalCost float64) string {
	var lines []string

	// Title
	lines = append(lines, v.styles.Section.Render("◆ Dashboard"))
	lines = append(lines, v.styles.Muted.Render("tab back • enter focus • space toggle • a clear"))
	lines = append(lines, "")

	// Attention banner (if any sessions need attention)
	attnBanner := v.renderAttentionBanner(counts)
	if attnBanner != "" {
		lines = append(lines, attnBanner)
		lines = append(lines, "")
	}

	// Quick stats
	statsLine := v.renderQuickStats(sessions, counts, totalCost)
	lines = append(lines, statsLine)
	lines = append(lines, "")

	// Filter input
	input := v.filterInput.View()
	if v.filterInput.Value() == "" && !v.filterInput.Focused() {
		input = v.styles.Muted.Render(v.filterInput.Prompt + v.filterInput.Placeholder)
	}
	lines = append(lines, input)
	lines = append(lines, "")

	// Projects table header
	header := fmt.Sprintf("%-20s %4s  %3s %3s %3s %3s %3s %3s  %s",
		"PROJECT", "CNT", "●", "◐", "◉", "◈", "◌", "◇", "LAST")
	lines = append(lines, v.styles.Muted.Render(header))

	// Filter projects
	items := state.FilterDashboardItems(sessions, v.filters)
	items = state.FilterProjectItems(items, v.filterInput.Value())

	if len(items) == 0 {
		lines = append(lines, v.styles.Muted.Render("No active projects found."))
		return v.styles.OverlayBox.Render(strings.Join(lines, "\n"))
	}

	// Normalize cursor
	if v.cursor >= len(items) {
		v.cursor = len(items) - 1
	}
	if v.cursor < 0 {
		v.cursor = 0
	}

	// Render project rows
	maxRows := widgets.MaxInt(6, v.height-14)
	if maxRows > len(items) {
		maxRows = len(items)
	}

	start := 0
	if v.cursor >= maxRows {
		start = v.cursor - maxRows + 1
	}
	end := widgets.MinInt(len(items), start+maxRows)

	for i := start; i < end; i++ {
		it := items[i]
		active := v.filters.ProjectFilter[strings.ToLower(it.Name)]
		check := " "
		if active {
			check = "●"
		}

		last := ""
		if !it.LastSeen.IsZero() {
			last = it.LastSeen.In(time.Local).Format("01-02 15:04")
		}

		line := fmt.Sprintf("%s %-20s %4d  %3d %3d %3d %3d %3d %3d  %s",
			check,
			widgets.TruncateString(it.Name, 20),
			it.Count,
			it.StatusCount[state.StatusRunning],
			it.StatusCount[state.StatusWaiting],
			it.StatusCount[state.StatusApproval],
			it.StatusCount[state.StatusNeedsAttn],
			it.StatusCount[state.StatusStale],
			it.StatusCount[state.StatusEnded],
			last,
		)

		if i == v.cursor {
			line = v.styles.OverlaySel.Render(line)
		}
		lines = append(lines, line)
	}

	return v.styles.OverlayBox.Render(strings.Join(lines, "\n"))
}

// renderAttentionBanner renders the attention-needed banner
func (v *DashboardView) renderAttentionBanner(counts map[state.Status]int) string {
	appr := counts[state.StatusApproval]
	attn := counts[state.StatusNeedsAttn]

	if appr == 0 && attn == 0 {
		return ""
	}

	var parts []string
	if appr > 0 {
		badge := v.styles.BadgeAppr.Render(fmt.Sprintf("◉ %d APPROVAL", appr))
		parts = append(parts, badge)
	}
	if attn > 0 {
		badge := v.styles.BadgeAttn.Render(fmt.Sprintf("◈ %d ATTENTION", attn))
		parts = append(parts, badge)
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, parts...)
}

// renderQuickStats renders the quick stats line
func (v *DashboardView) renderQuickStats(sessions []state.SessionView, counts map[state.Status]int, totalCost float64) string {
	total := len(sessions)

	// Count by provider
	claudeCount := 0
	codexCount := 0
	for _, s := range sessions {
		if s.Provider == state.ProviderClaude {
			claudeCount++
		} else if s.Provider == state.ProviderCodex {
			codexCount++
		}
	}

	statsLeft := fmt.Sprintf("Total: %d   Run: %d   Wait: %d   Stale: %d   Cost: %s",
		total,
		counts[state.StatusRunning],
		counts[state.StatusWaiting],
		counts[state.StatusStale],
		widgets.FormatCost(totalCost),
	)

	statsRight := fmt.Sprintf("Claude: %d   Codex: %d", claudeCount, codexCount)

	return v.styles.Muted.Render(statsLeft + "  |  " + statsRight)
}

// SelectedProject returns the currently selected project item
func (v *DashboardView) SelectedProject(sessions []state.SessionView) *state.ProjectItem {
	items := state.FilterDashboardItems(sessions, v.filters)
	items = state.FilterProjectItems(items, v.filterInput.Value())

	if len(items) == 0 {
		return nil
	}

	if v.cursor >= len(items) {
		v.cursor = len(items) - 1
	}
	if v.cursor < 0 {
		return nil
	}

	return &items[v.cursor]
}
