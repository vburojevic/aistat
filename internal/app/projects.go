package app

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	prettytable "github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
)

type ProjectStat struct {
	Name        string           `json:"name"`
	Count       int              `json:"count"`
	LastSeen    time.Time        `json:"last_seen"`
	StatusCount map[Status]int   `json:"status_count"`
	Providers   map[Provider]int `json:"providers"`
}

func newProjectsCmd() *cobra.Command {
	var (
		flagJSON     bool
		flagProvider string
		flagSort     string
		flagAll      bool
	)

	cmd := &cobra.Command{
		Use:   "projects",
		Short: "List all projects with counts and last activity",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadConfig()
			cfg.ProviderFilter = strings.TrimSpace(strings.ToLower(flagProvider))
			cfg.ProjectFilters = nil
			cfg.StatusFilters = nil
			cfg.IncludeEnded = flagAll
			if cfg.IncludeEnded {
				cfg.AllScanWindow = defaultAllScanWindow
			}
			cfg.MaxSessions = 0
			cfg.GroupBy = ""

			stats, err := gatherProjectStats(cfg)
			if err != nil {
				return err
			}
			sortProjectStats(stats, flagSort)

			if flagJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(stats)
			}
			renderProjectsTable(stats)
			return nil
		},
	}

	cmd.Flags().BoolVar(&flagJSON, "json", false, "Output JSON instead of table")
	cmd.Flags().StringVar(&flagProvider, "provider", "", "Filter by provider: claude|codex")
	cmd.Flags().StringVar(&flagSort, "sort", "count", "Sort by: count|name|last_seen")
	cmd.Flags().BoolVar(&flagAll, "all", false, "Include ended/stale sessions")
	return cmd
}

func gatherProjectStats(cfg Config) ([]ProjectStat, error) {
	sessions, err := gatherSessions(cfg)
	if err != nil {
		return nil, err
	}
	byName := map[string]*ProjectStat{}
	for _, s := range sessions {
		if s.Project == "" {
			continue
		}
		key := strings.ToLower(s.Project)
		stat := byName[key]
		if stat == nil {
			stat = &ProjectStat{
				Name:        s.Project,
				StatusCount: map[Status]int{},
				Providers:   map[Provider]int{},
			}
			byName[key] = stat
		}
		stat.Count++
		stat.StatusCount[s.Status]++
		stat.Providers[s.Provider]++
		if s.LastSeen.After(stat.LastSeen) {
			stat.LastSeen = s.LastSeen
		}
	}

	out := make([]ProjectStat, 0, len(byName))
	for _, stat := range byName {
		out = append(out, *stat)
	}
	return out, nil
}

func sortProjectStats(stats []ProjectStat, sortBy string) {
	key := strings.ToLower(strings.TrimSpace(sortBy))
	if key == "" {
		key = "count"
	}
	sort.SliceStable(stats, func(i, j int) bool {
		a := stats[i]
		b := stats[j]
		switch key {
		case "name":
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		case "last_seen":
			return a.LastSeen.After(b.LastSeen)
		default:
			if a.Count != b.Count {
				return a.Count > b.Count
			}
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		}
	})
}

func renderProjectsTable(stats []ProjectStat) {
	tw := prettytable.NewWriter()
	tw.SetOutputMirror(os.Stdout)
	tw.SetStyle(prettytable.StyleLight)
	tw.Style().Options.SeparateRows = false

	headers := prettytable.Row{"PROJECT", "COUNT", "LAST", "RUN", "WAIT", "APPR", "STALE", "END"}
	tw.AppendHeader(headers)
	tw.SetColumnConfigs([]prettytable.ColumnConfig{
		{Number: 2, Align: text.AlignRight},
		{Number: 4, Align: text.AlignRight},
		{Number: 5, Align: text.AlignRight},
		{Number: 6, Align: text.AlignRight},
		{Number: 7, Align: text.AlignRight},
		{Number: 8, Align: text.AlignRight},
	})

	for _, p := range stats {
		last := ""
		if !p.LastSeen.IsZero() {
			last = p.LastSeen.In(time.Local).Format("2006-01-02 15:04")
		}
		tw.AppendRow(prettytable.Row{
			p.Name,
			p.Count,
			last,
			p.StatusCount[StatusRunning],
			p.StatusCount[StatusWaiting],
			p.StatusCount[StatusApproval],
			p.StatusCount[StatusStale],
			p.StatusCount[StatusEnded],
		})
	}
	if len(stats) == 0 {
		fmt.Println("No projects found.")
		return
	}
	tw.Render()
}
