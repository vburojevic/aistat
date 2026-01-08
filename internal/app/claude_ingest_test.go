package app

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestIngestClaudeStatusline(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AISTAT_HOME", root)

	input := ClaudeStatuslineInput{
		SessionID:      "sess-1",
		TranscriptPath: "/tmp/claude.jsonl",
		CWD:            "/tmp",
	}
	input.Model.ID = "claude-3"
	input.Model.DisplayName = "Claude 3"
	input.Workspace.CurrentDir = "/tmp/dir"
	input.Workspace.ProjectDir = "/tmp/project"
	input.Cost.TotalCostUSD = 1.234
	input.ContextWindow.ContextWindowSize = 100
	input.ContextWindow.CurrentUsage = &struct {
		InputTokens              int `json:"input_tokens"`
		OutputTokens             int `json:"output_tokens"`
		CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
		CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	}{
		InputTokens:  10,
		OutputTokens: 5,
	}

	b, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	line, err := ingestClaudeStatusline(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("ingestClaudeStatusline error: %v", err)
	}
	if line == "" {
		t.Fatalf("expected non-empty statusline")
	}
	if !bytes.Contains([]byte(line), []byte("Claude 3")) {
		t.Fatalf("expected model in statusline, got: %q", line)
	}

	if err := drainClaudeSpool(); err != nil {
		t.Fatalf("drainClaudeSpool error: %v", err)
	}

	p, err := recordPath(ProviderClaude, "sess-1")
	if err != nil {
		t.Fatalf("recordPath error: %v", err)
	}
	rec, err := loadRecord(p)
	if err != nil {
		t.Fatalf("loadRecord error: %v", err)
	}
	if rec.ModelDisplay != "Claude 3" {
		t.Fatalf("unexpected model display: %q", rec.ModelDisplay)
	}
	if rec.ProjectDir != "/tmp/project" {
		t.Fatalf("unexpected project dir: %q", rec.ProjectDir)
	}
	if rec.CostUSD != 1.234 {
		t.Fatalf("unexpected cost: %v", rec.CostUSD)
	}
}
