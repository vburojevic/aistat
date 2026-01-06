package views

import (
	"strings"

	"github.com/vburojevic/aistat/internal/app/tui/theme"
)

// RenderHelpOverlay renders the help overlay
func RenderHelpOverlay(styles theme.Styles) string {
	lines := []string{
		styles.Section.Render("◆ Help"),
		"",
		styles.Bold.Render("Navigation"),
		"  ↑/↓ or j/k   Move cursor",
		"  Enter        Confirm/activate",
		"  Esc          Cancel/close/clear",
		"  Tab          Toggle dashboard/list",
		"",
		styles.Bold.Render("Search & Filter"),
		"  /            Start filtering",
		"  p:query      Filter by project",
		"  s:query      Filter by status",
		"  :            Open command palette",
		"",
		styles.Bold.Render("Views"),
		"  p            Projects picker",
		"  d            Toggle detail mode (split/full)",
		"  b            Toggle sidebar",
		"  v            Cycle view mode (full/compact/ultra/card)",
		"  m            Toggle last message column",
		"",
		styles.Bold.Render("Sorting & Grouping"),
		"  s            Cycle sort (last_seen/status/provider/cost/project)",
		"  g            Cycle grouping (none/provider/project/status/day/hour)",
		"",
		styles.Bold.Render("Filters"),
		"  1/2          Toggle Claude/Codex provider",
		"  R/W/E/S/Z/N  Toggle status (Running/Waiting/Approval/Stale/Ended/Attn)",
		"",
		styles.Bold.Render("Actions"),
		"  space        Select/deselect session",
		"  P            Pin/unpin session",
		"  y            Copy session ID(s)",
		"  D            Copy detail panel",
		"  o            Open session log file",
		"  a            Jump to approval status",
		"  u            Jump to running status",
		"",
		styles.Bold.Render("Display"),
		"  t            Cycle theme (mocha/frappe/latte)",
		"  A            Toggle accessible mode",
		"  c            Toggle redaction mode",
		"  r            Manual refresh",
		"",
		styles.Bold.Render("Quit"),
		"  q or Ctrl+C  Exit application",
		"",
		styles.Muted.Render("Press ? or Esc to close this help"),
	}

	return styles.OverlayBox.Render(strings.Join(lines, "\n"))
}
