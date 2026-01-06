package widgets

import (
	"fmt"
	"strings"
	"time"
)

// FormatAge formats a duration as a compact age string (alias for FormatAgo)
func FormatAge(d time.Duration) string {
	return FormatAgo(d)
}

// FormatAgo formats a duration as a human-readable age string
func FormatAgo(d time.Duration) string {
	if d < 0 {
		d = -d
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		if s > 0 {
			return fmt.Sprintf("%dm%ds", m, s)
		}
		return fmt.Sprintf("%dm", m)
	}
	if d < 24*time.Hour {
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		if m > 0 {
			return fmt.Sprintf("%dh%dm", h, m)
		}
		return fmt.Sprintf("%dh", h)
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	if hours > 0 {
		return fmt.Sprintf("%dd%dh", days, hours)
	}
	return fmt.Sprintf("%dd", days)
}

// FormatCost formats a cost value
func FormatCost(c float64) string {
	if c <= 0 {
		return ""
	}
	return fmt.Sprintf("$%.3f", c)
}

// FormatSince formats a timestamp as a relative date/time
func FormatSince(ts time.Time) string {
	if ts.IsZero() {
		return ""
	}
	return ts.In(time.Local).Format("01-02 15:04")
}

// FormatTime formats a timestamp for display
func FormatTime(ts time.Time) string {
	if ts.IsZero() {
		return ""
	}
	return ts.In(time.Local).Format("2006-01-02 15:04:05")
}

// TruncateString truncates a string to a maximum length
func TruncateString(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}

// PadRight pads a string to the right to a fixed width
func PadRight(s string, width int) string {
	runes := []rune(s)
	if len(runes) >= width {
		return string(runes[:width])
	}
	return s + strings.Repeat(" ", width-len(runes))
}

// PadLeft pads a string to the left to a fixed width
func PadLeft(s string, width int) string {
	runes := []rune(s)
	if len(runes) >= width {
		return string(runes[:width])
	}
	return strings.Repeat(" ", width-len(runes)) + s
}

// Center centers a string within a fixed width
func Center(s string, width int) string {
	runes := []rune(s)
	if len(runes) >= width {
		return string(runes[:width])
	}
	pad := width - len(runes)
	left := pad / 2
	right := pad - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

// MaxInt returns the maximum of two ints
func MaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// MinInt returns the minimum of two ints
func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ClampInt clamps a value between min and max
func ClampInt(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

// Safe returns the first non-empty string
func Safe(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// FormatInt formats an integer as a string
func FormatInt(n int) string {
	return fmt.Sprintf("%d", n)
}
