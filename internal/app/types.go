package app

import (
	"regexp"
	"time"
)

const (
	appName                = "aistat"
	defaultActiveWindow    = 30 * time.Minute
	defaultRunningWindow   = 3 * time.Second
	defaultRefreshInterval = 1 * time.Second
	defaultAllScanWindow   = 7 * 24 * time.Hour
	defaultMaxSessions     = 50

	defaultTailBytesCodex  = 1024 * 512 // 512KB
	defaultTailBytesClaude = 1024 * 256 // 256KB
	defaultHeaderScanLines = 5000

	// Claude Code status line can update very frequently; this throttles disk writes.
	defaultStatuslineMinWrite = 800 * time.Millisecond
)

type Provider string

const (
	ProviderClaude Provider = "claude"
	ProviderCodex  Provider = "codex"
)

type Status string

const (
	StatusRunning   Status = "running"
	StatusWaiting   Status = "waiting"
	StatusApproval  Status = "approval"
	StatusEnded     Status = "ended"
	StatusStale     Status = "stale"
	StatusUnknown   Status = "unknown"
	StatusNeedsAttn Status = "needs_attention"
)

type SessionRecord struct {
	Provider Provider `json:"provider"`
	ID       string   `json:"id"`

	// Paths
	TranscriptPath string `json:"transcript_path,omitempty"` // Claude
	RolloutPath    string `json:"rollout_path,omitempty"`    // Codex

	// Working dirs
	CWD        string `json:"cwd,omitempty"`
	ProjectDir string `json:"project_dir,omitempty"`

	// Model
	ModelID      string `json:"model_id,omitempty"`
	ModelDisplay string `json:"model_display,omitempty"`

	// Claude metrics
	CostUSD       float64 `json:"cost_usd,omitempty"`
	DurationMS    int64   `json:"duration_ms,omitempty"`
	APIDurationMS int64   `json:"api_duration_ms,omitempty"`
	LinesAdded    int     `json:"lines_added,omitempty"`
	LinesRemoved  int     `json:"lines_removed,omitempty"`

	TotalInputTokens         int `json:"total_input_tokens,omitempty"`
	TotalOutputTokens        int `json:"total_output_tokens,omitempty"`
	ContextWindowSize        int `json:"context_window_size,omitempty"`
	CurrentInputTokens       int `json:"current_input_tokens,omitempty"`
	CurrentOutputTokens      int `json:"current_output_tokens,omitempty"`
	CurrentCacheCreateTokens int `json:"current_cache_create_tokens,omitempty"`
	CurrentCacheReadTokens   int `json:"current_cache_read_tokens,omitempty"`

	// Codex notify metadata
	ThreadID string `json:"thread_id,omitempty"`
	TurnID   string `json:"turn_id,omitempty"`
	Title    string `json:"title,omitempty"`
	Message  string `json:"message,omitempty"`
	// Last message snippets (best-effort; provider-specific)
	LastUserText      string `json:"last_user_text,omitempty"`
	LastAssistantText string `json:"last_assistant_text,omitempty"`

	ApprovalPolicy string `json:"approval_policy,omitempty"`

	// Activity / status
	LastSeen             time.Time  `json:"last_seen,omitempty"` // last “heartbeat” (statusline/log tail/modtime)
	LastEvent            time.Time  `json:"last_event,omitempty"`
	LastEventName        string     `json:"last_event_name,omitempty"`
	LastNotificationType string     `json:"last_notification_type,omitempty"`
	LastNotificationMsg  string     `json:"last_notification_msg,omitempty"`
	Status               Status     `json:"status,omitempty"`        // last known explicit status (from hooks/notify)
	StatusReason         string     `json:"status_reason,omitempty"` // human readable
	EndedAt              *time.Time `json:"ended_at,omitempty"`

	UpdatedAt time.Time `json:"updated_at,omitempty"` // when we last wrote this record
}

type Config struct {
	Redact         bool
	ActiveWindow   time.Duration
	RunningWindow  time.Duration
	RefreshEvery   time.Duration
	MaxSessions    int
	IncludeEnded   bool
	ProviderFilter string // "", "claude", "codex"
	NoColor        bool
	AllScanWindow  time.Duration
	ProjectFilters []string
	StatusFilters  []Status
	SortBy         string
	GroupBy        string
	IncludeLastMsg bool

	// Internal tuning
	TailBytesCodex     int
	TailBytesClaude    int
	HeaderScanLines    int
	StatuslineMinWrite time.Duration
}

type ConfigFile struct {
	Redact             *bool  `json:"redact,omitempty"`
	ActiveWindow       string `json:"active_window,omitempty"`
	RunningWindow      string `json:"running_window,omitempty"`
	RefreshEvery       string `json:"refresh_every,omitempty"`
	MaxSessions        *int   `json:"max_sessions,omitempty"`
	AllScanWindow      string `json:"all_scan_window,omitempty"`
	StatuslineMinWrite string `json:"statusline_min_write,omitempty"`
}

var (
	fileSafeRe = regexp.MustCompile(`[^A-Za-z0-9._-]+`)
)
