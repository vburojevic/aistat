package components

import (
	"fmt"
	"sort"
	"strings"

	"github.com/vburojevic/aistat/internal/app/tui/state"
	"github.com/vburojevic/aistat/internal/app/tui/theme"
	"github.com/vburojevic/aistat/internal/app/tui/widgets"
)

// RenderActiveFilters renders the active filters as pills
func RenderActiveFilters(filters *state.FilterState, selectedCount int, styles theme.Styles) string {
	var pills []string

	// Selection count
	if selectedCount > 0 {
		pills = append(pills, styles.PillActive.Render(fmt.Sprintf("SELECT %d", selectedCount)))
	}

	// Provider filters
	if len(filters.ProviderFilter) > 0 {
		var providers []string
		for p := range filters.ProviderFilter {
			providers = append(providers, string(p))
		}
		sort.Strings(providers)
		for _, p := range providers {
			pills = append(pills, styles.PillActive.Render("provider:"+p))
		}
	}

	// Status filters
	if len(filters.StatusFilter) > 0 {
		var statuses []string
		for s := range filters.StatusFilter {
			statuses = append(statuses, string(s))
		}
		sort.Strings(statuses)
		for _, s := range statuses {
			pills = append(pills, styles.PillActive.Render("status:"+s))
		}
	}

	// Project filters
	if len(filters.ProjectFilter) > 0 {
		projects := filters.ActiveProjects()
		limit := widgets.MinInt(3, len(projects))
		for _, p := range projects[:limit] {
			pills = append(pills, styles.PillActive.Render("project:"+p))
		}
		if len(projects) > limit {
			pills = append(pills, styles.Pill.Render(fmt.Sprintf("+%d more", len(projects)-limit)))
		}
	}

	// Query filter
	if q := strings.TrimSpace(filters.TextQuery); q != "" {
		pills = append(pills, styles.PillActive.Render("query:"+q))
	}

	if len(pills) == 0 {
		return styles.Muted.Render("Filters: none")
	}

	return strings.Join(pills, " ")
}

// RenderLegend renders the status legend with counts
func RenderLegend(counts map[state.Status]int, total int, cost float64, viewMode, sortBy, groupBy, themeName string, styles theme.Styles) string {
	// Status badges with counts
	statusLine := strings.Join([]string{
		fmt.Sprintf("%s %d", widgets.StatusBadgeCompact(state.StatusRunning, styles), counts[state.StatusRunning]),
		fmt.Sprintf("%s %d", widgets.StatusBadgeCompact(state.StatusWaiting, styles), counts[state.StatusWaiting]),
		fmt.Sprintf("%s %d", widgets.StatusBadgeCompact(state.StatusApproval, styles), counts[state.StatusApproval]),
		fmt.Sprintf("%s %d", widgets.StatusBadgeCompact(state.StatusNeedsAttn, styles), counts[state.StatusNeedsAttn]),
		fmt.Sprintf("%s %d", widgets.StatusBadgeCompact(state.StatusStale, styles), counts[state.StatusStale]),
		fmt.Sprintf("%s %d", widgets.StatusBadgeCompact(state.StatusEnded, styles), counts[state.StatusEnded]),
	}, "  ")

	// Metadata
	group := groupBy
	if group == "" {
		group = "none"
	}
	meta := fmt.Sprintf("total:%d  cost:%s  view:%s  sort:%s  group:%s  theme:%s",
		total,
		widgets.FormatCost(cost),
		viewMode,
		sortBy,
		group,
		themeName,
	)

	return styles.Muted.Render(statusLine + "  " + meta)
}

// RenderFilterInput renders the filter input with mode indicator
func RenderFilterInput(inputView string, queryMode string, focused bool, styles theme.Styles) string {
	modeLabel := strings.ToUpper(queryMode)
	modeChip := styles.GroupHeader.Render("[" + modeLabel + "]")
	return styles.Box.Render(modeChip + " " + inputView)
}
