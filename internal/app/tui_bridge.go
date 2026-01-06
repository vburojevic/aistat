package app

import (
	"github.com/vburojevic/aistat/internal/app/tui"
	"github.com/vburojevic/aistat/internal/app/tui/state"
)

// runTUINew runs the new redesigned TUI
func runTUINew(cfg Config) error {
	tuiCfg := tui.Config{
		Redact:         cfg.Redact,
		ActiveWindow:   cfg.ActiveWindow,
		RunningWindow:  cfg.RunningWindow,
		RefreshEvery:   cfg.RefreshEvery,
		MaxSessions:    cfg.MaxSessions,
		IncludeEnded:   cfg.IncludeEnded,
		ProviderFilter: cfg.ProviderFilter,
		ProjectFilters: cfg.ProjectFilters,
		StatusFilters:  convertStatusFilters(cfg.StatusFilters),
		SortBy:         cfg.SortBy,
		GroupBy:        cfg.GroupBy,
		IncludeLastMsg: cfg.IncludeLastMsg,
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

// convertStatusFilters converts app.Status slice to state.Status slice
func convertStatusFilters(statuses []Status) []state.Status {
	result := make([]state.Status, len(statuses))
	for i, s := range statuses {
		result[i] = state.Status(s)
	}
	return result
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
