package app

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// -------------------------
// Tail
// -------------------------

func newTailCmd() *cobra.Command {
	var (
		provider string
		lines    int
		follow   bool
	)

	cmd := &cobra.Command{
		Use:   "tail <id>",
		Short: "Tail a session transcript/log",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := strings.TrimSpace(args[0])
			if id == "" {
				return fmt.Errorf("missing session id")
			}
			if lines <= 0 {
				lines = 50
			}
			path, _, err := resolveSourcePath(provider, id)
			if err != nil {
				return err
			}

			tailArgs := []string{"-n", strconv.Itoa(lines)}
			if follow {
				tailArgs = append(tailArgs, "-f")
			}
			tailArgs = append(tailArgs, path)

			tail := exec.Command("tail", tailArgs...)
			tail.Stdout = os.Stdout
			tail.Stderr = os.Stderr
			tail.Stdin = os.Stdin
			return tail.Run()
		},
	}

	cmd.Flags().StringVar(&provider, "provider", "", "Filter by provider: claude|codex (optional)")
	cmd.Flags().IntVar(&lines, "lines", 50, "Number of lines to show before following")
	cmd.Flags().BoolVar(&follow, "follow", true, "Follow the file for new lines")
	return cmd
}
