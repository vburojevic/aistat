package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/vburojevic/aistat/internal/app/tui/state"
	"github.com/vburojevic/aistat/internal/app/tui/theme"
	"github.com/vburojevic/aistat/internal/app/tui/widgets"
)

// RenderDetail renders the detail pane for a session with key-value layout
func RenderDetail(s *state.SessionView, styles theme.Styles, width int) string {
	if s == nil {
		return RenderEmptyDetail(styles)
	}

	var b strings.Builder

	// Status with colored icon
	statusIcon := widgets.StatusIcon(s.Status, styles)
	statusText := statusLabel(s.Status)
	b.WriteString(renderRow("Status", statusIcon+" "+statusText, styles))

	// Project
	if s.Project != "" {
		b.WriteString(renderRow("Project", s.Project, styles))
	}

	// Branch
	if s.Branch != "" {
		b.WriteString(renderRow("Branch", s.Branch, styles))
	}

	// Model (with color based on model type)
	if s.Model != "" {
		modelStyled := styledModel(s.Model, styles)
		b.WriteString(renderRow("Model", modelStyled, styles))
	}

	// Cost
	if s.Cost > 0 {
		b.WriteString(renderRow("Cost", fmt.Sprintf("$%.2f", s.Cost), styles))
	}

	// Age
	b.WriteString(renderRow("Age", widgets.FormatAge(s.Age), styles))

	// Provider (with colored icon)
	providerIcon := widgets.ProviderLetterStyled(s.Provider, styles)
	providerName := providerLabel(s.Provider)
	b.WriteString(renderRow("Provider", providerIcon+" "+providerName, styles))

	// Session ID (truncated)
	id := s.ID
	if len(id) > 20 {
		id = id[:8] + "..." + id[len(id)-4:]
	}
	b.WriteString(renderRow("Session", id, styles))

	// Last seen timestamp
	if !s.LastSeen.IsZero() {
		b.WriteString(renderRow("Last Seen", s.LastSeen.In(time.Local).Format("15:04:05"), styles))
	}

	// Last Exchange section (if available)
	if s.LastUser != "" || s.LastAssist != "" {
		b.WriteString("\n")
		b.WriteString(styles.Divider.Render(strings.Repeat("─", minInt(width-4, 30))))
		b.WriteString(" Last Exchange ")
		b.WriteString(styles.Divider.Render(strings.Repeat("─", minInt(width-4, 30))))
		b.WriteString("\n\n")

		if s.LastUser != "" {
			userMsg := truncate(s.LastUser, 100)
			b.WriteString(styles.Label.Render("User:"))
			b.WriteString("  ")
			b.WriteString(userMsg)
			b.WriteString("\n")
		}
		if s.LastAssist != "" {
			assistMsg := truncate(s.LastAssist, 100)
			b.WriteString(styles.Label.Render("Asst:"))
			b.WriteString("  ")
			b.WriteString(assistMsg)
			b.WriteString("\n")
		}
	}

	return b.String()
}

// RenderEmptyDetail renders the empty state for the detail pane
func RenderEmptyDetail(styles theme.Styles) string {
	return styles.Muted.Render("No session selected\n\nUse j/k to navigate")
}

// RenderEmptyState renders when there are no sessions at all
func RenderEmptyState(styles theme.Styles) string {
	var lines []string
	lines = append(lines, styles.Title.Render("No active sessions"))
	lines = append(lines, "")
	lines = append(lines, "Start a Claude Code or Codex session")
	lines = append(lines, "and it will appear here automatically.")
	lines = append(lines, "")
	lines = append(lines, styles.Muted.Render("Tips:"))
	lines = append(lines, "  r  refresh now")
	lines = append(lines, "  a  show all (including ended)")
	lines = append(lines, "  ?  help")
	return strings.Join(lines, "\n")
}

// RenderFilteredEmpty renders when filters hide all sessions
func RenderFilteredEmpty(query string, styles theme.Styles) string {
	var lines []string
	lines = append(lines, styles.Title.Render("No matches"))
	if query != "" {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("Filter: %s", query))
	}
	lines = append(lines, "")
	lines = append(lines, styles.Muted.Render("Press Esc to clear filter"))
	return strings.Join(lines, "\n")
}

// statusLabel returns a human-readable label for a status
func statusLabel(s state.Status) string {
	ui := widgets.ToUIStatus(s)
	switch ui {
	case widgets.UIStatusNeedsInput:
		return "NEEDS INPUT"
	case widgets.UIStatusActive:
		return "ACTIVE"
	default:
		return "IDLE"
	}
}

// providerLabel returns a human-readable name for a provider
func providerLabel(p state.Provider) string {
	switch p {
	case state.ProviderClaude:
		return "Claude"
	case state.ProviderCodex:
		return "Codex"
	default:
		return string(p)
	}
}

// styledModel returns the model name with appropriate color styling
func styledModel(model string, styles theme.Styles) string {
	modelLower := strings.ToLower(model)
	switch {
	case strings.Contains(modelLower, "opus"):
		return styles.ModelOpus.Render(model)
	case strings.Contains(modelLower, "sonnet"):
		return styles.ModelSonnet.Render(model)
	case strings.Contains(modelLower, "haiku"):
		return styles.ModelHaiku.Render(model)
	default:
		return model
	}
}

// renderRow renders a single key-value row
func renderRow(key, value string, styles theme.Styles) string {
	return styles.Label.Render(key) + value + "\n"
}

// truncate truncates a string with ellipsis
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// RenderError renders an error message in the list area
func RenderError(err error, styles theme.Styles) string {
	if err == nil {
		return ""
	}
	var lines []string
	lines = append(lines, styles.ErrorText.Render("⚠ Error loading sessions"))
	lines = append(lines, "")
	lines = append(lines, styles.Muted.Render(err.Error()))
	lines = append(lines, "")
	lines = append(lines, styles.Muted.Render("Press r to retry"))
	return strings.Join(lines, "\n")
}
