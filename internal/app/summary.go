package app

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
)

// -------------------------
// Summary
// -------------------------

type summaryRow struct {
	Group    string  `json:"group"`
	Total    int     `json:"total"`
	Running  int     `json:"running"`
	Waiting  int     `json:"waiting"`
	Approval int     `json:"approval"`
	Stale    int     `json:"stale"`
	Ended    int     `json:"ended"`
	Attn     int     `json:"needs_attention"`
	Cost     float64 `json:"cost_usd"`
}

func newSummaryCmd() *cobra.Command {
	var (
		groupBy string
		jsonOut bool
	)

	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Summarize sessions by project/provider/status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadConfig()
			cfg.GroupBy = strings.TrimSpace(strings.ToLower(groupBy))
			if cfg.GroupBy == "" {
				cfg.GroupBy = "project"
			}
			sessions, err := gatherSessions(cfg)
			if err != nil {
				return err
			}
			rows := summarizeSessions(sessions, cfg.GroupBy)

			if jsonOut {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(rows)
			}

			renderSummaryTable(rows)
			return nil
		},
	}

	cmd.Flags().StringVar(&groupBy, "group-by", "project", "Group by: provider|project|status|day|hour")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output JSON")
	return cmd
}

func summarizeSessions(views []SessionView, groupBy string) []summaryRow {
	groups := groupSessions(views, groupBy)
	rows := make([]summaryRow, 0, len(groups))
	for _, g := range groups {
		row := summaryRow{Group: g.Group}
		for _, s := range g.Sessions {
			row.Total++
			row.Cost += s.Cost
			switch s.Status {
			case StatusRunning:
				row.Running++
			case StatusWaiting:
				row.Waiting++
			case StatusApproval:
				row.Approval++
			case StatusStale:
				row.Stale++
			case StatusEnded:
				row.Ended++
			case StatusNeedsAttn:
				row.Attn++
			}
		}
		rows = append(rows, row)
	}
	return rows
}

func renderSummaryTable(rows []summaryRow) {
	if len(rows) == 0 {
		fmt.Fprintln(os.Stdout, "No sessions found.")
		return
	}

	tw := table.NewWriter()
	tw.SetOutputMirror(os.Stdout)
	tw.SetStyle(table.StyleLight)
	tw.Style().Options.SeparateRows = false

	tw.AppendHeader(table.Row{"GROUP", "TOTAL", "RUN", "WAIT", "APPR", "ATTN", "STALE", "END", "COST"})
	tw.SetColumnConfigs([]table.ColumnConfig{
		{Number: 2, Align: text.AlignRight},
		{Number: 3, Align: text.AlignRight},
		{Number: 4, Align: text.AlignRight},
		{Number: 5, Align: text.AlignRight},
		{Number: 6, Align: text.AlignRight},
		{Number: 7, Align: text.AlignRight},
		{Number: 8, Align: text.AlignRight},
		{Number: 9, Align: text.AlignRight},
	})

	for _, r := range rows {
		cost := ""
		if r.Cost > 0 {
			cost = fmt.Sprintf("$%.3f", r.Cost)
		}
		group := r.Group
		if strings.TrimSpace(group) == "" {
			group = "unknown"
		}
		tw.AppendRow(table.Row{group, r.Total, r.Running, r.Waiting, r.Approval, r.Attn, r.Stale, r.Ended, cost})
	}
	tw.Render()
}
