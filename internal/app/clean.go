package app

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// -------------------------
// Clean
// -------------------------

func newCleanCmd() *cobra.Command {
	var (
		dryRun        bool
		cleanSpool    bool
		cleanSessions bool
	)

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean spool data and invalid session records",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cleanSpool && !cleanSessions {
				return errors.New("nothing to clean (enable --spool and/or --sessions)")
			}
			var parts []string
			if cleanSpool {
				n, err := cleanSpoolData(dryRun)
				if err != nil {
					return err
				}
				parts = append(parts, fmt.Sprintf("spool:%d", n))
			}
			if cleanSessions {
				n, err := cleanInvalidSessions(dryRun)
				if err != nil {
					return err
				}
				parts = append(parts, fmt.Sprintf("invalid_sessions:%d", n))
			}
			prefix := "Cleaned"
			if dryRun {
				prefix = "Would clean"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", prefix, strings.Join(parts, " "))
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be removed without deleting")
	cmd.Flags().BoolVar(&cleanSpool, "spool", true, "Clean spool files")
	cmd.Flags().BoolVar(&cleanSessions, "sessions", true, "Clean invalid session records")
	return cmd
}

func cleanSpoolData(dryRun bool) (int, error) {
	sd, err := spoolDir()
	if err != nil {
		return 0, err
	}
	var files []string
	_ = filepath.WalkDir(sd, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if len(files) == 0 {
		return 0, nil
	}
	if dryRun {
		return len(files), nil
	}
	if err := os.RemoveAll(sd); err != nil {
		return 0, err
	}
	return len(files), nil
}

func cleanInvalidSessions(dryRun bool) (int, error) {
	sd, err := sessionsDir()
	if err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(sd)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}
	removed := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		p := filepath.Join(sd, e.Name())
		rec, err := loadRecord(p)
		if err != nil {
			continue
		}
		if !validSessionID(rec.ID) || strings.TrimSpace(string(rec.Provider)) == "" {
			if !dryRun {
				_ = os.Remove(p)
			}
			removed++
		}
	}
	return removed, nil
}
