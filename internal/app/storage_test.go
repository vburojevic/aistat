package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRecordLifecycle(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AISTAT_HOME", root)

	id := "abc/def"
	if err := updateRecord(ProviderClaude, id, func(rec *SessionRecord) {
		rec.CWD = "/tmp/project"
		rec.Status = StatusRunning
	}); err != nil {
		t.Fatalf("updateRecord error: %v", err)
	}

	sd, err := sessionsDir()
	if err != nil {
		t.Fatalf("sessionsDir error: %v", err)
	}
	entries, err := os.ReadDir(sd)
	if err != nil {
		t.Fatalf("ReadDir error: %v", err)
	}
	jsonCount := 0
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			jsonCount++
		}
	}
	if jsonCount != 1 {
		t.Fatalf("expected 1 record file, got %d", jsonCount)
	}

	p, err := recordPath(ProviderClaude, id)
	if err != nil {
		t.Fatalf("recordPath error: %v", err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("record file missing: %v", err)
	}

	rec, err := loadRecord(p)
	if err != nil {
		t.Fatalf("loadRecord error: %v", err)
	}
	if rec.ID != id {
		t.Fatalf("expected id %q, got %q", id, rec.ID)
	}
	if rec.CWD != "/tmp/project" {
		t.Fatalf("unexpected cwd: %q", rec.CWD)
	}
}

func TestConfigFilePathRespectsAISTATHome(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AISTAT_HOME", root)

	p, err := configFilePath()
	if err != nil {
		t.Fatalf("configFilePath error: %v", err)
	}
	expected := filepath.Join(root, "config.json")
	if p != expected {
		t.Fatalf("expected %q, got %q", expected, p)
	}
}
