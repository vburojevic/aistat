package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
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
	lines = append(lines, v.styles.Muted.Render("tab list • enter focus • space toggle • a clear"))
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

	// Filter projects
	items := state.FilterDashboardItems(sessions, v.filters)
	items = state.FilterProjectItems(items, v.filterInput.Value())

	if len(items) == 0 {
		lines = append(lines, v.styles.Muted.Render("No active projects found."))
		return strings.Join(lines, "\n")
	}

	// Normalize cursor
	if v.cursor >= len(items) {
		v.cursor = len(items) - 1
	}
	if v.cursor < 0 {
		v.cursor = 0
	}

	// Calculate visible cards (each card takes ~3 lines with margin)
	maxCards := widgets.MaxInt(3, (v.height-12)/3)
	if maxCards > len(items) {
		maxCards = len(items)
	}

	start := 0
	if v.cursor >= maxCards {
		start = v.cursor - maxCards + 1
	}
	end := widgets.MinInt(len(items), start+maxCards)

	// Render project cards
	for i := start; i < end; i++ {
		it := items[i]
		selected := i == v.cursor
		active := v.filters.ProjectFilter[strings.ToLower(it.Name)]
		card := v.renderProjectCard(it, selected, active)
		lines = append(lines, card)
	}

	// Scroll indicator
	if len(items) > maxCards {
		remaining := len(items) - end
		if remaining > 0 {
			lines = append(lines, v.styles.Muted.Render(fmt.Sprintf("  ↓ %d more projects...", remaining)))
		}
	}

	return strings.Join(lines, "\n")
}

// renderProjectCard renders a single project card with health-based borders
func (v *DashboardView) renderProjectCard(it state.ProjectItem, selected, active bool) string {
	cardWidth := widgets.MinInt(60, widgets.MaxInt(40, v.width-10))

	// Determine card health
	health := projectHealth(it)

	// Build header: checkbox + name + status indicator + time
	check := " "
	if active {
		check = "●"
	}

	// Time ago
	timeAgo := ""
	if !it.LastSeen.IsZero() {
		timeAgo = formatTimeAgo(it.LastSeen)
	}

	// Status indicator in header (only for healthy/dormant cards)
	statusIndicator := ""
	if health == "green" {
		statusIndicator = v.styles.Accent.Render(" ✓ all clear")
	}

	// Calculate padding for right-aligned time
	nameLen := len(it.Name)
	if nameLen > 20 {
		nameLen = 20
	}
	name := widgets.TruncateString(it.Name, 20)
	padding := cardWidth - nameLen - len(timeAgo) - len(statusIndicator) - 6

	var header string
	if padding > 0 {
		header = fmt.Sprintf("%s %s%s%s%s",
			check,
			name,
			statusIndicator,
			strings.Repeat(" ", padding),
			v.styles.Muted.Render(timeAgo),
		)
	} else {
		header = fmt.Sprintf("%s %s%s  %s", check, name, statusIndicator, v.styles.Muted.Render(timeAgo))
	}

	// Build badge line (only urgent states)
	var badges []string
	if n := it.StatusCount[state.StatusApproval]; n > 0 {
		badges = append(badges, widgets.StatusChip(state.StatusApproval, n, v.styles))
	}
	if n := it.StatusCount[state.StatusNeedsAttn]; n > 0 {
		badges = append(badges, widgets.StatusChip(state.StatusNeedsAttn, n, v.styles))
	}

	// Build summary line for non-urgent counts
	activeCount := it.StatusCount[state.StatusRunning] + it.StatusCount[state.StatusWaiting]
	doneCount := it.StatusCount[state.StatusEnded] + it.StatusCount[state.StatusStale]

	var statusLine string
	if len(badges) > 0 {
		// Has urgent items: show badges + summary
		badgeLine := strings.Join(badges, "  ")
		var summaryParts []string
		if activeCount > 0 {
			summaryParts = append(summaryParts, fmt.Sprintf("%d active", activeCount))
		}
		if doneCount > 0 {
			summaryParts = append(summaryParts, fmt.Sprintf("%d done", doneCount))
		}
		if len(summaryParts) > 0 {
			statusLine = badgeLine + "\n  " + v.styles.Muted.Render("+ "+strings.Join(summaryParts, ", "))
		} else {
			statusLine = badgeLine
		}
	} else if activeCount > 0 || doneCount > 0 {
		// No urgent: show just counts
		var parts []string
		if activeCount > 0 {
			parts = append(parts, fmt.Sprintf("%d active", activeCount))
		}
		if doneCount > 0 {
			parts = append(parts, fmt.Sprintf("%d done", doneCount))
		}
		statusLine = v.styles.Muted.Render(strings.Join(parts, ", "))
	} else {
		statusLine = v.styles.Muted.Render("(no sessions)")
	}

	// Combine into card
	content := header + "\n  " + statusLine

	// Pick card style based on health
	var cardStyle = v.styles.DashCard.Width(cardWidth)
	switch health {
	case "red":
		cardStyle = v.styles.DashCardRed.Width(cardWidth)
	case "green":
		cardStyle = v.styles.DashCardGreen.Width(cardWidth)
	default:
		cardStyle = v.styles.DashCardGray.Width(cardWidth)
	}

	// Override border if selected
	if selected {
		cardStyle = cardStyle.BorderForeground(v.styles.Accent.GetForeground())
	}

	return cardStyle.Render(content)
}

// projectHealth determines the card health based on status counts
func projectHealth(it state.ProjectItem) string {
	// Red = needs input (approval or attention)
	if it.StatusCount[state.StatusApproval] > 0 || it.StatusCount[state.StatusNeedsAttn] > 0 {
		return "red"
	}
	// Green = healthy (has running or waiting)
	if it.StatusCount[state.StatusRunning] > 0 || it.StatusCount[state.StatusWaiting] > 0 {
		return "green"
	}
	// Gray = dormant (only ended/stale or empty)
	return "gray"
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

	return strings.Join(parts, "  ")
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

// formatTimeAgo formats a time as a human-readable "Xm ago" string
func formatTimeAgo(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return "now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(d.Hours()/24))
}
