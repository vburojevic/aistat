package app

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// -------------------------
// Codex ingestion + scanning
// -------------------------

type CodexNotifyPayload struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Data      struct {
		CWD       string `json:"cwd"`
		Message   string `json:"message"`
		Title     string `json:"title"`
		SessionID string `json:"session_id"`
		ThreadID  string `json:"thread_id"`
		TurnID    string `json:"turn_id"`
	} `json:"data"`
}

func ingestCodexNotify(r io.Reader) error {
	b, err := io.ReadAll(io.LimitReader(r, 10*1024*1024))
	if err != nil {
		return err
	}
	var n CodexNotifyPayload
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	id := strings.TrimSpace(n.Data.SessionID)
	if id == "" {
		id = strings.TrimSpace(n.Data.ThreadID)
	}
	if id == "" {
		// Avoid breaking user’s Codex. Just no-op.
		return nil
	}

	ts := time.Now().UTC()
	if n.Timestamp != "" {
		if parsed, err := parseRFC3339ish(n.Timestamp); err == nil {
			ts = parsed
		}
	}

	return updateRecord(ProviderCodex, id, func(rec *SessionRecord) {
		rec.LastSeen = maxTime(rec.LastSeen, ts)
		rec.LastEvent = maxTime(rec.LastEvent, ts)
		rec.LastEventName = "notify:" + n.Type

		if n.Data.CWD != "" {
			rec.CWD = n.Data.CWD
		}
		rec.ThreadID = n.Data.ThreadID
		rec.TurnID = n.Data.TurnID
		rec.Title = n.Data.Title
		rec.Message = n.Data.Message

		rec.Status = StatusWaiting
		rec.StatusReason = "turn complete"
	})
}

type codexLogEntry struct {
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

type codexHeader struct {
	SessionID      string
	CWD            string
	Model          string
	ApprovalPolicy string
	CreatedAt      time.Time
}

type codexTail struct {
	LastTS            time.Time
	LastEntryType     string
	LastPayloadType   string
	LastRole          string
	LastUserText      string
	LastAssistantText string
}

func scanCodexRollouts(cfg Config, now time.Time) ([]SessionRecord, error) {
	codexHome := os.Getenv("CODEX_HOME")
	if strings.TrimSpace(codexHome) == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		codexHome = filepath.Join(home, ".codex")
	}

	sessionDirs := []string{
		filepath.Join(codexHome, "sessions"),
		filepath.Join(codexHome, "archived_sessions"),
	}

	scanWindow := cfg.ActiveWindow
	if cfg.IncludeEnded {
		scanWindow = cfg.AllScanWindow
	}

	var files []string
	for _, dir := range sessionDirs {
		_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			name := d.Name()
			if !strings.HasPrefix(name, "rollout-") || !strings.HasSuffix(name, ".jsonl") {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return nil
			}
			if now.Sub(info.ModTime().UTC()) > scanWindow {
				return nil
			}
			files = append(files, path)
			return nil
		})
	}

	var out []SessionRecord
	for _, fp := range files {
		hdr, err := scanCodexHeader(fp, cfg.HeaderScanLines)
		if err != nil {
			continue
		}
		tail, err := scanCodexTail(fp, cfg.TailBytesCodex)
		if err != nil {
			continue
		}
		id := hdr.SessionID
		if id == "" {
			id = strings.TrimSuffix(filepath.Base(fp), ".jsonl")
		}

		rec := SessionRecord{
			Provider:       ProviderCodex,
			ID:             id,
			RolloutPath:    fp,
			CWD:            hdr.CWD,
			ModelID:        hdr.Model,
			ApprovalPolicy: hdr.ApprovalPolicy,
			LastSeen:       tail.LastTS,
			LastEvent:      tail.LastTS,
			LastEventName:  fmt.Sprintf("%s/%s", tail.LastEntryType, tail.LastPayloadType),
			UpdatedAt:      now,
		}

		// If we couldn't parse a timestamp, fall back to modtime.
		if rec.LastSeen.IsZero() {
			if st, err := os.Stat(fp); err == nil {
				rec.LastSeen = st.ModTime().UTC()
			} else {
				rec.LastSeen = now
			}
			rec.LastEvent = rec.LastSeen
		}

		// Heuristic: mark approvals if the tail indicates approval requested.
		if strings.Contains(strings.ToLower(tail.LastPayloadType), "approval") {
			rec.Status = StatusApproval
			rec.StatusReason = "awaiting approval"
		}

		out = append(out, rec)
	}

	return out, nil
}

func scanCodexHeader(filePath string, maxLines int) (codexHeader, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return codexHeader{}, err
	}
	defer f.Close()

	var hdr codexHeader
	sc := bufio.NewScanner(f)
	buf := make([]byte, 0, 1024*1024)
	sc.Buffer(buf, 50*1024*1024)

	lines := 0
	for sc.Scan() {
		lines++
		if lines > maxLines {
			break
		}
		var e codexLogEntry
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			continue
		}
		switch e.Type {
		case "session_meta":
			var payload map[string]any
			_ = json.Unmarshal(e.Payload, &payload)
			if hdr.SessionID == "" {
				hdr.SessionID = asString(payload["id"])
			}
			if hdr.CWD == "" {
				hdr.CWD = asString(payload["cwd"])
			}
			if hdr.CreatedAt.IsZero() {
				if ts := asString(payload["timestamp"]); ts != "" {
					if t, err := parseRFC3339ish(ts); err == nil {
						hdr.CreatedAt = t
					}
				}
			}
		case "turn_context":
			var payload map[string]any
			_ = json.Unmarshal(e.Payload, &payload)
			if hdr.CWD == "" {
				hdr.CWD = asString(payload["cwd"])
			}
			if hdr.Model == "" {
				hdr.Model = asString(payload["model"])
			}
			if hdr.ApprovalPolicy == "" {
				hdr.ApprovalPolicy = asString(payload["approval_policy"])
			}
		}
		if hdr.SessionID != "" && hdr.CWD != "" && hdr.Model != "" {
			break
		}
	}
	if err := sc.Err(); err != nil {
		return codexHeader{}, err
	}
	if hdr.SessionID == "" && hdr.CWD == "" && hdr.Model == "" {
		return codexHeader{}, errors.New("no header found")
	}
	return hdr, nil
}

func scanCodexTail(filePath string, tailBytes int) (codexTail, error) {
	b, err := readTailBytes(filePath, tailBytes)
	if err != nil {
		return codexTail{}, err
	}
	lines := splitLines(b)
	if len(lines) == 0 {
		return codexTail{}, errors.New("empty tail")
	}

	var tail codexTail
	// Iterate backwards to find last meaningful entry + last user/assistant message.
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		var e codexLogEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}

		// Set "last" entry if not set yet.
		if tail.LastEntryType == "" {
			tail.LastEntryType = e.Type

			// payload.type if present
			var payload map[string]any
			if len(e.Payload) > 0 {
				_ = json.Unmarshal(e.Payload, &payload)
			}
			tail.LastPayloadType = asString(payload["type"])
			tail.LastRole = asString(payload["role"])

			if e.Timestamp != "" {
				if t, err := parseRFC3339ish(e.Timestamp); err == nil {
					tail.LastTS = t
				}
			}
		}

		// Capture last user + assistant snippets (optional).
		if e.Type == "response_item" {
			var payload map[string]any
			_ = json.Unmarshal(e.Payload, &payload)
			if asString(payload["type"]) == "message" {
				role := asString(payload["role"])
				content, _ := payload["content"].([]any)
				text := extractCodexMessageText(role, content)
				text = strings.TrimSpace(text)
				if role == "assistant" && tail.LastAssistantText == "" && text != "" {
					tail.LastAssistantText = text
				}
				if role == "user" && tail.LastUserText == "" && text != "" && !looksLikeEnvironmentContext(text) {
					tail.LastUserText = text
				}
			}
		}

		if tail.LastEntryType != "" && !tail.LastTS.IsZero() && tail.LastUserText != "" && tail.LastAssistantText != "" {
			break
		}
	}

	return tail, nil
}

func extractCodexMessageText(role string, content []any) string {
	if len(content) == 0 {
		return ""
	}
	var parts []string
	for _, item := range content {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		tp := asString(m["type"])
		switch role {
		case "user":
			if tp == "input_text" {
				if txt := asString(m["text"]); txt != "" {
					parts = append(parts, txt)
				}
			}
		default: // assistant or unknown
			if txt := asString(m["text"]); txt != "" {
				parts = append(parts, txt)
			}
		}
	}
	return strings.Join(parts, "\n")
}
