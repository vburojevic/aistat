package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// -------------------------
// Claude ingestion
// -------------------------

func ingestClaudeHook(r io.Reader) error {
	b, err := io.ReadAll(io.LimitReader(r, 10*1024*1024))
	if err != nil {
		return err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}

	event := strings.TrimSpace(getString(m, "hook_event_name"))
	sid := strings.TrimSpace(getString(m, "session_id"))
	if sid == "" {
		// No session => ignore (but return nil to not break hooks)
		return nil
	}

	now := time.Now().UTC()
	tp := getString(m, "transcript_path")
	cwd := getString(m, "cwd")

	notifType := getString(m, "notification_type")
	notifMsg := getString(m, "message")

	return updateRecord(ProviderClaude, sid, func(rec *SessionRecord) {
		if tp != "" {
			rec.TranscriptPath = tp
		}
		if cwd != "" {
			rec.CWD = cwd
		}
		rec.LastEvent = now
		rec.LastEventName = event
		rec.LastSeen = maxTime(rec.LastSeen, now)

		switch event {
		case "SessionStart":
			rec.Status = StatusRunning
			rec.StatusReason = "session started"
			rec.EndedAt = nil
			rec.LastNotificationType = ""
			rec.LastNotificationMsg = ""
		case "UserPromptSubmit":
			rec.Status = StatusRunning
			rec.StatusReason = "user prompt submitted"
			rec.LastNotificationType = ""
			rec.LastNotificationMsg = ""
		case "PreToolUse", "PostToolUse":
			rec.Status = StatusRunning
			rec.StatusReason = "tool activity"
			rec.LastNotificationType = ""
			rec.LastNotificationMsg = ""
		case "Stop":
			rec.Status = StatusWaiting
			rec.StatusReason = "awaiting input"
		case "Notification":
			rec.LastNotificationType = notifType
			rec.LastNotificationMsg = notifMsg
			switch notifType {
			case "permission_prompt":
				rec.Status = StatusApproval
				rec.StatusReason = "awaiting approval"
			case "idle_prompt":
				rec.Status = StatusWaiting
				rec.StatusReason = "awaiting input"
			default:
				rec.Status = StatusWaiting
				if notifType != "" {
					rec.StatusReason = "notification: " + notifType
				} else {
					rec.StatusReason = "notification"
				}
			}
		case "SessionEnd":
			rec.Status = StatusEnded
			rec.StatusReason = "session ended"
			rec.EndedAt = ptrTime(now)
		default:
			// keep as-is
		}
	})
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

// ingestClaudeStatusline updates the session record and returns a single-line statusline string for Claude Code.
func ingestClaudeStatusline(r io.Reader) (string, error) {
	b, err := io.ReadAll(io.LimitReader(r, 10*1024*1024))
	if err != nil {
		return "", err
	}
	var in ClaudeStatuslineInput
	if err := json.Unmarshal(b, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.SessionID) == "" {
		return "", errors.New("missing session_id")
	}

	cfg := loadConfig()
	now := time.Now().UTC()
	sid := in.SessionID

	// Throttled write (statusline can be called a lot).
	p, err := recordPath(ProviderClaude, sid)
	if err != nil {
		return "", err
	}
	lock := p + ".lock"
	_ = withLock(lock, func() error {
		rec := SessionRecord{Provider: ProviderClaude, ID: sid}
		if existing, err := loadRecord(p); err == nil {
			rec = existing
		}

		if !rec.UpdatedAt.IsZero() && now.Sub(rec.UpdatedAt) < cfg.StatuslineMinWrite && rec.ModelID != "" {
			// Skip writing to disk; we'll refresh later.
			return nil
		}

		if in.TranscriptPath != "" {
			rec.TranscriptPath = in.TranscriptPath
		}
		if in.Workspace.CurrentDir != "" {
			rec.CWD = in.Workspace.CurrentDir
		} else if in.CWD != "" {
			rec.CWD = in.CWD
		}
		if in.Workspace.ProjectDir != "" {
			rec.ProjectDir = in.Workspace.ProjectDir
		}
		rec.ModelID = in.Model.ID
		rec.ModelDisplay = in.Model.DisplayName

		rec.CostUSD = in.Cost.TotalCostUSD
		rec.DurationMS = in.Cost.TotalDurationMS
		rec.APIDurationMS = in.Cost.TotalAPIDurationMS
		rec.LinesAdded = in.Cost.TotalLinesAdded
		rec.LinesRemoved = in.Cost.TotalLinesRemoved

		rec.TotalInputTokens = in.ContextWindow.TotalInputTokens
		rec.TotalOutputTokens = in.ContextWindow.TotalOutputTokens
		rec.ContextWindowSize = in.ContextWindow.ContextWindowSize
		if in.ContextWindow.CurrentUsage != nil {
			rec.CurrentInputTokens = in.ContextWindow.CurrentUsage.InputTokens
			rec.CurrentOutputTokens = in.ContextWindow.CurrentUsage.OutputTokens
			rec.CurrentCacheCreateTokens = in.ContextWindow.CurrentUsage.CacheCreationInputTokens
			rec.CurrentCacheReadTokens = in.ContextWindow.CurrentUsage.CacheReadInputTokens
		}

		rec.LastSeen = maxTime(rec.LastSeen, now)
		if rec.Status == "" || rec.Status == StatusUnknown {
			rec.Status = StatusRunning
			rec.StatusReason = "active"
		}
		rec.UpdatedAt = now

		// Ensure dirs exist then save.
		_ = ensureAppDirs()
		return saveRecord(p, rec)
	})

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
