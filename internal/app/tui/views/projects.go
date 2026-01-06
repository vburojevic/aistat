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

// ProjectsView renders the projects picker overlay
type ProjectsView struct {
	styles      theme.Styles
	filters     *state.FilterState
	width       int
	height      int
	cursor      int
	filterInput textinput.Model
}

// NewProjectsView creates a new projects picker view
func NewProjectsView(styles theme.Styles, filters *state.FilterState) *ProjectsView {
	input := textinput.New()
	input.Placeholder = "filter projects"
	input.Prompt = "◆ "
	input.CharLimit = 64
	input.Width = 32

	return &ProjectsView{
		styles:      styles,
		filters:     filters,
		filterInput: input,
	}
}

// SetSize sets the view dimensions
func (v *ProjectsView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.filterInput.Width = widgets.MinInt(40, widgets.MaxInt(20, width-20))
}

// Focus focuses the view
func (v *ProjectsView) Focus() {
	v.filterInput.Focus()
}

// Blur blurs the view
func (v *ProjectsView) Blur() {
	v.filterInput.Blur()
}

// SetStyles updates the styles
func (v *ProjectsView) SetStyles(styles theme.Styles) {
	v.styles = styles
}

// Cursor returns the current cursor position
func (v *ProjectsView) Cursor() int {
	return v.cursor
}

// SetCursor sets the cursor position
func (v *ProjectsView) SetCursor(pos int) {
	v.cursor = pos
}

// MoveCursor moves the cursor by delta
func (v *ProjectsView) MoveCursor(delta int, max int) {
	v.cursor += delta
	if v.cursor < 0 {
		v.cursor = 0
	}
	if max > 0 && v.cursor >= max {
		v.cursor = max - 1
	}
}

// FilterValue returns the current filter value
func (v *ProjectsView) FilterValue() string {
	return v.filterInput.Value()
}

// SetFilterValue sets the filter value
func (v *ProjectsView) SetFilterValue(val string) {
	v.filterInput.SetValue(val)
}

// FilterInput returns the filter input model
func (v *ProjectsView) FilterInput() *textinput.Model {
	return &v.filterInput
}

// Render renders the projects picker view
func (v *ProjectsView) Render(items []state.ProjectItem) string {
	var lines []string

	// Title
	lines = append(lines, v.styles.Section.Render("◆ Projects"))
	lines = append(lines, v.styles.Muted.Render("enter/space toggle • a clear • esc close"))
	lines = append(lines, "")

	// Filter input
	input := v.filterInput.View()
	if v.filterInput.Value() == "" && !v.filterInput.Focused() {
		input = v.styles.Muted.Render(v.filterInput.Prompt + v.filterInput.Placeholder)
	}
	lines = append(lines, input)
	lines = append(lines, "")

	// Filter items
	filtered := state.FilterProjectItems(items, v.filterInput.Value())

	if len(filtered) == 0 {
		lines = append(lines, "No projects found.")
		return v.styles.OverlayBox.Render(strings.Join(lines, "\n"))
	}

	// Normalize cursor
	if v.cursor >= len(filtered) {
		v.cursor = len(filtered) - 1
	}
	if v.cursor < 0 {
		v.cursor = 0
	}

	// Render project rows
	maxRows := widgets.MaxInt(6, v.height-12)
	if maxRows > len(filtered) {
		maxRows = len(filtered)
	}

	start := 0
	if v.cursor >= maxRows {
		start = v.cursor - maxRows + 1
	}
	end := widgets.MinInt(len(filtered), start+maxRows)

	for i := start; i < end; i++ {
		it := filtered[i]
		active := v.filters.ProjectFilter[strings.ToLower(it.Name)]
		check := " "
		if active {
			check = "●"
		}

		last := ""
		if !it.LastSeen.IsZero() {
			last = it.LastSeen.In(time.Local).Format("01-02 15:04")
		}

		statusBits := fmt.Sprintf("●%d ◐%d ◉%d",
			it.StatusCount[state.StatusRunning],
			it.StatusCount[state.StatusWaiting],
			it.StatusCount[state.StatusApproval])

		line := fmt.Sprintf("%s %-18s %4d  %s  %s", check, widgets.TruncateString(it.Name, 18), it.Count, last, statusBits)

		if i == v.cursor {
			line = v.styles.OverlaySel.Render(line)
		}
		lines = append(lines, line)
	}

	return v.styles.OverlayBox.Render(strings.Join(lines, "\n"))
}

// SelectedProject returns the currently selected project item
func (v *ProjectsView) SelectedProject(items []state.ProjectItem) *state.ProjectItem {
	filtered := state.FilterProjectItems(items, v.filterInput.Value())

	if len(filtered) == 0 {
		return nil
	}

	if v.cursor >= len(filtered) {
		v.cursor = len(filtered) - 1
	}
	if v.cursor < 0 {
		return nil
	}

	return &filtered[v.cursor]
}
