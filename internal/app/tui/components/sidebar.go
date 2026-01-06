package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vburojevic/aistat/internal/app/tui/state"
	"github.com/vburojevic/aistat/internal/app/tui/theme"
)

// SidebarData holds the data needed to render the sidebar
type SidebarData struct {
	ProviderCounts map[state.Provider]int
	StatusCounts   map[state.Status]int
	ProjectCounts  map[string]int
	ActiveProvider map[state.Provider]bool
	ActiveStatus   map[state.Status]bool
	ActiveProject  map[string]bool
}

// BuildSidebarData builds sidebar data from sessions
func BuildSidebarData(sessions []state.SessionView, filters *state.FilterState) SidebarData {
	data := SidebarData{
		ProviderCounts: make(map[state.Provider]int),
		StatusCounts:   make(map[state.Status]int),
		ProjectCounts:  make(map[string]int),
		ActiveProvider: filters.ProviderFilter,
		ActiveStatus:   filters.StatusFilter,
		ActiveProject:  filters.ProjectFilter,
	}

	for _, s := range sessions {
		data.ProviderCounts[s.Provider]++
		data.StatusCounts[s.Status]++
		if s.Project != "" {
			data.ProjectCounts[strings.ToLower(s.Project)]++
		}
	}

	return data
}

// RenderSidebar renders the filter sidebar
func RenderSidebar(data SidebarData, styles theme.Styles, width int) string {
	if width <= 0 {
		return ""
	}

	var b strings.Builder

	// Providers section
	b.WriteString(styles.Section.Render("PROVIDERS"))
	b.WriteString("\n")
	b.WriteString(sidebarLine("1", "claude", data.ProviderCounts[state.ProviderClaude], data.ActiveProvider[state.ProviderClaude], styles))
	b.WriteString("\n")
	b.WriteString(sidebarLine("2", "codex", data.ProviderCounts[state.ProviderCodex], data.ActiveProvider[state.ProviderCodex], styles))
	b.WriteString("\n\n")

	// Status section
	b.WriteString(styles.Section.Render("STATUS"))
	b.WriteString("\n")
	b.WriteString(sidebarLine("R", "running", data.StatusCounts[state.StatusRunning], data.ActiveStatus[state.StatusRunning], styles))
	b.WriteString("\n")
	b.WriteString(sidebarLine("W", "waiting", data.StatusCounts[state.StatusWaiting], data.ActiveStatus[state.StatusWaiting], styles))
	b.WriteString("\n")
	b.WriteString(sidebarLine("E", "approval", data.StatusCounts[state.StatusApproval], data.ActiveStatus[state.StatusApproval], styles))
	b.WriteString("\n")
	b.WriteString(sidebarLine("N", "attn", data.StatusCounts[state.StatusNeedsAttn], data.ActiveStatus[state.StatusNeedsAttn], styles))
	b.WriteString("\n")
	b.WriteString(sidebarLine("S", "stale", data.StatusCounts[state.StatusStale], data.ActiveStatus[state.StatusStale], styles))
	b.WriteString("\n")
	b.WriteString(sidebarLine("Z", "ended", data.StatusCounts[state.StatusEnded], data.ActiveStatus[state.StatusEnded], styles))
	b.WriteString("\n\n")

	// Projects section (top 6)
	b.WriteString(styles.Section.Render("PROJECTS"))
	b.WriteString("\n")

	projects := topProjects(data.ProjectCounts, 6)
	for _, p := range projects {
		b.WriteString(sidebarLine("p", p.name, p.count, data.ActiveProject[p.name], styles))
		b.WriteString("\n")
	}

	content := styles.Box.Render(b.String())
	return lipgloss.NewStyle().Width(width).Render(content)
}

func sidebarLine(key, label string, count int, active bool, styles theme.Styles) string {
	check := " "
	if active {
		check = "●"
	}
	line := fmt.Sprintf("[%s] %s %-10s %3d", key, check, label, count)
	if active {
		return styles.OverlaySel.Render(line)
	}
	return line
}

type projectCount struct {
	name  string
	count int
}

func topProjects(counts map[string]int, limit int) []projectCount {
	items := make([]projectCount, 0, len(counts))
	for name, count := range counts {
		items = append(items, projectCount{name: name, count: count})
	}

	// Sort by count descending, then name ascending
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			swap := false
			if items[i].count < items[j].count {
				swap = true
			} else if items[i].count == items[j].count && items[i].name > items[j].name {
				swap = true
			}
			if swap {
				items[i], items[j] = items[j], items[i]
			}
		}
	}

	if len(items) > limit {
		items = items[:limit]
	}
	return items
}
