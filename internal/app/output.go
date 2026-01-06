package app

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	prettytable "github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"golang.org/x/term"
)

// -------------------------
// Output (non-TUI)
// -------------------------

func runList(cfg Config, asJSON bool, watch bool) error {
	if watch {
		for {
			if asJSON {
				if err := renderJSONStream(cfg); err != nil {
					return err
				}
			} else {
				if err := renderOnce(cfg, false); err != nil {
					return err
				}
				// Clear screen between renders (nice watch UX)
				if term.IsTerminal(int(os.Stdout.Fd())) {
					fmt.Print("\033[H\033[2J")
				}
			}
			time.Sleep(cfg.RefreshEvery)
		}
	}
	return renderOnce(cfg, asJSON)
}

func renderOnce(cfg Config, asJSON bool) error {
	sessions, err := gatherSessions(cfg)
	if err != nil {
		return err
	}

	if asJSON {
		return renderJSONSnapshot(cfg, sessions)
	}

	if len(sessions) == 0 {
		fmt.Println("No active sessions found.")
		return nil
	}

	if cfg.GroupBy == "" {
		renderTable(sessions, cfg.Fields)
		return nil
	}

	groups := groupSessions(sessions, cfg.GroupBy)
	for i, group := range groups {
		if i > 0 {
			fmt.Println()
		}
		label := group.Group
		if label == "" {
			label = "unknown"
		}
		fmt.Printf("== %s\n", label)
		renderTable(group.Sessions, cfg.Fields)
	}
	return nil
}

func renderTable(sessions []SessionView, fields []string) {
	tw := prettytable.NewWriter()
	tw.SetOutputMirror(os.Stdout)
	tw.SetStyle(prettytable.StyleLight)
	tw.Style().Options.SeparateRows = false

	if len(fields) == 0 {
		fields = defaultFields()
	}

	headers := make(prettytable.Row, 0, len(fields))
	var configs []prettytable.ColumnConfig
	for i, f := range fields {
		headers = append(headers, strings.ToUpper(f))
		switch f {
		case "age", "cost":
			configs = append(configs, prettytable.ColumnConfig{Number: i + 1, Align: text.AlignRight})
		}
	}
	tw.AppendHeader(headers)
	if len(configs) > 0 {
		tw.SetColumnConfigs(configs)
	}

	for _, s := range sessions {
		row := make(prettytable.Row, 0, len(fields))
		for _, f := range fields {
			row = append(row, fieldValue(s, f))
		}
		tw.AppendRow(row)
	}
	tw.Render()
}

func fieldValue(s SessionView, field string) string {
	switch field {
	case "provider":
		return string(s.Provider)
	case "status":
		return string(s.Status)
	case "id":
		return s.ID
	case "project":
		return s.Project
	case "dir":
		return s.Dir
	case "model":
		return s.Model
	case "age":
		return fmtAgo(s.Age)
	case "since":
		if s.LastSeen.IsZero() {
			return ""
		}
		return s.LastSeen.In(time.Local).Format(time.RFC3339)
	case "cost":
		return formatCost(s.Cost)
	case "last_user":
		return s.LastUser
	case "last_assistant":
		return s.LastAssist
	default:
		return ""
	}
}

func sessionsToMaps(sessions []SessionView, fields []string) []map[string]any {
	if len(fields) == 0 {
		fields = defaultFields()
	}
	out := make([]map[string]any, 0, len(sessions))
	for _, s := range sessions {
		m := map[string]any{}
		for _, f := range fields {
			m[strings.ToLower(f)] = fieldValue(s, f)
		}
		out = append(out, m)
	}
	return out
}

func renderJSONSnapshot(cfg Config, sessions []SessionView) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	payload := jsonPayload(cfg, sessions)
	return enc.Encode(payload)
}

func renderJSONStream(cfg Config) error {
	sessions, err := gatherSessions(cfg)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	payload := jsonPayload(cfg, sessions)
	wrapped := map[string]any{
		"ts":   time.Now().UTC().Format(time.RFC3339),
		"data": payload,
	}
	return enc.Encode(wrapped)
}

func jsonPayload(cfg Config, sessions []SessionView) any {
	if cfg.FieldsExplicit {
		if cfg.GroupBy == "" {
			return sessionsToMaps(sessions, cfg.Fields)
		}
		groups := groupSessions(sessions, cfg.GroupBy)
		out := make([]map[string]any, 0, len(groups))
		for _, g := range groups {
			out = append(out, map[string]any{
				"group":    g.Group,
				"sessions": sessionsToMaps(g.Sessions, cfg.Fields),
			})
		}
		return out
	}
	if cfg.GroupBy != "" {
		return groupSessions(sessions, cfg.GroupBy)
	}
	return sessions
}
