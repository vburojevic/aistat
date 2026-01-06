package app

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// -------------------------
// Claude scanning (fallback)
// -------------------------

func scanClaudeTranscripts(cfg Config, now time.Time) ([]SessionRecord, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	projectsDir := filepath.Join(home, ".claude", "projects")

	scanWindow := cfg.ActiveWindow
	if cfg.IncludeEnded {
		scanWindow = cfg.AllScanWindow
	}

	var files []string
	_ = filepath.WalkDir(projectsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if !(strings.HasSuffix(name, ".jsonl") || strings.HasSuffix(name, ".json")) {
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

	var out []SessionRecord
	for _, fp := range files {
		id := strings.TrimSuffix(filepath.Base(fp), filepath.Ext(fp))
		st, err := os.Stat(fp)
		if err != nil {
			continue
		}
		lastSeen := st.ModTime().UTC()
		cwd := scanClaudeTailForCWD(fp, cfg.TailBytesClaude)

		out = append(out, SessionRecord{
			Provider:       ProviderClaude,
			ID:             id,
			TranscriptPath: fp,
			CWD:            cwd,
			LastSeen:       lastSeen,
			LastEvent:      lastSeen,
			LastEventName:  "transcript",
			Status:         StatusUnknown,
			StatusReason:   "observed via transcript",
			UpdatedAt:      now,
		})
	}

	return out, nil
}

func scanClaudeTailForCWD(filePath string, tailBytes int) string {
	b, err := readTailBytes(filePath, tailBytes)
	if err != nil {
		return ""
	}
	lines := splitLines(b)
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			continue
		}
		if cwd, ok := m["cwd"].(string); ok && strings.TrimSpace(cwd) != "" {
			return cwd
		}
	}
	return ""
}
