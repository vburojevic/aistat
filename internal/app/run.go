package app

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// Run executes the CLI and returns a process exit code.
func Run() int {
	if runtime.GOOS != "darwin" {
		fmt.Fprintln(os.Stderr, "This build targets macOS only (per spec).")
		return 2
	}

	baseCfg := loadConfig()

	var (
		flagJSON          bool
		flagWatch         bool
		flagNoTUI         bool
		flagProvider      string
		flagAll           bool
		flagRedact        bool
		flagActiveWindow  string
		flagRunningWindow string
		flagRefreshEvery  string
		flagMax           int
		flagNoColor       bool
		flagProjects      []string
		flagStatus        []string
		flagFields        []string
		flagSortBy        string
		flagGroupBy       string
		flagIncludeLast   bool
	)

	rootCmd := &cobra.Command{
		Use:   appName,
		Short: "List active Claude Code and Codex sessions (with real-time statuses)",
		Long:  "aistat collects session events from Claude Code hooks/statusline and from Codex rollout logs/notify, then renders a slick live list.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fieldsExplicit := cmd.Flags().Changed("fields")
			cfg, err := cfgFromFlags(baseCfg, flagProvider, flagAll, flagRedact, flagActiveWindow, flagRunningWindow, flagRefreshEvery, flagMax, flagNoColor, flagProjects, flagStatus, flagFields, fieldsExplicit, flagSortBy, flagGroupBy, flagIncludeLast)
			if err != nil {
				return err
			}

			// Default behavior:
			// - If stdout is a TTY and --no-tui not set and --json not set => TUI
			// - Else => list once (or watch if --watch)
			if term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd())) && !flagNoTUI && !flagJSON && cfg.GroupBy == "" {
				return runTUINew(cfg)
			}
			return runList(cfg, flagJSON, flagWatch)
		},
	}

	rootCmd.Flags().BoolVar(&flagJSON, "json", false, "Output JSON instead of a table/TUI")
	rootCmd.Flags().BoolVar(&flagWatch, "watch", false, "Continuously refresh output (non-TUI)")
	rootCmd.Flags().BoolVar(&flagNoTUI, "no-tui", false, "Force non-interactive output even on a TTY")

	rootCmd.Flags().StringVar(&flagProvider, "provider", "", "Filter by provider: claude|codex (default: both)")
	rootCmd.Flags().BoolVar(&flagAll, "all", false, "Include ended/stale sessions too (scans a wider window)")
	rootCmd.Flags().BoolVar(&flagRedact, "redact", baseCfg.Redact, "Redact paths/IDs (recommended; default from config)")
	rootCmd.Flags().StringVar(&flagActiveWindow, "active-window", baseCfg.ActiveWindow.String(), "Consider a session 'active' if seen within this duration (e.g. 30m)")
	rootCmd.Flags().StringVar(&flagRunningWindow, "running-window", baseCfg.RunningWindow.String(), "Consider a session 'running' if last activity is within this duration (e.g. 3s)")
	rootCmd.Flags().StringVar(&flagRefreshEvery, "refresh", baseCfg.RefreshEvery.String(), "Refresh interval for watch/TUI (e.g. 1s)")
	rootCmd.Flags().IntVar(&flagMax, "max", baseCfg.MaxSessions, "Maximum sessions to show")
	rootCmd.Flags().BoolVar(&flagNoColor, "no-color", false, "Disable color output (TUI + table)")
	rootCmd.Flags().StringSliceVar(&flagProjects, "project", nil, "Filter by project name (repeatable or comma-separated)")
	rootCmd.Flags().StringSliceVar(&flagStatus, "status", nil, "Filter by status: running|waiting|approval|stale|ended|needs_attention")
	rootCmd.Flags().StringSliceVar(&flagFields, "fields", nil, "Output fields (comma-separated or repeatable)")
	rootCmd.Flags().StringVar(&flagSortBy, "sort", "last_seen", "Sort by: last_seen|status|provider|cost|project")
	rootCmd.Flags().StringVar(&flagGroupBy, "group-by", "", "Group by: provider|project|status|day|hour (non-TUI only)")
	rootCmd.Flags().BoolVar(&flagIncludeLast, "include-last-msg", false, "Include last user/assistant messages when available")

	// install
	rootCmd.AddCommand(newInstallCmd())
	// doctor
	rootCmd.AddCommand(newDoctorCmd())

	// ingest (hidden)
	ingest := &cobra.Command{
		Use:    "ingest",
		Short:  "Internal: ingest provider hooks/notify JSON from stdin",
		Hidden: true,
	}
	ingest.AddCommand(&cobra.Command{
		Use:    "claude-hook",
		Short:  "Internal: ingest Claude Code hook events",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ingestClaudeHook(os.Stdin)
		},
	})
	ingest.AddCommand(&cobra.Command{
		Use:    "codex-notify",
		Short:  "Internal: ingest Codex notify payloads",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ingestCodexNotify(os.Stdin)
		},
	})
	rootCmd.AddCommand(ingest)

	// statusline (hidden)
	rootCmd.AddCommand(&cobra.Command{
		Use:    "statusline",
		Short:  "Internal: Claude Code statusLine command (reads JSON from stdin; prints one line)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			line, err := ingestClaudeStatusline(os.Stdin)
			if err != nil {
				// Statusline must never be noisy; fall back to empty line.
				fmt.Println("")
				return nil
			}
			fmt.Println(line)
			return nil
		},
	})

	// config
	rootCmd.AddCommand(newConfigCmd())
	// tail
	rootCmd.AddCommand(newTailCmd())
	// show
	rootCmd.AddCommand(newShowCmd())
	// summary
	rootCmd.AddCommand(newSummaryCmd())
	// projects
	rootCmd.AddCommand(newProjectsCmd())
	// help (agent-friendly)
	rootCmd.AddCommand(newHelpCmd())

	if err := rootCmd.Execute(); err != nil {
		return 1
	}
	return 0
}

func cfgFromFlags(base Config, provider string, all bool, redact bool, activeWinStr, runningWinStr, refreshStr string, max int, noColor bool, projects []string, statuses []string, fields []string, fieldsExplicit bool, sortBy string, groupBy string, includeLast bool) (Config, error) {
	cfg := base

	cfg.ProviderFilter = strings.TrimSpace(strings.ToLower(provider))
	cfg.IncludeEnded = all
	cfg.Redact = redact
	cfg.NoColor = noColor
	cfg.ProjectFilters = normalizeList(projects)
	statusFilters, err := parseStatusFilters(normalizeList(statuses))
	if err != nil {
		return Config{}, err
	}
	cfg.StatusFilters = statusFilters
	fieldList, err := parseFields(fields, base.Fields)
	if err != nil {
		return Config{}, err
	}
	cfg.Fields = fieldList
	cfg.FieldsExplicit = fieldsExplicit
	cfg.SortBy = strings.TrimSpace(strings.ToLower(sortBy))
	cfg.GroupBy = strings.TrimSpace(strings.ToLower(groupBy))
	cfg.IncludeLastMsg = includeLast

	if d, err := time.ParseDuration(activeWinStr); err == nil && d > 0 {
		cfg.ActiveWindow = d
	} else if err != nil {
		return Config{}, fmt.Errorf("invalid --active-window: %w", err)
	}

	if d, err := time.ParseDuration(runningWinStr); err == nil && d > 0 {
		cfg.RunningWindow = d
	} else if err != nil {
		return Config{}, fmt.Errorf("invalid --running-window: %w", err)
	}

	if d, err := time.ParseDuration(refreshStr); err == nil && d > 0 {
		cfg.RefreshEvery = d
	} else if err != nil {
		return Config{}, fmt.Errorf("invalid --refresh: %w", err)
	}

	if max > 0 {
		cfg.MaxSessions = max
	}

	// Wider scan window when --all is set
	if cfg.IncludeEnded {
		cfg.AllScanWindow = defaultAllScanWindow
	}

	if cfg.SortBy == "" {
		cfg.SortBy = "last_seen"
	}

	switch cfg.SortBy {
	case "last_seen", "status", "provider", "cost", "project":
		// ok
	default:
		return Config{}, fmt.Errorf("invalid --sort: %s", cfg.SortBy)
	}

	if cfg.GroupBy != "" {
		switch cfg.GroupBy {
		case "provider", "project", "status", "day", "hour":
			// ok
		default:
			return Config{}, fmt.Errorf("invalid --group-by: %s", cfg.GroupBy)
		}
	}

	return cfg, nil
}
