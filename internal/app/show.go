package app

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// -------------------------
// Show
// -------------------------

func newShowCmd() *cobra.Command {
	var (
		provider       string
		jsonOut        bool
		includeLastMsg bool
		redact         bool
	)

	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show details for a single session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := strings.TrimSpace(args[0])
			if id == "" {
				return fmt.Errorf("missing session id")
			}

			rec, err := resolveRecord(provider, id)
			if err != nil {
				return err
			}

			cfg := loadConfig()
			cfg.IncludeEnded = true
			cfg.IncludeLastMsg = includeLastMsg
			cfg.Redact = redact
			view := makeView(rec, time.Now().UTC(), cfg)

			if jsonOut {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(view)
			}

			fmt.Fprint(cmd.OutOrStdout(), view.Detail)
			return nil
		},
	}

	cmd.Flags().StringVar(&provider, "provider", "", "Filter by provider: claude|codex (optional)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output JSON")
	cmd.Flags().BoolVar(&includeLastMsg, "include-last-msg", false, "Include last user/assistant messages when available")
	cmd.Flags().BoolVar(&redact, "redact", true, "Redact paths/IDs in output")
	return cmd
}
