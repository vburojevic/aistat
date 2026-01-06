package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScanCodexHeaderAndTail(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CODEX_HOME", root)

	sessionsDir := filepath.Join(root, "sessions")
	if err := os.MkdirAll(sessionsDir, 0o700); err != nil {
		t.Fatalf("mkdir sessions: %v", err)
	}

	fp := filepath.Join(sessionsDir, "rollout-abc.jsonl")

	lines := []map[string]any{
		{
			"timestamp": "2024-01-02T03:04:05Z",
			"type":      "session_meta",
			"payload": map[string]any{
				"id":        "session-1",
				"cwd":       "/tmp/proj",
				"timestamp": "2024-01-02T03:04:05Z",
			},
		},
		{
			"timestamp": "2024-01-02T03:04:06Z",
			"type":      "turn_context",
			"payload": map[string]any{
				"cwd":             "/tmp/proj",
				"model":           "gpt-4",
				"approval_policy": "never",
			},
		},
		{
			"timestamp": "2024-01-02T03:04:10Z",
			"type":      "response_item",
			"payload": map[string]any{
				"type": "message",
				"role": "user",
				"content": []any{
					map[string]any{"type": "input_text", "text": "hello"},
				},
			},
		},
		{
			"timestamp": "2024-01-02T03:04:11Z",
			"type":      "response_item",
			"payload": map[string]any{
				"type": "message",
				"role": "assistant",
				"content": []any{
					map[string]any{"type": "text", "text": "hi"},
				},
			},
		},
		{
			"timestamp": "2024-01-02T03:05:00Z",
			"type":      "some_event",
			"payload": map[string]any{
				"type": "approval_prompt",
			},
		},
	}

	f, err := os.Create(fp)
	if err != nil {
		t.Fatalf("create file: %v", err)
	}
	enc := json.NewEncoder(f)
	for _, line := range lines {
		if err := enc.Encode(line); err != nil {
			_ = f.Close()
			t.Fatalf("encode line: %v", err)
		}
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close file: %v", err)
	}

	now := time.Date(2024, 1, 2, 3, 6, 0, 0, time.UTC)
	if err := os.Chtimes(fp, now, now); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	hdr, err := scanCodexHeader(fp, 100)
	if err != nil {
		t.Fatalf("scanCodexHeader error: %v", err)
	}
	if hdr.SessionID != "session-1" {
		t.Fatalf("unexpected session id: %q", hdr.SessionID)
	}
	if hdr.CWD != "/tmp/proj" {
		t.Fatalf("unexpected cwd: %q", hdr.CWD)
	}
	if hdr.Model != "gpt-4" {
		t.Fatalf("unexpected model: %q", hdr.Model)
	}
	if hdr.ApprovalPolicy != "never" {
		t.Fatalf("unexpected approval policy: %q", hdr.ApprovalPolicy)
	}

	tail, err := scanCodexTail(fp, 64*1024)
	if err != nil {
		t.Fatalf("scanCodexTail error: %v", err)
	}
	if tail.LastPayloadType != "approval_prompt" {
		t.Fatalf("unexpected last payload type: %q", tail.LastPayloadType)
	}
	if tail.LastUserText != "hello" {
		t.Fatalf("unexpected last user text: %q", tail.LastUserText)
	}
	if tail.LastAssistantText != "hi" {
		t.Fatalf("unexpected last assistant text: %q", tail.LastAssistantText)
	}

	cfg := defaultConfig()
	cfg.ActiveWindow = 2 * time.Hour
	cfg.HeaderScanLines = 100
	cfg.TailBytesCodex = 64 * 1024
	recs, err := scanCodexRollouts(cfg, now)
	if err != nil {
		t.Fatalf("scanCodexRollouts error: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("expected 1 record, got %d", len(recs))
	}
	if recs[0].Status != StatusApproval {
		t.Fatalf("expected approval status, got %q", recs[0].Status)
	}
}
