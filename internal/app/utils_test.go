package app

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseRFC3339ish(t *testing.T) {
	base := "2024-01-02T03:04:05Z"
	got, err := parseRFC3339ish(base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.UTC().Format(time.RFC3339) != base {
		t.Fatalf("unexpected time: %s", got.UTC().Format(time.RFC3339))
	}

	nano := "2024-01-02T03:04:05.123456789Z"
	got, err = parseRFC3339ish(nano)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.UTC().Format(time.RFC3339Nano) != nano {
		t.Fatalf("unexpected time: %s", got.UTC().Format(time.RFC3339Nano))
	}

	if _, err := parseRFC3339ish("not-a-time"); err == nil {
		t.Fatalf("expected error for invalid timestamp")
	}
}

func TestShortenPath(t *testing.T) {
	p := filepath.Join("/", "Users", "me", "project", "subdir")
	short := shortenPath(p, 2)
	if !strings.HasSuffix(short, "project/subdir") {
		t.Fatalf("unexpected shortened path: %q", short)
	}
	if !strings.HasPrefix(short, "…/") {
		t.Fatalf("expected ellipsis prefix, got: %q", short)
	}
}

func TestRedactIDIfNeeded(t *testing.T) {
	id := "1234567890abcdef"
	redacted := redactIDIfNeeded(id, true)
	if redacted == id {
		t.Fatalf("expected redaction")
	}
	if !strings.Contains(redacted, "…") {
		t.Fatalf("expected ellipsis in redacted id")
	}
	short := "shortid"
	if redactIDIfNeeded(short, true) != short {
		t.Fatalf("short id should not be redacted")
	}
}

func TestFmtAgo(t *testing.T) {
	if got := fmtAgo(500 * time.Millisecond); got != "0s" {
		t.Fatalf("expected 0s, got %q", got)
	}
	if got := fmtAgo(5 * time.Second); got != "5s" {
		t.Fatalf("expected 5s, got %q", got)
	}
	if got := fmtAgo(2 * time.Minute); got != "2m" {
		t.Fatalf("expected 2m, got %q", got)
	}
	if got := fmtAgo(3 * time.Hour); got != "3h" {
		t.Fatalf("expected 3h, got %q", got)
	}
	if got := fmtAgo(48 * time.Hour); got != "2d" {
		t.Fatalf("expected 2d, got %q", got)
	}
}
