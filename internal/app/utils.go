package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func ptrBool(v bool) *bool           { return &v }
func ptrInt(v int) *int              { return &v }
func ptrTime(t time.Time) *time.Time { return &t }

func getString(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch vv := v.(type) {
	case string:
		return vv
	default:
		return fmt.Sprintf("%v", vv)
	}
}

func looksLikeEnvironmentContext(s string) bool {
	return strings.Contains(s, "<environment_context>") || strings.Contains(s, "<cwd>")
}

func asString(v any) string {
	if v == nil {
		return ""
	}
	switch vv := v.(type) {
	case string:
		return vv
	default:
		return fmt.Sprintf("%v", vv)
	}
}

func readTailBytes(path string, maxBytes int) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := st.Size()
	if size <= 0 {
		return []byte{}, nil
	}
	var offset int64
	if size > int64(maxBytes) {
		offset = size - int64(maxBytes)
	} else {
		offset = 0
	}
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}
	return io.ReadAll(f)
}

func splitLines(b []byte) []string {
	s := strings.ReplaceAll(string(b), "\r\n", "\n")
	return strings.Split(s, "\n")
}

func parseRFC3339ish(s string) (time.Time, error) {
	layouts := []string{time.RFC3339Nano, time.RFC3339}
	var lastErr error
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.UTC(), nil
		} else {
			lastErr = err
		}
	}
	if lastErr == nil {
		lastErr = errors.New("unsupported timestamp")
	}
	return time.Time{}, lastErr
}

func safe(primary, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	return fallback
}

func baseName(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	return filepath.Base(p)
}

func maxTime(a, b time.Time) time.Time {
	if a.IsZero() {
		return b
	}
	if b.IsZero() {
		return a
	}
	if b.After(a) {
		return b
	}
	return a
}

func shortenPath(p string, keep int) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	p = filepath.Clean(p)
	parts := strings.Split(p, string(os.PathSeparator))
	if keep <= 0 || len(parts) <= keep {
		return p
	}
	return "…/" + strings.Join(parts[len(parts)-keep:], "/")
}

func redactPath(p string) string {
	if p == "" {
		return ""
	}
	return shortenPath(p, 2)
}

func maybeRedactPath(p string, redact bool) string {
	if !redact {
		return p
	}
	return redactPath(p)
}

func redactProject(p string) string {
	return p
}

func redactIDIfNeeded(id string, redact bool) string {
	if !redact {
		return id
	}
	id = strings.TrimSpace(id)
	if len(id) <= 10 {
		return id
	}
	return id[:6] + "…" + id[len(id)-3:]
}

func redactMessageIfNeeded(s string, redact bool) string {
	if !redact {
		return s
	}
	if s == "" {
		return ""
	}
	return "<redacted>"
}

func fmtAgo(d time.Duration) string {
	if d < 0 {
		d = -d
	}
	switch {
	case d < time.Second:
		return "0s"
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func normalizeList(vals []string) []string {
	var out []string
	for _, v := range vals {
		for _, part := range strings.Split(v, ",") {
			p := strings.TrimSpace(part)
			if p == "" {
				continue
			}
			out = append(out, strings.ToLower(p))
		}
	}
	return out
}

func defaultFields() []string {
	return []string{"provider", "status", "id", "project", "dir", "model", "age", "cost"}
}

func parseFields(vals []string, defaults []string) ([]string, error) {
	fields := normalizeList(vals)
	if len(fields) == 0 {
		return append([]string(nil), defaults...), nil
	}
	allowed := map[string]bool{
		"provider":       true,
		"status":         true,
		"id":             true,
		"project":        true,
		"dir":            true,
		"model":          true,
		"age":            true,
		"since":          true,
		"cost":           true,
		"last_user":      true,
		"last_assistant": true,
	}
	seen := map[string]bool{}
	var out []string
	for _, f := range fields {
		if !allowed[f] {
			return nil, fmt.Errorf("invalid --fields: %s", f)
		}
		if seen[f] {
			continue
		}
		seen[f] = true
		out = append(out, f)
	}
	return out, nil
}

func parseStatusFilters(vals []string) ([]Status, error) {
	if len(vals) == 0 {
		return nil, nil
	}
	var out []Status
	for _, v := range vals {
		norm := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(v, "-", "_")))
		switch norm {
		case string(StatusRunning), string(StatusWaiting), string(StatusApproval), string(StatusStale), string(StatusEnded), string(StatusNeedsAttn):
			out = append(out, Status(norm))
		default:
			return nil, fmt.Errorf("invalid --status: %s", v)
		}
	}
	return out, nil
}

func matchesStatus(status Status, filters []Status) bool {
	if len(filters) == 0 {
		return true
	}
	for _, f := range filters {
		if status == f {
			return true
		}
	}
	return false
}
