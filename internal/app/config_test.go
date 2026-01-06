package app

import "testing"

func TestCfgFromFlags(t *testing.T) {
	base := defaultConfig()

	cfg, err := cfgFromFlags(base, "CODEx", false, true, "30m", "3s", "2s", 10, true)
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

	if _, err := cfgFromFlags(base, "", false, true, "bad", "3s", "1s", 0, false); err == nil {
		t.Fatalf("expected error for invalid active window")
	}
	if _, err := cfgFromFlags(base, "", false, true, "30m", "bad", "1s", 0, false); err == nil {
		t.Fatalf("expected error for invalid running window")
	}
	if _, err := cfgFromFlags(base, "", false, true, "30m", "3s", "bad", 0, false); err == nil {
		t.Fatalf("expected error for invalid refresh")
	}
}
