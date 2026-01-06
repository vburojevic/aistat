package app

import (
	"testing"
	"time"
)

func TestStatusFilter(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AISTAT_HOME", root)
	t.Setenv("HOME", root)
	t.Setenv("CODEX_HOME", root)
	now := time.Now().UTC()
	old := now.Add(-time.Minute)

	if err := updateRecord(ProviderClaude, "run", func(rec *SessionRecord) {
		rec.Status = StatusRunning
		rec.LastSeen = old
	}); err != nil {
		t.Fatalf("updateRecord: %v", err)
	}
	if err := updateRecord(ProviderClaude, "wait", func(rec *SessionRecord) {
		rec.Status = StatusWaiting
		rec.LastSeen = old
	}); err != nil {
		t.Fatalf("updateRecord: %v", err)
	}

	cfg := defaultConfig()
	cfg.IncludeEnded = true
	cfg.ActiveWindow = time.Hour
	cfg.StatusFilters = []Status{StatusRunning}

	views, err := gatherSessions(cfg)
	if err != nil {
		t.Fatalf("gatherSessions: %v", err)
	}
	if len(views) != 1 {
		t.Fatalf("expected 1 view, got %d", len(views))
	}
	if views[0].Status != StatusRunning {
		t.Fatalf("expected running status, got %q", views[0].Status)
	}
}
