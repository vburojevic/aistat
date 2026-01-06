package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/vburojevic/aistat/internal/app/tui/state"
	"github.com/vburojevic/aistat/internal/app/tui/theme"
)

// RenderSessionDetail renders the detail view for a session
func RenderSessionDetail(s state.SessionView, styles theme.Styles) string {
	lines := strings.Split(strings.TrimSpace(s.Detail), "\n")

	// Find max key width for alignment
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
			b.WriteString(styles.Muted.Render(key + pad + " : "))
			b.WriteString(val)
			b.WriteString("\n")
			continue
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString(styles.Muted.Render("Seen at: "))
	b.WriteString(s.LastSeen.In(time.Local).Format("2006-01-02 15:04:05"))
	b.WriteString("\n")

	return b.String()
}

// RenderProjectDetail renders the detail view for a project in the dashboard
func RenderProjectDetail(p state.ProjectItem, styles theme.Styles) string {
	var b strings.Builder

	b.WriteString(styles.Bold.Render("Project: "))
	b.WriteString(p.Name)
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("Total:     %d\n", p.Count))
	b.WriteString(fmt.Sprintf("Running:   %d\n", p.StatusCount[state.StatusRunning]))
	b.WriteString(fmt.Sprintf("Waiting:   %d\n", p.StatusCount[state.StatusWaiting]))
	b.WriteString(fmt.Sprintf("Approval:  %d\n", p.StatusCount[state.StatusApproval]))
	b.WriteString(fmt.Sprintf("Attention: %d\n", p.StatusCount[state.StatusNeedsAttn]))
	b.WriteString(fmt.Sprintf("Stale:     %d\n", p.StatusCount[state.StatusStale]))
	b.WriteString(fmt.Sprintf("Ended:     %d\n", p.StatusCount[state.StatusEnded]))

	// Providers
	if len(p.Providers) > 0 {
		b.WriteString("\n")
		var providers []string
		for prov, count := range p.Providers {
			providers = append(providers, fmt.Sprintf("%s: %d", prov, count))
		}
		b.WriteString("Providers: ")
		b.WriteString(strings.Join(providers, ", "))
		b.WriteString("\n")
	}

	// Last seen
	if !p.LastSeen.IsZero() {
		b.WriteString("\n")
		b.WriteString(styles.Muted.Render("Last seen: "))
		b.WriteString(p.LastSeen.In(time.Local).Format("2006-01-02 15:04"))
		b.WriteString("\n")
	}

	return b.String()
}

// RenderEmptyState renders the empty state view
func RenderEmptyState(hasAnySessions bool, query string, styles theme.Styles) string {
	var lines []string

	if !hasAnySessions {
		lines = append(lines, styles.Title.Render("No sessions found"))
		lines = append(lines, "Start a Claude or Codex session and come back.")
		lines = append(lines, "")
		lines = append(lines, "Next steps:")
		lines = append(lines, "• press r to refresh")
		lines = append(lines, "• press p to view projects")
		lines = append(lines, "• run aistat doctor --fix")
		lines = append(lines, "• check config: aistat config show")
		lines = append(lines, "• toggle filters (1/2/R/W/E/S/Z/N)")
	} else {
		lines = append(lines, styles.Title.Render("No matches for current filters"))
		if query != "" {
			lines = append(lines, fmt.Sprintf("Query: %q", query))
		}
		lines = append(lines, "")
		lines = append(lines, "Try:")
		lines = append(lines, "• press esc to clear the filter")
		lines = append(lines, "• press p to choose projects")
		lines = append(lines, "• edit search: /  (examples: p:myproj, s:running)")
		lines = append(lines, "• toggle filters (1/2/R/W/E/S/Z/N)")
	}

	return styles.DetailBox.Render(strings.Join(lines, "\n"))
}

// RenderError renders an error message
func RenderError(err error, styles theme.Styles) string {
	if err == nil {
		return ""
	}
	return styles.Muted.Render("⚠ " + err.Error())
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
