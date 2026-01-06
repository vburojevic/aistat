package app

import (
	"testing"
	"time"
)

func TestProjectFilter(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AISTAT_HOME", root)
	t.Setenv("HOME", root)
	t.Setenv("CODEX_HOME", root)

	now := time.Now().UTC()
	if err := updateRecord(ProviderClaude, "a", func(rec *SessionRecord) {
		rec.ProjectDir = "/tmp/Alpha"
		rec.CWD = "/tmp/Alpha"
		rec.LastSeen = now
	}); err != nil {
		t.Fatalf("updateRecord: %v", err)
	}
	if err := updateRecord(ProviderClaude, "b", func(rec *SessionRecord) {
		rec.ProjectDir = "/tmp/Beta"
		rec.CWD = "/tmp/Beta"
		rec.LastSeen = now
	}); err != nil {
		t.Fatalf("updateRecord: %v", err)
	}

	cfg := defaultConfig()
	cfg.ProjectFilters = []string{"alpha"}
	cfg.ActiveWindow = time.Hour
	cfg.IncludeEnded = true

	views, err := gatherSessions(cfg)
	if err != nil {
		t.Fatalf("gatherSessions: %v", err)
	}
	if len(views) != 1 {
		t.Fatalf("expected 1 view, got %d", len(views))
	}
	if views[0].Project != "Alpha" {
		t.Fatalf("expected project Alpha, got %q", views[0].Project)
	}
}

func TestSortSessionsByCost(t *testing.T) {
	views := []SessionView{
		{ID: "a", Cost: 1.0, LastSeen: time.Now().Add(-time.Hour)},
		{ID: "b", Cost: 3.0, LastSeen: time.Now()},
		{ID: "c", Cost: 2.0, LastSeen: time.Now().Add(-2 * time.Hour)},
	}
	sortSessions(views, "cost")
	if views[0].ID != "b" || views[1].ID != "c" || views[2].ID != "a" {
		t.Fatalf("unexpected order: %q, %q, %q", views[0].ID, views[1].ID, views[2].ID)
	}
}

func TestGroupSessionsByProvider(t *testing.T) {
	views := []SessionView{
		{Provider: ProviderCodex, ID: "a"},
		{Provider: ProviderClaude, ID: "b"},
		{Provider: ProviderCodex, ID: "c"},
	}
	groups := groupSessions(views, "provider")
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	if groups[0].Group != "codex" {
		t.Fatalf("expected first group codex, got %q", groups[0].Group)
	}
	if len(groups[0].Sessions) != 2 {
		t.Fatalf("expected 2 codex sessions, got %d", len(groups[0].Sessions))
	}
}

func TestGroupSessionsByDayHour(t *testing.T) {
	ts := time.Date(2026, 1, 6, 12, 34, 0, 0, time.UTC)
	views := []SessionView{
		{Provider: ProviderCodex, ID: "a", LastSeen: ts},
		{Provider: ProviderCodex, ID: "b", LastSeen: ts.Add(30 * time.Minute)},
	}
	dayGroups := groupSessions(views, "day")
	if len(dayGroups) != 1 {
		t.Fatalf("expected 1 day group, got %d", len(dayGroups))
	}
	hourGroups := groupSessions(views, "hour")
	if len(hourGroups) != 2 {
		t.Fatalf("expected 2 hour groups, got %d", len(hourGroups))
	}
}
