package app

import (
	"fmt"
	"os"

	"github.com/vburojevic/aistat/internal/app/tui"
	"github.com/vburojevic/aistat/internal/app/tui/state"
)

// runTUINew runs the new redesigned TUI
func runTUINew(cfg Config) (err error) {
	// Ensure terminal state is restored on panic/crash
	defer func() {
		if r := recover(); r != nil {
			// Reset terminal to sane state
			fmt.Fprint(os.Stdout, "\033[?1049l") // Exit alt-screen
			fmt.Fprint(os.Stdout, "\033[?25h")   // Show cursor
			fmt.Fprint(os.Stdout, "\033[0m")     // Reset colors
			panic(r)                             // Re-panic after cleanup
		}
	}()
	tuiCfg := tui.Config{
		RefreshEvery: cfg.RefreshEvery,
		MaxSessions:  cfg.MaxSessions,
		ShowEnded:    cfg.IncludeEnded,
	}

	fetcher := func() ([]state.SessionView, error) {
		views, err := gatherSessions(cfg)
		if err != nil {
			return nil, err
		}
		return convertSessionViews(views), nil
	}

	return tui.Run(tuiCfg, fetcher)
}

// convertSessionViews converts app.SessionView slice to state.SessionView slice
func convertSessionViews(views []SessionView) []state.SessionView {
	result := make([]state.SessionView, len(views))
	for i, v := range views {
		result[i] = state.SessionView{
			Provider:   state.Provider(v.Provider),
			ID:         v.ID,
			Status:     state.Status(v.Status),
			Reason:     v.Reason,
			Project:    v.Project,
			Dir:        v.Dir,
			Branch:     v.Branch,
			Model:      v.Model,
			Cost:       v.Cost,
			Age:        v.Age,
			LastSeen:   v.LastSeen,
			SourcePath: v.SourcePath,
			Detail:     v.Detail,
			LastUser:   v.LastUser,
			LastAssist: v.LastAssist,
		}
	}
	return result
}
