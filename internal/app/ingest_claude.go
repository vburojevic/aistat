package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

// -------------------------
// Claude ingestion
// -------------------------

func ingestClaudeHook(r io.Reader) error {
	if f, ok := r.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		// Avoid reading from a TTY; hook input should be piped JSON.
		return nil
	}
	var m map[string]any
	dec := json.NewDecoder(io.LimitReader(r, 10*1024*1024))
	if err := dec.Decode(&m); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}

	event := strings.TrimSpace(getString(m, "hook_event_name"))
	sid := normalizePlaceholder(getString(m, "session_id"))
	if sid == "" {
		// No session => ignore (but return nil to not break hooks)
		return nil
	}

	now := time.Now().UTC()
	tp := normalizePlaceholder(getString(m, "transcript_path"))
	cwd := normalizePlaceholder(getString(m, "cwd"))

	notifType := normalizePlaceholder(getString(m, "notification_type"))
	notifMsg := getString(m, "message")

	patch := ClaudeHookPatch{
		SessionID:      sid,
		At:             now.Format(time.RFC3339Nano),
		TranscriptPath: tp,
		CWD:            cwd,
		LastEventName:  event,
	}

	switch event {
	case "SessionStart":
		patch.Status = StatusRunning
		patch.StatusReason = "session started"
	case "UserPromptSubmit":
		patch.Status = StatusRunning
		patch.StatusReason = "user prompt submitted"
	case "PreToolUse", "PostToolUse":
		patch.Status = StatusRunning
		patch.StatusReason = "tool activity"
	case "Stop":
		patch.Status = StatusWaiting
		patch.StatusReason = "awaiting input"
	case "Notification":
		patch.LastNotificationType = notifType
		patch.LastNotificationMsg = notifMsg
		switch notifType {
		case "permission_prompt":
			patch.Status = StatusApproval
			patch.StatusReason = "awaiting approval"
		case "idle_prompt":
			patch.Status = StatusWaiting
			patch.StatusReason = "awaiting input"
		default:
			patch.Status = StatusWaiting
			if notifType != "" {
				patch.StatusReason = "notification: " + notifType
			} else {
				patch.StatusReason = "notification"
			}
		}
	case "SessionEnd":
		patch.Status = StatusEnded
		patch.StatusReason = "session ended"
		patch.EndedAt = now.Format(time.RFC3339Nano)
	default:
		// keep as-is
	}

	if b, err := json.Marshal(patch); err == nil {
		_ = writeSpoolBytes(ProviderClaude, "hook", sid, b, true)
	}
	return nil
}

type ClaudeStatuslineInput struct {
	HookEventName  string `json:"hook_event_name"`
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	CWD            string `json:"cwd"`
	Model          struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
	} `json:"model"`
	Workspace struct {
		CurrentDir string `json:"current_dir"`
		ProjectDir string `json:"project_dir"`
	} `json:"workspace"`
	Version string `json:"version"`
	Cost    struct {
		TotalCostUSD       float64 `json:"total_cost_usd"`
		TotalDurationMS    int64   `json:"total_duration_ms"`
		TotalAPIDurationMS int64   `json:"total_api_duration_ms"`
		TotalLinesAdded    int     `json:"total_lines_added"`
		TotalLinesRemoved  int     `json:"total_lines_removed"`
	} `json:"cost"`
	ContextWindow struct {
		TotalInputTokens  int `json:"total_input_tokens"`
		TotalOutputTokens int `json:"total_output_tokens"`
		ContextWindowSize int `json:"context_window_size"`
		CurrentUsage      *struct {
			InputTokens              int `json:"input_tokens"`
			OutputTokens             int `json:"output_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
			CacheReadInputTokens     int `json:"cache_read_input_tokens"`
		} `json:"current_usage"`
	} `json:"context_window"`
}

type ClaudeHookPatch struct {
	SessionID            string `json:"session_id"`
	At                   string `json:"at"`
	TranscriptPath       string `json:"transcript_path,omitempty"`
	CWD                  string `json:"cwd,omitempty"`
	LastEventName        string `json:"last_event_name,omitempty"`
	Status               Status `json:"status,omitempty"`
	StatusReason         string `json:"status_reason,omitempty"`
	EndedAt              string `json:"ended_at,omitempty"`
	LastNotificationType string `json:"last_notification_type,omitempty"`
	LastNotificationMsg  string `json:"last_notification_msg,omitempty"`
}

type ClaudeStatuslinePatch struct {
	SessionID                string  `json:"session_id"`
	At                       string  `json:"at"`
	TranscriptPath           string  `json:"transcript_path,omitempty"`
	CWD                      string  `json:"cwd,omitempty"`
	ProjectDir               string  `json:"project_dir,omitempty"`
	ModelID                  string  `json:"model_id,omitempty"`
	ModelDisplay             string  `json:"model_display,omitempty"`
	CostUSD                  float64 `json:"cost_usd,omitempty"`
	DurationMS               int64   `json:"duration_ms,omitempty"`
	APIDurationMS            int64   `json:"api_duration_ms,omitempty"`
	LinesAdded               int     `json:"lines_added,omitempty"`
	LinesRemoved             int     `json:"lines_removed,omitempty"`
	TotalInputTokens         int     `json:"total_input_tokens,omitempty"`
	TotalOutputTokens        int     `json:"total_output_tokens,omitempty"`
	ContextWindowSize        int     `json:"context_window_size,omitempty"`
	CurrentInputTokens       int     `json:"current_input_tokens,omitempty"`
	CurrentOutputTokens      int     `json:"current_output_tokens,omitempty"`
	CurrentCacheCreateTokens int     `json:"current_cache_create_tokens,omitempty"`
	CurrentCacheReadTokens   int     `json:"current_cache_read_tokens,omitempty"`
}

// ingestClaudeStatusline updates the session record and returns a single-line statusline string for Claude Code.
func ingestClaudeStatusline(r io.Reader) (string, error) {
	if f, ok := r.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		return "", errors.New("stdin is a TTY")
	}
	var in ClaudeStatuslineInput
	dec := json.NewDecoder(io.LimitReader(r, 10*1024*1024))
	if err := dec.Decode(&in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.SessionID) == "" {
		return "", errors.New("missing session_id")
	}
	in.SessionID = normalizePlaceholder(in.SessionID)
	if in.SessionID == "" {
		return "", errors.New("invalid session_id")
	}

	cfg := loadConfig()
	now := time.Now().UTC()
	sid := in.SessionID

	// Spool a lightweight patch for aistat to apply during refresh.
	shouldSpool := true
	if p, err := spoolPath(ProviderClaude, "statusline", sid); err == nil {
		if st, err := os.Stat(p); err == nil {
			if now.Sub(st.ModTime()) < cfg.StatuslineMinWrite {
				shouldSpool = false
			}
		}
	}
	if shouldSpool {
		patch := ClaudeStatuslinePatch{
			SessionID:                sid,
			At:                       now.Format(time.RFC3339Nano),
			TranscriptPath:           normalizePlaceholder(in.TranscriptPath),
			ModelID:                  normalizePlaceholder(in.Model.ID),
			ModelDisplay:             normalizePlaceholder(in.Model.DisplayName),
			CostUSD:                  in.Cost.TotalCostUSD,
			DurationMS:               in.Cost.TotalDurationMS,
			APIDurationMS:            in.Cost.TotalAPIDurationMS,
			LinesAdded:               in.Cost.TotalLinesAdded,
			LinesRemoved:             in.Cost.TotalLinesRemoved,
			TotalInputTokens:         in.ContextWindow.TotalInputTokens,
			TotalOutputTokens:        in.ContextWindow.TotalOutputTokens,
			ContextWindowSize:        in.ContextWindow.ContextWindowSize,
			CurrentInputTokens:       0,
			CurrentOutputTokens:      0,
			CurrentCacheCreateTokens: 0,
			CurrentCacheReadTokens:   0,
		}
		if cwd := normalizePlaceholder(in.Workspace.CurrentDir); cwd != "" {
			patch.CWD = cwd
		} else if cwd := normalizePlaceholder(in.CWD); cwd != "" {
			patch.CWD = cwd
		}
		if pd := normalizePlaceholder(in.Workspace.ProjectDir); pd != "" {
			patch.ProjectDir = pd
		}
		if in.ContextWindow.CurrentUsage != nil {
			patch.CurrentInputTokens = in.ContextWindow.CurrentUsage.InputTokens
			patch.CurrentOutputTokens = in.ContextWindow.CurrentUsage.OutputTokens
			patch.CurrentCacheCreateTokens = in.ContextWindow.CurrentUsage.CacheCreationInputTokens
			patch.CurrentCacheReadTokens = in.ContextWindow.CurrentUsage.CacheReadInputTokens
		}
		if b, err := json.Marshal(patch); err == nil {
			_ = writeSpoolBytes(ProviderClaude, "statusline", sid, b, true)
		}
	}
	// Build a compact statusline for Claude Code (ANSI is allowed).
	model := in.Model.DisplayName
	if model == "" {
		model = in.Model.ID
	}
	repo := baseName(in.Workspace.ProjectDir)
	if repo == "" {
		repo = baseName(in.Workspace.CurrentDir)
	}
	cost := fmt.Sprintf("$%.3f", in.Cost.TotalCostUSD)

	// Context % based on current_usage when present.
	ctxPart := ""
	if in.ContextWindow.ContextWindowSize > 0 && in.ContextWindow.CurrentUsage != nil {
		cur := in.ContextWindow.CurrentUsage.InputTokens +
			in.ContextWindow.CurrentUsage.CacheCreationInputTokens +
			in.ContextWindow.CurrentUsage.CacheReadInputTokens +
			in.ContextWindow.CurrentUsage.OutputTokens
		pct := float64(cur) / float64(in.ContextWindow.ContextWindowSize) * 100.0
		ctxPart = fmt.Sprintf("  %.0f%% ctx", pct)
	}

	line := fmt.Sprintf("\x1b[1m%s\x1b[0m  %s  %s%s", model, repo, cost, ctxPart)
	return line, nil
}
