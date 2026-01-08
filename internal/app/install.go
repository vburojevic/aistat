package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// -------------------------
// Install
// -------------------------

func newInstallCmd() *cobra.Command {
	var (
		force       bool
		dryRun      bool
		skipClaude  bool
		skipCodex   bool
		noWizard    bool
		usePath     bool
		cmdOverride string
	)

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Claude Code hooks/statusline + Codex notify to feed aistat",
		Long: `This will:
- Update ~/.claude/settings.json to call aistat from hooks + statusLine
- Update ~/.codex/config.toml to set notify = ["<cmd>", "ingest", "codex-notify"]

Backups are created with a timestamp suffix.

By default, when run in a TTY, this command starts an interactive setup wizard.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			exe, err := os.Executable()
			if err != nil {
				return err
			}
			exe, _ = filepath.Abs(exe)

			// Determine how configs should invoke us.
			callCmd := exe
			if usePath {
				callCmd = "aistat"
			}
			if strings.TrimSpace(cmdOverride) != "" {
				callCmd = strings.TrimSpace(cmdOverride)
			}

			interactive := term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))

			// If we're interactive and the user didn't explicitly opt out or pass flags,
			// run the wizard to gather choices.
			flagsExplicit := cmd.Flags().Changed("force") ||
				cmd.Flags().Changed("dry-run") ||
				cmd.Flags().Changed("skip-claude") ||
				cmd.Flags().Changed("skip-codex") ||
				cmd.Flags().Changed("use-path") ||
				cmd.Flags().Changed("cmd")

			if interactive && !noWizard && !flagsExplicit {
				choices, err := runInstallWizard(exe)
				if err != nil {
					return err
				}
				if choices.Aborted {
					fmt.Println("Aborted.")
					return nil
				}
				force = choices.Force
				dryRun = choices.DryRun
				skipClaude = !choices.InstallClaude
				skipCodex = !choices.InstallCodex
				callCmd = choices.Command
			}

			if skipClaude && skipCodex {
				fmt.Println("Nothing to install (both providers skipped).")
				return nil
			}

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
				return errors.New(strings.Join(errs, "\n"))
			}
			if dryRun {
				fmt.Println("Dry run complete.")
			} else {
				fmt.Println("Install complete. Run `aistat`.")
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing statusLine/notify if present")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show changes without writing files")
	cmd.Flags().BoolVar(&skipClaude, "skip-claude", false, "Skip Claude Code setup")
	cmd.Flags().BoolVar(&skipCodex, "skip-codex", false, "Skip Codex setup")
	cmd.Flags().BoolVar(&noWizard, "no-wizard", false, "Disable interactive setup wizard")
	cmd.Flags().BoolVar(&usePath, "use-path", false, "Use 'aistat' instead of an absolute path in configs (requires PATH)")
	cmd.Flags().StringVar(&cmdOverride, "cmd", "", "Override the command/path written into configs (e.g. /usr/local/bin/aistat)")
	return cmd
}

type installWizardChoices struct {
	InstallClaude bool
	InstallCodex  bool
	Force         bool
	DryRun        bool
	Command       string
	Aborted       bool
}

func runInstallWizard(exeAbs string) (installWizardChoices, error) {
	var (
		integrations []string
		cmdMode      = "abs"
		customCmd    string
		force        bool
		dryRun       bool
		apply        = true
	)

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("What should aistat integrate with?").
				Description("Space to toggle. Enter to continue.").
				Options(
					huh.NewOption("Claude Code (hooks + status line)", "claude").Selected(true),
					huh.NewOption("Codex (notify integration)", "codex").Selected(true),
				).
				Value(&integrations).
				Validate(func(v []string) error {
					if len(v) == 0 {
						return errors.New("select at least one integration")
					}
					return nil
				}),

			huh.NewSelect[string]().
				Title("How should configs invoke aistat?").
				Description(fmt.Sprintf("Absolute path detected: %s", exeAbs)).
				Options(
					huh.NewOption("Absolute path (recommended)", "abs"),
					huh.NewOption("Just `aistat` (relies on PATH)", "path"),
					huh.NewOption("Custom command/path", "custom"),
				).
				Value(&cmdMode),

			huh.NewInput().
				Title("Custom command/path").
				Description("Only used if you selected \"Custom\" above.").
				Placeholder(exeAbs).
				Value(&customCmd).
				Validate(func(s string) error {
					if cmdMode == "custom" && strings.TrimSpace(s) == "" {
						return errors.New("please enter a command/path")
					}
					return nil
				}),

			huh.NewConfirm().
				Title("Overwrite existing entries?").
				Description("Replaces existing statusLine/notify/hook commands if present.").
				Value(&force),

			huh.NewConfirm().
				Title("Dry run (preview only)?").
				Description("Show what would change without writing files.").
				Value(&dryRun),
		),
		huh.NewGroup(
			huh.NewConfirm().
				Title("Apply these changes now?").
				Affirmative("Apply").
				Negative("Cancel").
				Value(&apply),
		),
	)

	// Nice default theme.
	form.WithTheme(huh.ThemeDracula())

	// Enable accessible mode when requested.
	if os.Getenv("ACCESSIBLE") != "" {
		form.WithAccessible(true)
	}

	if err := form.Run(); err != nil {
		return installWizardChoices{}, err
	}

	if !apply {
		return installWizardChoices{Aborted: true}, nil
	}

	choices := installWizardChoices{
		InstallClaude: sliceContains(integrations, "claude"),
		InstallCodex:  sliceContains(integrations, "codex"),
		Force:         force,
		DryRun:        dryRun,
	}

	switch cmdMode {
	case "abs":
		choices.Command = exeAbs
	case "path":
		choices.Command = "aistat"
	case "custom":
		choices.Command = strings.TrimSpace(customCmd)
	default:
		choices.Command = exeAbs
	}

	return choices, nil
}

func sliceContains(ss []string, v string) bool {
	for _, s := range ss {
		if s == v {
			return true
		}
	}
	return false
}

func shellEscape(s string) string {
	// Minimal shell escaping for paths with spaces.
	if s == "" {
		return s
	}
	if strings.ContainsAny(s, " \t\n\"'\\") {
		// Wrap in single quotes, escape existing single quotes.
		return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
	}
	return s
}

func writeWrapper(path string, content string, dryRun bool) error {
	if dryRun {
		fmt.Printf("Would write %s:\n%s\n", path, content)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(content), 0o700); err != nil {
		return err
	}
	return nil
}

func installClaude(exe string, force bool, dryRun bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	ad, err := appDir()
	if err != nil {
		return err
	}
	wrapperDir := filepath.Join(ad, "bin")
	hookWrapper := filepath.Join(wrapperDir, "aistat-claude-hook")
	statusWrapper := filepath.Join(wrapperDir, "aistat-claude-statusline")
	hookScript := fmt.Sprintf(`#!/bin/sh
# Generated by aistat. Safe Claude hook wrapper.
[ -t 0 ] && exit 0
export TERM=dumb
export NO_COLOR=1
exec 1>/dev/null 2>/dev/null
exec %s ingest claude-hook
`, shellEscape(exe))
	statusScript := fmt.Sprintf(`#!/bin/sh
# Generated by aistat. Safe Claude statusline wrapper.
if [ -t 0 ]; then
  printf "\n"
  exit 0
fi
export TERM=dumb
export NO_COLOR=1
exec 2>/dev/null
exec %s statusline
`, shellEscape(exe))
	if err := writeWrapper(hookWrapper, hookScript, dryRun); err != nil {
		return err
	}
	if err := writeWrapper(statusWrapper, statusScript, dryRun); err != nil {
		return err
	}
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o700); err != nil {
		return err
	}

	var settings map[string]any
	if b, err := os.ReadFile(settingsPath); err == nil {
		if err := json.Unmarshal(b, &settings); err != nil {
			return fmt.Errorf("failed to parse %s: %w", settingsPath, err)
		}
	} else {
		settings = map[string]any{}
	}

	hooks, _ := settings["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
	}

	hookCmd := shellEscape(hookWrapper)

	ensureHook := func(event string, matcher any) {
		arr, _ := hooks[event].([]any)
		// de-dupe
		for _, item := range arr {
			mm, _ := item.(map[string]any)
			if mm == nil {
				continue
			}
			hs, _ := mm["hooks"].([]any)
			for _, h := range hs {
				hm, _ := h.(map[string]any)
				if hm == nil {
					continue
				}
				if asString(hm["command"]) == hookCmd {
					return
				}
			}
		}
		newItem := map[string]any{
			"matcher": matcher,
			"hooks": []any{
				map[string]any{
					"type":    "command",
					"command": hookCmd,
				},
			},
		}
		arr = append(arr, newItem)
		hooks[event] = arr
	}

	// Core events
	ensureHook("SessionStart", "*")
	ensureHook("SessionEnd", "*")
	ensureHook("UserPromptSubmit", "*")
	ensureHook("PreToolUse", "*")
	ensureHook("PostToolUse", "*")
	ensureHook("Stop", "*")
	// Status transitions
	ensureHook("Notification", "permission_prompt")
	ensureHook("Notification", "idle_prompt")

	settings["hooks"] = hooks

	// statusLine setup
	if _, exists := settings["statusLine"]; !exists || force {
		settings["statusLine"] = map[string]any{
			"type":    "command",
			"command": shellEscape(statusWrapper),
			"padding": 0,
		}
	}

	b, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	if dryRun {
		fmt.Printf("Would write %s:\n%s\n", settingsPath, string(b))
		return nil
	}

	if _, err := os.Stat(settingsPath); err == nil {
		backup := settingsPath + ".bak." + time.Now().UTC().Format("20060102T150405Z")
		if err := copyFile(settingsPath, backup); err != nil {
			return err
		}
		fmt.Printf("Backed up %s -> %s\n", settingsPath, backup)
	}

	if err := os.WriteFile(settingsPath, b, 0o600); err != nil {
		return err
	}
	fmt.Printf("Updated %s\n", settingsPath)
	return nil
}

func installCodex(exe string, force bool, dryRun bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	ad, err := appDir()
	if err != nil {
		return err
	}
	wrapperDir := filepath.Join(ad, "bin")
	notifyWrapper := filepath.Join(wrapperDir, "aistat-codex-notify")
	notifyScript := fmt.Sprintf(`#!/bin/sh
# Generated by aistat. Safe Codex notify wrapper.
[ -t 0 ] && exit 0
export TERM=dumb
export NO_COLOR=1
exec 1>/dev/null 2>/dev/null
exec %s ingest codex-notify
`, shellEscape(exe))
	if err := writeWrapper(notifyWrapper, notifyScript, dryRun); err != nil {
		return err
	}
	cfgPath := filepath.Join(home, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o700); err != nil {
		return err
	}

	notifyLine := fmt.Sprintf(`notify = [%q]`, notifyWrapper)

	var content string
	if b, err := os.ReadFile(cfgPath); err == nil {
		content = string(b)
	} else {
		content = ""
	}

	if strings.Contains(content, "notify =") {
		if !force {
			if strings.Contains(content, "codex-notify") && strings.Contains(content, exe) {
				fmt.Printf("Codex notify already configured in %s\n", cfgPath)
				return nil
			}
			return fmt.Errorf("notify already set in %s (use --force to overwrite)", cfgPath)
		}
		re := regexp.MustCompile(`(?m)^notify\s*=.*$`)
		content = re.ReplaceAllString(content, notifyLine)
	} else {
		lines := strings.Split(content, "\n")
		insertAt := len(lines)
		for i, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "[") {
				insertAt = i
				break
			}
		}
		var out []string
		out = append(out, lines[:insertAt]...)
		out = append(out, notifyLine)
		out = append(out, lines[insertAt:]...)
		content = strings.Join(out, "\n")
	}

	if dryRun {
		fmt.Printf("Would write %s:\n%s\n", cfgPath, content)
		return nil
	}

	if _, err := os.Stat(cfgPath); err == nil {
		backup := cfgPath + ".bak." + time.Now().UTC().Format("20060102T150405Z")
		if err := copyFile(cfgPath, backup); err != nil {
			return err
		}
		fmt.Printf("Backed up %s -> %s\n", cfgPath, backup)
	}

	if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
		return err
	}
	fmt.Printf("Updated %s\n", cfgPath)
	return nil
}
