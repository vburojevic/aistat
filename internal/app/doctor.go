package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// -------------------------
// Doctor
// -------------------------

func newDoctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check setup + show where we read sessions from",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadConfig()
			ad, _ := appDir()
			sd, _ := sessionsDir()
			cp, _ := configFilePath()

			fmt.Printf("aistat\n")
			fmt.Printf("  app dir: %s\n", ad)
			fmt.Printf("  sessions dir: %s\n", sd)
			fmt.Printf("  config: %s\n", cp)
			fmt.Printf("  redact: %v\n", cfg.Redact)
			fmt.Printf("  active window: %s\n", cfg.ActiveWindow)
			fmt.Printf("  refresh: %s\n", cfg.RefreshEvery)
			fmt.Println()

			home, _ := os.UserHomeDir()
			claudeSettings := filepath.Join(home, ".claude", "settings.json")
			fmt.Printf("Claude Code\n")
			fmt.Printf("  settings: %s (%s)\n", claudeSettings, existsStr(claudeSettings))
			claudeProjects := filepath.Join(home, ".claude", "projects")
			fmt.Printf("  projects: %s (%s)\n", claudeProjects, existsStr(claudeProjects))
			fmt.Println()

			codexCfg := filepath.Join(home, ".codex", "config.toml")
			fmt.Printf("Codex\n")
			fmt.Printf("  config: %s (%s)\n", codexCfg, existsStr(codexCfg))
			codexSessions := filepath.Join(home, ".codex", "sessions")
			fmt.Printf("  sessions: %s (%s)\n", codexSessions, existsStr(codexSessions))
			fmt.Println()

			fmt.Println("Tip: run `aistat install` to wire up hooks/statusline/notify.")
			return nil
		},
	}
	return cmd
}

func existsStr(path string) string {
	if _, err := os.Stat(path); err == nil {
		return "ok"
	}
	return "missing"
}

func copyFile(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o600)
}
