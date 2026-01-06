package app

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// -------------------------
// Help (agent-friendly)
// -------------------------

type helpFlag struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Default     string `json:"default"`
	Description string `json:"description"`
}

type helpCommand struct {
	Name        string     `json:"name"`
	Usage       string     `json:"usage"`
	Description string     `json:"description"`
	Flags       []helpFlag `json:"flags,omitempty"`
}

type helpDoc struct {
	Name        string            `json:"name"`
	OneLiner    string            `json:"one_liner"`
	Usage       []string          `json:"usage"`
	Commands    []helpCommand     `json:"commands"`
	GlobalFlags []helpFlag        `json:"global_flags"`
	IOContract  map[string]string `json:"io_contract"`
	ExitCodes   map[string]string `json:"exit_codes"`
	Env         map[string]string `json:"env"`
	Config      map[string]string `json:"config"`
	Notes       []string          `json:"notes"`
}

func newHelpCmd() *cobra.Command {
	var format string
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "help",
		Short: "Show extended help (agent-friendly)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if jsonOut {
				format = "json"
			}
			format = strings.TrimSpace(strings.ToLower(format))
			if format == "" {
				format = "text"
			}

			doc := buildHelpDoc()

			switch format {
			case "json":
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(doc)
			default:
				fmt.Fprint(cmd.OutOrStdout(), renderHelpText(doc))
				return nil
			}
		},
	}

	cmd.Flags().StringVar(&format, "format", "text", "Output format: text|json")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output JSON")
	return cmd
}

func buildHelpDoc() helpDoc {
	global := []helpFlag{
		{Name: "--json", Type: "bool", Default: "false", Description: "Output JSON instead of a table/TUI"},
		{Name: "--watch", Type: "bool", Default: "false", Description: "Continuously refresh output (non-TUI); with --json emits NDJSON"},
		{Name: "--no-tui", Type: "bool", Default: "false", Description: "Force non-interactive output even on a TTY"},
		{Name: "--provider", Type: "string", Default: "", Description: "Filter by provider: claude|codex"},
		{Name: "--project", Type: "string[]", Default: "", Description: "Filter by project name (repeatable or comma-separated)"},
		{Name: "--status", Type: "string[]", Default: "", Description: "Filter by status (repeatable or comma-separated)"},
		{Name: "--fields", Type: "string[]", Default: "", Description: "Select output columns (comma-separated or repeatable)"},
		{Name: "--sort", Type: "string", Default: "last_seen", Description: "Sort by: last_seen|status|provider|cost|project"},
		{Name: "--group-by", Type: "string", Default: "", Description: "Group by: provider|project|status|day|hour (non-TUI)"},
		{Name: "--include-last-msg", Type: "bool", Default: "false", Description: "Include last message snippets when available"},
		{Name: "--all", Type: "bool", Default: "false", Description: "Include ended/stale sessions (wider scan window)"},
		{Name: "--redact", Type: "bool", Default: "true", Description: "Redact paths/IDs"},
		{Name: "--active-window", Type: "duration", Default: "30m", Description: "Active session window"},
		{Name: "--running-window", Type: "duration", Default: "3s", Description: "Running activity window"},
		{Name: "--refresh", Type: "duration", Default: "1s", Description: "Refresh interval"},
		{Name: "--max", Type: "int", Default: "50", Description: "Maximum sessions to show"},
		{Name: "--no-color", Type: "bool", Default: "false", Description: "Disable color output"},
	}

	commands := []helpCommand{
		{Name: "aistat", Usage: "aistat [flags]", Description: "List sessions (TUI on TTY unless --no-tui or --json)"},
		{Name: "projects", Usage: "aistat projects [--json] [--sort count|name|last_seen]", Description: "List projects with counts and last activity"},
		{Name: "show", Usage: "aistat show <id> [--json]", Description: "Show details for a single session"},
		{Name: "summary", Usage: "aistat summary [--group-by project] [--json]", Description: "Summarize sessions by group"},
		{Name: "tail", Usage: "aistat tail <id> [--follow]", Description: "Tail a session transcript/log"},
		{Name: "install", Usage: "aistat install [flags]", Description: "Install Claude/Codex integrations"},
		{Name: "doctor", Usage: "aistat doctor [--fix]", Description: "Check setup and optionally auto-fix"},
		{Name: "config", Usage: "aistat config --show|--init", Description: "Show or initialize config"},
		{Name: "help", Usage: "aistat help [--format json]", Description: "Extended help for humans/agents"},
	}

	return helpDoc{
		Name:     appName,
		OneLiner: "List active Claude Code and Codex sessions (with real-time statuses)",
		Usage: []string{
			"aistat [flags]",
			"aistat projects [flags]",
			"aistat show <id> [flags]",
			"aistat summary [flags]",
			"aistat tail <id> [flags]",
			"aistat install [flags]",
			"aistat doctor [--fix]",
			"aistat config --show|--init",
			"aistat help [--format json]",
		},
		Commands:    commands,
		GlobalFlags: global,
		IOContract: map[string]string{
			"stdout": "Primary data output (table/TUI/JSON).",
			"stderr": "Diagnostics and errors.",
		},
		ExitCodes: map[string]string{
			"0": "Success",
			"1": "Generic failure",
			"2": "Invalid usage or unsupported platform",
		},
		Env: map[string]string{
			"AISTAT_HOME": "Override app data directory",
			"CODEX_HOME":  "Override Codex home directory",
			"ACCESSIBLE":  "Enable accessible install wizard",
		},
		Config: map[string]string{
			"path": "~/Library/Application Support/aistat/config.json",
		},
		Notes: []string{
			"Use `--watch --json` to stream NDJSON for dashboards.",
			"TUI keybinds: / filter, : palette, p projects, s sort, g group, v view, m last-msg, b sidebar.",
		},
	}
}

func renderHelpText(doc helpDoc) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s — %s\n\n", doc.Name, doc.OneLiner))

	b.WriteString("USAGE\n")
	for _, u := range doc.Usage {
		b.WriteString("  " + u + "\n")
	}
	b.WriteString("\nCOMMANDS\n")
	for _, c := range doc.Commands {
		b.WriteString(fmt.Sprintf("  %-8s %s\n", c.Name, c.Description))
	}
	b.WriteString("\nGLOBAL FLAGS\n")
	for _, f := range doc.GlobalFlags {
		b.WriteString(fmt.Sprintf("  %-18s %-7s %-6s %s\n", f.Name, f.Type, f.Default, f.Description))
	}
	b.WriteString("\nI/O CONTRACT\n")
	for k, v := range doc.IOContract {
		b.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
	}
	b.WriteString("\nEXIT CODES\n")
	for k, v := range doc.ExitCodes {
		b.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
	}
	b.WriteString("\nENV\n")
	for k, v := range doc.Env {
		b.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
	}
	b.WriteString("\nCONFIG\n")
	for k, v := range doc.Config {
		b.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
	}
	if len(doc.Notes) > 0 {
		b.WriteString("\nNOTES\n")
		for _, n := range doc.Notes {
			b.WriteString("  - " + n + "\n")
		}
	}
	return b.String()
}
