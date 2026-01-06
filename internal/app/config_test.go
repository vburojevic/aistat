package app

import "testing"

func TestCfgFromFlags(t *testing.T) {
	base := defaultConfig()

	cfg, err := cfgFromFlags(base, "CODEx", false, true, "30m", "3s", "2s", 10, true, []string{"proj1"}, "cost", "provider", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ProviderFilter != "codex" {
		t.Fatalf("expected provider filter codex, got %q", cfg.ProviderFilter)
	}
	if cfg.MaxSessions != 10 {
		t.Fatalf("expected max 10, got %d", cfg.MaxSessions)
	}
	if !cfg.NoColor {
		t.Fatalf("expected no color true")
	}
	if cfg.SortBy != "cost" {
		t.Fatalf("expected sort cost, got %q", cfg.SortBy)
	}
	if cfg.GroupBy != "provider" {
		t.Fatalf("expected group-by provider, got %q", cfg.GroupBy)
	}
	if len(cfg.ProjectFilters) != 1 || cfg.ProjectFilters[0] != "proj1" {
		t.Fatalf("unexpected project filters: %+v", cfg.ProjectFilters)
	}
	if !cfg.IncludeLastMsg {
		t.Fatalf("expected include-last-msg true")
	}

	if _, err := cfgFromFlags(base, "", false, true, "bad", "3s", "1s", 0, false, nil, "last_seen", "", false); err == nil {
		t.Fatalf("expected error for invalid active window")
	}
	if _, err := cfgFromFlags(base, "", false, true, "30m", "bad", "1s", 0, false, nil, "last_seen", "", false); err == nil {
		t.Fatalf("expected error for invalid running window")
	}
	if _, err := cfgFromFlags(base, "", false, true, "30m", "3s", "bad", 0, false, nil, "last_seen", "", false); err == nil {
		t.Fatalf("expected error for invalid refresh")
	}
	if _, err := cfgFromFlags(base, "", false, true, "30m", "3s", "1s", 0, false, nil, "nope", "", false); err == nil {
		t.Fatalf("expected error for invalid sort")
	}
	if _, err := cfgFromFlags(base, "", false, true, "30m", "3s", "1s", 0, false, nil, "last_seen", "nope", false); err == nil {
		t.Fatalf("expected error for invalid group-by")
	}
}
