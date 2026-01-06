package app

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// -------------------------
// Gather + derive session views
// -------------------------

type SessionView struct {
	Provider Provider
	ID       string
	Status   Status
	Reason   string

	Project string
	Dir     string
	Branch  string // Git branch name for worktree identification
	Model   string
	Cost    float64
	Age     time.Duration

	LastSeen time.Time

	SourcePath string
	Detail     string
	LastUser   string
	LastAssist string
}

func gatherSessions(cfg Config) ([]SessionView, error) {
	now := time.Now().UTC()

	stored, err := loadAllRecords()
	if err != nil {
		return nil, err
	}

	merged := map[string]SessionRecord{}
	for _, r := range stored {
		merged[keyFor(r.Provider, r.ID)] = r
	}

	// Fallback scans (Codex rollouts are essential for real-time; Claude scan is a fallback)
	if cfg.ProviderFilter == "" || cfg.ProviderFilter == string(ProviderCodex) {
		codexScan, _ := scanCodexRollouts(cfg, now)
		for _, r := range codexScan {
			mergeInto(merged, r)
		}
	}
	if cfg.ProviderFilter == "" || cfg.ProviderFilter == string(ProviderClaude) {
		claudeScan, _ := scanClaudeTranscripts(cfg, now)
		for _, r := range claudeScan {
			mergeInto(merged, r)
		}
	}

	var views []SessionView
	for _, r := range merged {
		// Provider filter
		if cfg.ProviderFilter != "" && string(r.Provider) != cfg.ProviderFilter {
			continue
		}
		if len(cfg.ProjectFilters) > 0 && !matchesProject(projectNameForRecord(r), cfg.ProjectFilters) {
			continue
		}
		v := makeView(r, now, cfg)

		if !cfg.IncludeEnded {
			if v.Status == StatusEnded || v.Status == StatusStale {
				continue
			}
		}
		if !matchesStatus(v.Status, cfg.StatusFilters) {
			continue
		}

		// Active filter: unless --all, show only active-window sessions
		if !cfg.IncludeEnded && v.Age > cfg.ActiveWindow {
			continue
		}

		views = append(views, v)
	}

	sortSessions(views, cfg.SortBy)

	if cfg.MaxSessions > 0 && len(views) > cfg.MaxSessions {
		views = views[:cfg.MaxSessions]
	}

	return views, nil
}

func keyFor(p Provider, id string) string {
	return string(p) + ":" + id
}

func mergeInto(dst map[string]SessionRecord, src SessionRecord) {
	k := keyFor(src.Provider, src.ID)
	cur, ok := dst[k]
	if !ok {
		dst[k] = src
		return
	}
	// Merge fields (prefer non-empty, prefer newer LastSeen)
	if cur.TranscriptPath == "" {
		cur.TranscriptPath = src.TranscriptPath
	}
	if cur.RolloutPath == "" {
		cur.RolloutPath = src.RolloutPath
	}
	if cur.CWD == "" {
		cur.CWD = src.CWD
	}
	if cur.ProjectDir == "" {
		cur.ProjectDir = src.ProjectDir
	}
	if cur.ModelID == "" {
		cur.ModelID = src.ModelID
	}
	if cur.ModelDisplay == "" {
		cur.ModelDisplay = src.ModelDisplay
	}
	if cur.ApprovalPolicy == "" {
		cur.ApprovalPolicy = src.ApprovalPolicy
	}

	// Activity
	if src.LastSeen.After(cur.LastSeen) {
		cur.LastSeen = src.LastSeen
	}
	if src.LastEvent.After(cur.LastEvent) {
		cur.LastEvent = src.LastEvent
		cur.LastEventName = src.LastEventName
	}
	// Keep explicit status from hooks/notify; if empty, take src.
	if cur.Status == "" || cur.Status == StatusUnknown {
		if src.Status != "" {
			cur.Status = src.Status
			cur.StatusReason = src.StatusReason
		}
	}
	// Preserve endedAt if we have one
	if cur.EndedAt == nil && src.EndedAt != nil {
		cur.EndedAt = src.EndedAt
	}
	// Prefer higher-fidelity Claude numbers (non-zero)
	if src.CostUSD != 0 {
		cur.CostUSD = src.CostUSD
		cur.DurationMS = src.DurationMS
		cur.APIDurationMS = src.APIDurationMS
		cur.LinesAdded = src.LinesAdded
		cur.LinesRemoved = src.LinesRemoved

		cur.TotalInputTokens = src.TotalInputTokens
		cur.TotalOutputTokens = src.TotalOutputTokens
		cur.ContextWindowSize = src.ContextWindowSize
		cur.CurrentInputTokens = src.CurrentInputTokens
		cur.CurrentOutputTokens = src.CurrentOutputTokens
		cur.CurrentCacheCreateTokens = src.CurrentCacheCreateTokens
		cur.CurrentCacheReadTokens = src.CurrentCacheReadTokens
	}

	// Codex notify metadata
	if cur.Title == "" {
		cur.Title = src.Title
	}
	if cur.Message == "" {
		cur.Message = src.Message
	}
	if cur.ThreadID == "" {
		cur.ThreadID = src.ThreadID
	}
	if cur.TurnID == "" {
		cur.TurnID = src.TurnID
	}
	if src.LastUserText != "" {
		cur.LastUserText = src.LastUserText
	}
	if src.LastAssistantText != "" {
		cur.LastAssistantText = src.LastAssistantText
	}

	dst[k] = cur
}

func makeView(r SessionRecord, now time.Time, cfg Config) SessionView {
	last := r.LastSeen
	if last.IsZero() {
		last = r.UpdatedAt
	}
	if last.IsZero() {
		last = now
	}
	age := now.Sub(last)

	status, reason := deriveStatus(r, now, cfg)

	project := baseName(r.ProjectDir)
	if project == "" {
		project = baseName(r.CWD)
	}
	dir := r.CWD
	model := r.ModelDisplay
	if model == "" {
		model = r.ModelID
	}

	source := ""
	if r.Provider == ProviderClaude {
		source = r.TranscriptPath
	} else {
		source = r.RolloutPath
	}

	displayID := r.ID
	if cfg.Redact {
		displayID = redactIDIfNeeded(r.ID, true)
		project = redactProject(project)
		dir = shortenPath(dir, 2)
		source = redactPath(source)
	}

	detail := buildDetail(r, status, reason, cfg, now)
	lastUser := ""
	lastAssistant := ""
	if cfg.IncludeLastMsg {
		lastUser = redactMessageIfNeeded(r.LastUserText, cfg.Redact)
		lastAssistant = redactMessageIfNeeded(r.LastAssistantText, cfg.Redact)
	}

	// Get git branch for worktree identification
	branch := getBranchName(r.CWD)

	return SessionView{
		Provider:   r.Provider,
		ID:         displayID,
		Status:     status,
		Reason:     reason,
		Project:    project,
		Dir:        dir,
		Branch:     branch,
		Model:      model,
		Cost:       r.CostUSD,
		Age:        age,
		LastSeen:   last,
		SourcePath: source,
		Detail:     detail,
		LastUser:   lastUser,
		LastAssist: lastAssistant,
	}
}

func deriveStatus(r SessionRecord, now time.Time, cfg Config) (Status, string) {
	// Ended wins.
	if r.EndedAt != nil && !r.EndedAt.IsZero() {
		return StatusEnded, "ended"
	}

	age := now.Sub(nonZeroTime(r.LastSeen, r.UpdatedAt, now))

	if age > cfg.ActiveWindow && !cfg.IncludeEnded {
		return StatusStale, fmt.Sprintf("stale (%s)", fmtAgo(age))
	}

	// Explicit approval wins.
	if r.Status == StatusApproval || r.LastNotificationType == "permission_prompt" {
		return StatusApproval, "awaiting approval"
	}

	// Running heuristic: very recent activity.
	if age <= cfg.RunningWindow {
		return StatusRunning, "running"
	}

	// Explicit statuses from hooks/notify
	switch r.Status {
	case StatusWaiting:
		if r.StatusReason != "" {
			return StatusWaiting, r.StatusReason
		}
		return StatusWaiting, "awaiting input"
	case StatusNeedsAttn:
		return StatusNeedsAttn, "needs attention"
	case StatusRunning:
		return StatusRunning, "active"
	}

	// Fallback
	return StatusWaiting, "awaiting input"
}

func nonZeroTime(ts ...time.Time) time.Time {
	for _, t := range ts {
		if !t.IsZero() {
			return t
		}
	}
	return time.Now().UTC()
}

// getBranchName returns the git branch for a directory
func getBranchName(dir string) string {
	if dir == "" {
		return ""
	}
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func buildDetail(r SessionRecord, status Status, reason string, cfg Config, now time.Time) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Status: %s — %s\n", status, reason)

	if r.Provider == ProviderClaude {
		if r.ModelDisplay != "" || r.ModelID != "" {
			fmt.Fprintf(&b, "Model: %s\n", safe(r.ModelDisplay, r.ModelID))
		}
		if r.ProjectDir != "" {
			fmt.Fprintf(&b, "Project: %s\n", maybeRedactPath(r.ProjectDir, cfg.Redact))
		}
		if r.CWD != "" {
			fmt.Fprintf(&b, "CWD: %s\n", maybeRedactPath(r.CWD, cfg.Redact))
		}
		if r.CostUSD != 0 {
			fmt.Fprintf(&b, "Cost: $%.4f\n", r.CostUSD)
		}
		if r.ContextWindowSize > 0 && r.CurrentInputTokens > 0 {
			cur := r.CurrentInputTokens + r.CurrentOutputTokens + r.CurrentCacheCreateTokens + r.CurrentCacheReadTokens
			pct := float64(cur) / float64(r.ContextWindowSize) * 100
			fmt.Fprintf(&b, "Context: %d/%d (%.0f%%)\n", cur, r.ContextWindowSize, pct)
		}
		if r.TranscriptPath != "" {
			fmt.Fprintf(&b, "Transcript: %s\n", maybeRedactPath(r.TranscriptPath, cfg.Redact))
		}
	} else {
		if r.ModelID != "" {
			fmt.Fprintf(&b, "Model: %s\n", r.ModelID)
		}
		if r.CWD != "" {
			fmt.Fprintf(&b, "CWD: %s\n", maybeRedactPath(r.CWD, cfg.Redact))
		}
		if r.ApprovalPolicy != "" {
			fmt.Fprintf(&b, "Approval policy: %s\n", r.ApprovalPolicy)
		}
		if r.ThreadID != "" || r.TurnID != "" {
			fmt.Fprintf(&b, "Thread/Turn: %s / %s\n", safe(r.ThreadID, "-"), safe(r.TurnID, "-"))
		}
		if r.Title != "" {
			fmt.Fprintf(&b, "Title: %s\n", redactMessageIfNeeded(r.Title, cfg.Redact))
		}
		if r.Message != "" {
			fmt.Fprintf(&b, "Message: %s\n", redactMessageIfNeeded(r.Message, cfg.Redact))
		}
		if r.RolloutPath != "" {
			fmt.Fprintf(&b, "Rollout: %s\n", maybeRedactPath(r.RolloutPath, cfg.Redact))
		}
		if cfg.IncludeLastMsg {
			if r.LastUserText != "" {
				fmt.Fprintf(&b, "Last user: %s\n", redactMessageIfNeeded(r.LastUserText, cfg.Redact))
			}
			if r.LastAssistantText != "" {
				fmt.Fprintf(&b, "Last assistant: %s\n", redactMessageIfNeeded(r.LastAssistantText, cfg.Redact))
			}
		}
	}

	if !r.LastSeen.IsZero() {
		fmt.Fprintf(&b, "Last: %s ago\n", fmtAgo(now.Sub(r.LastSeen)))
	}
	return b.String()
}

func matchesProject(project string, filters []string) bool {
	if project == "" {
		return false
	}
	p := strings.ToLower(project)
	for _, f := range filters {
		if f == p {
			return true
		}
	}
	return false
}

func projectNameForRecord(r SessionRecord) string {
	project := baseName(r.ProjectDir)
	if project == "" {
		project = baseName(r.CWD)
	}
	return project
}

func sortSessions(views []SessionView, sortBy string) {
	sortKey := strings.ToLower(strings.TrimSpace(sortBy))
	if sortKey == "" {
		sortKey = "last_seen"
	}
	sort.SliceStable(views, func(i, j int) bool {
		a := views[i]
		b := views[j]
		switch sortKey {
		case "status":
			if a.Status != b.Status {
				return string(a.Status) < string(b.Status)
			}
		case "provider":
			if a.Provider != b.Provider {
				return string(a.Provider) < string(b.Provider)
			}
		case "cost":
			if a.Cost != b.Cost {
				return a.Cost > b.Cost
			}
		case "project":
			if strings.ToLower(a.Project) != strings.ToLower(b.Project) {
				return strings.ToLower(a.Project) < strings.ToLower(b.Project)
			}
		default:
			// last_seen
		}
		return a.LastSeen.After(b.LastSeen)
	})
}

type SessionGroup struct {
	Group    string        `json:"group"`
	Sessions []SessionView `json:"sessions"`
}

func groupSessions(views []SessionView, groupBy string) []SessionGroup {
	groupKey := strings.ToLower(strings.TrimSpace(groupBy))
	if groupKey == "" {
		return []SessionGroup{{Group: "", Sessions: views}}
	}

	order := []string{}
	groups := map[string][]SessionView{}
	for _, v := range views {
		key := ""
		switch groupKey {
		case "provider":
			key = string(v.Provider)
		case "project":
			key = v.Project
		case "status":
			key = string(v.Status)
		case "day":
			key = v.LastSeen.In(time.Local).Format("2006-01-02")
		case "hour":
			key = v.LastSeen.In(time.Local).Format("2006-01-02 15:00")
		default:
			key = ""
		}
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], v)
	}

	out := make([]SessionGroup, 0, len(order))
	for _, k := range order {
		out = append(out, SessionGroup{Group: k, Sessions: groups[k]})
	}
	return out
}
