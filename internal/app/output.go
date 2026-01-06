package app

import (
	"encoding/json"
	"fmt"
	"os"
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
			if err := renderOnce(cfg, asJSON); err != nil {
				return err
			}
			time.Sleep(cfg.RefreshEvery)
			// Clear screen between renders (nice watch UX)
			if !asJSON && term.IsTerminal(int(os.Stdout.Fd())) {
				fmt.Print("\033[H\033[2J")
			}
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
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(sessions)
	}

	if len(sessions) == 0 {
		fmt.Println("No active sessions found.")
		return nil
	}

	tw := prettytable.NewWriter()
	tw.SetOutputMirror(os.Stdout)
	tw.SetStyle(prettytable.StyleLight)
	tw.Style().Options.SeparateRows = false

	tw.AppendHeader(prettytable.Row{"PROVIDER", "STATUS", "ID", "PROJECT", "DIR", "MODEL", "AGE", "COST"})
	tw.SetColumnConfigs([]prettytable.ColumnConfig{
		{Number: 7, Align: text.AlignRight},
		{Number: 8, Align: text.AlignRight},
	})

	for _, s := range sessions {
		cost := ""
		if s.Cost > 0 {
			cost = fmt.Sprintf("$%.3f", s.Cost)
		}
		tw.AppendRow(prettytable.Row{
			string(s.Provider),
			string(s.Status),
			s.ID,
			s.Project,
			s.Dir,
			s.Model,
			fmtAgo(s.Age),
			cost,
		})
	}
	tw.Render()
	return nil
}
