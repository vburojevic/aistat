package app

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// -------------------------
// Doctor
// -------------------------

func newDoctorCmd() *cobra.Command {
	var (
		fix         bool
		force       bool
		dryRun      bool
		skipClaude  bool
		skipCodex   bool
		usePath     bool
		cmdOverride string
	)
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check setup + show where we read sessions from",
		RunE: func(cmd *cobra.Command, args []string) error {
			if fix {
				if term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd())) && !force && !dryRun {
					if !confirm("Run doctor --fix? This will edit Claude/Codex configs. [y/N]: ") {
						fmt.Println("Aborted.")
						return nil
					}
				}

				exe, err := os.Executable()
				if err != nil {
					return err
				}
				exe, _ = filepath.Abs(exe)

				callCmd := exe
				if usePath {
					callCmd = "aistat"
				}
				if strings.TrimSpace(cmdOverride) != "" {
					callCmd = strings.TrimSpace(cmdOverride)
				}

				home, _ := os.UserHomeDir()
				claudeSettings := filepath.Join(home, ".claude", "settings.json")
				codexCfg := filepath.Join(home, ".codex", "config.toml")

				claudeSnap := ""
				codexSnap := ""
				if !dryRun {
					if !skipClaude {
						if snap, err := snapshotFile(claudeSettings); err == nil {
							claudeSnap = snap
						}
					}
					if !skipCodex {
						if snap, err := snapshotFile(codexCfg); err == nil {
							codexSnap = snap
						}
					}
				}

				if skipClaude && skipCodex {
					fmt.Println("Nothing to install (both providers skipped).")
				} else {
					var errs []string
					if !skipClaude {
						if err := installClaude(callCmd, force, dryRun); err != nil {
							errs = append(errs, "Claude: "+err.Error())
						}
					}
					if !skipCodex {
						if err := installCodex(callCmd, force, dryRun); err != nil {
							errs = append(errs, "Codex: "+err.Error())
						}
					}
					if len(errs) > 0 {
						if !dryRun {
							_ = restoreSnapshot(claudeSnap, claudeSettings)
							_ = restoreSnapshot(codexSnap, codexCfg)
						}
						return errors.New(strings.Join(errs, "\n"))
					}
					if dryRun {
						fmt.Println("Dry run complete.")
					} else {
						_ = cleanupSnapshot(claudeSnap)
						_ = cleanupSnapshot(codexSnap)
					}
				}
			}

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
	cmd.Flags().BoolVar(&fix, "fix", false, "Attempt to auto-fix setup (runs install)")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing statusLine/notify if present")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show changes without writing files")
	cmd.Flags().BoolVar(&skipClaude, "skip-claude", false, "Skip Claude Code setup")
	cmd.Flags().BoolVar(&skipCodex, "skip-codex", false, "Skip Codex setup")
	cmd.Flags().BoolVar(&usePath, "use-path", false, "Use 'aistat' instead of an absolute path in configs (requires PATH)")
	cmd.Flags().StringVar(&cmdOverride, "cmd", "", "Override the command/path written into configs (e.g. /usr/local/bin/aistat)")
	return cmd
}

func confirm(prompt string) bool {
	fmt.Fprint(os.Stdout, prompt)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes"
}

func snapshotFile(path string) (string, error) {
	if _, err := os.Stat(path); err != nil {
		return "", nil
	}
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, filepath.Base(path)+".aistat.bak.*")
	if err != nil {
		return "", err
	}
	_ = f.Close()
	if err := copyFile(path, f.Name()); err != nil {
		return "", err
	}
	return f.Name(), nil
}

func restoreSnapshot(snapshotPath, destPath string) error {
	if strings.TrimSpace(snapshotPath) == "" {
		return nil
	}
	if _, err := os.Stat(snapshotPath); err != nil {
		return nil
	}
	return copyFile(snapshotPath, destPath)
}

func cleanupSnapshot(snapshotPath string) error {
	if strings.TrimSpace(snapshotPath) == "" {
		return nil
	}
	return os.Remove(snapshotPath)
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
