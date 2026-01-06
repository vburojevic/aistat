package app

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

func defaultConfig() Config {
	return Config{
		Redact:         true,
		ActiveWindow:   defaultActiveWindow,
		RunningWindow:  defaultRunningWindow,
		RefreshEvery:   defaultRefreshInterval,
		MaxSessions:    defaultMaxSessions,
		IncludeEnded:   false,
		ProviderFilter: "",
		NoColor:        false,
		AllScanWindow:  defaultAllScanWindow,

		TailBytesCodex:     defaultTailBytesCodex,
		TailBytesClaude:    defaultTailBytesClaude,
		HeaderScanLines:    defaultHeaderScanLines,
		StatuslineMinWrite: defaultStatuslineMinWrite,
	}
}

func loadConfig() Config {
	cfg := defaultConfig()

	p, err := configFilePath()
	if err != nil {
		return cfg
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return cfg
	}
	var cf ConfigFile
	if err := json.Unmarshal(b, &cf); err != nil {
		return cfg
	}
	if cf.Redact != nil {
		cfg.Redact = *cf.Redact
	}
	if cf.ActiveWindow != "" {
		if d, err := time.ParseDuration(cf.ActiveWindow); err == nil && d > 0 {
			cfg.ActiveWindow = d
		}
	}
	if cf.RunningWindow != "" {
		if d, err := time.ParseDuration(cf.RunningWindow); err == nil && d > 0 {
			cfg.RunningWindow = d
		}
	}
	if cf.RefreshEvery != "" {
		if d, err := time.ParseDuration(cf.RefreshEvery); err == nil && d > 0 {
			cfg.RefreshEvery = d
		}
	}
	if cf.MaxSessions != nil && *cf.MaxSessions > 0 {
		cfg.MaxSessions = *cf.MaxSessions
	}
	if cf.AllScanWindow != "" {
		if d, err := time.ParseDuration(cf.AllScanWindow); err == nil && d > 0 {
			cfg.AllScanWindow = d
		}
	}
	if cf.StatuslineMinWrite != "" {
		if d, err := time.ParseDuration(cf.StatuslineMinWrite); err == nil && d > 0 {
			cfg.StatuslineMinWrite = d
		}
	}
	return cfg
}

func newConfigCmd() *cobra.Command {
	var (
		show bool
		init bool
	)
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show or initialize aistat config",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := configFilePath()
			if err != nil {
				return err
			}
			if init {
				if err := ensureAppDirs(); err != nil {
					return err
				}
				if _, err := os.Stat(p); err == nil {
					fmt.Printf("Config already exists: %s\n", p)
					return nil
				}
				cf := ConfigFile{
					Redact:             ptrBool(true),
					ActiveWindow:       defaultActiveWindow.String(),
					RunningWindow:      defaultRunningWindow.String(),
					RefreshEvery:       defaultRefreshInterval.String(),
					MaxSessions:        ptrInt(defaultMaxSessions),
					AllScanWindow:      defaultAllScanWindow.String(),
					StatuslineMinWrite: defaultStatuslineMinWrite.String(),
				}
				b, _ := json.MarshalIndent(cf, "", "  ")
				if err := os.WriteFile(p, b, 0o600); err != nil {
					return err
				}
				fmt.Printf("Wrote %s\n", p)
				return nil
			}
			if show {
				cfg := loadConfig()
				fmt.Printf("Config file: %s\n", p)
				fmt.Printf("  redact: %v\n", cfg.Redact)
				fmt.Printf("  active_window: %s\n", cfg.ActiveWindow)
				fmt.Printf("  running_window: %s\n", cfg.RunningWindow)
				fmt.Printf("  refresh: %s\n", cfg.RefreshEvery)
				fmt.Printf("  max_sessions: %d\n", cfg.MaxSessions)
				fmt.Printf("  all_scan_window: %s\n", cfg.AllScanWindow)
				fmt.Printf("  statusline_min_write: %s\n", cfg.StatuslineMinWrite)
				return nil
			}
			_ = cmd.Help()
			return nil
		},
	}
	cmd.Flags().BoolVar(&show, "show", true, "Show config (default)")
	cmd.Flags().BoolVar(&init, "init", false, "Write a default config file if missing")
	return cmd
}
