package state

import (
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
)

// Provider and Status are re-exported from the app package
type Provider string
type Status string

const (
	ProviderClaude Provider = "claude"
	ProviderCodex  Provider = "codex"
)

const (
	StatusRunning   Status = "running"
	StatusWaiting   Status = "waiting"
	StatusApproval  Status = "approval"
	StatusEnded     Status = "ended"
	StatusStale     Status = "stale"
	StatusUnknown   Status = "unknown"
	StatusNeedsAttn Status = "needs_attention"
)

// SessionView represents a session for display
type SessionView struct {
	Provider Provider
	ID       string
	Status   Status
	Reason   string

	Project string
	Dir     string
	Model   string
	Cost    float64
	Age     time.Duration

	LastSeen time.Time

	SourcePath string
	Detail     string
	LastUser   string
	LastAssist string
}

// RowKind distinguishes between session rows and group header rows
type RowKind int

const (
	RowSession RowKind = iota
	RowGroup
)

// RowMeta holds metadata for each table row
type RowMeta struct {
	Kind    RowKind
	Session *SessionView
	Group   string
}

// ProjectItem represents a project in the dashboard/picker
type ProjectItem struct {
	Name        string
	Count       int
	LastSeen    time.Time
	StatusCount map[Status]int
	Providers   map[Provider]int
}

// AppState holds the shared state for the TUI
type AppState struct {
	// Session data
	AllSessions      []SessionView
	FilteredSessions []SessionView
	RowMeta          []RowMeta

	// Project data
	ProjectItems []ProjectItem

	// Derived counts
	FilterCounts map[Status]int
	FilterCost   float64
	FilterTotal  int

	// Selection and pinning
	Selected map[string]bool
	Pinned   map[string]bool

	// Change tracking
	ChangedAt    map[string]time.Time
	LastSnapshot map[string]SessionView
	History      map[string][]time.Time
	LastOrder    map[string]int
	MoveDir      map[string]int

	// Last refresh
	LastRefresh time.Time
	Err         error
}

// NewAppState creates a new AppState with initialized maps
func NewAppState() *AppState {
	return &AppState{
		FilterCounts: make(map[Status]int),
		Selected:     make(map[string]bool),
		Pinned:       make(map[string]bool),
		ChangedAt:    make(map[string]time.Time),
		LastSnapshot: make(map[string]SessionView),
		History:      make(map[string][]time.Time),
		LastOrder:    make(map[string]int),
		MoveDir:      make(map[string]int),
	}
}

// UpdateSessions updates the session list and marks changes
func (s *AppState) UpdateSessions(sessions []SessionView, refreshInterval time.Duration) {
	s.markChanges(sessions)
	s.AllSessions = sessions
	s.ProjectItems = BuildProjectItems(sessions)
	s.LastRefresh = time.Now().UTC()
}

// markChanges tracks which sessions have changed recently
func (s *AppState) markChanges(sessions []SessionView) {
	now := time.Now().UTC()
	for _, sess := range sessions {
		id := stripANSI(sess.ID)
		prev, ok := s.LastSnapshot[id]
		if !ok || prev.Status != sess.Status || !prev.LastSeen.Equal(sess.LastSeen) || prev.Cost != sess.Cost {
			s.ChangedAt[id] = now
			s.History[id] = append(s.History[id], now)
			if len(s.History[id]) > 8 {
				s.History[id] = s.History[id][len(s.History[id])-8:]
			}
		}
		s.LastSnapshot[id] = sess
	}
}

// IsRecentlyChanged checks if a session changed recently
func (s *AppState) IsRecentlyChanged(id string, refreshInterval time.Duration) bool {
	when, ok := s.ChangedAt[id]
	if !ok {
		return false
	}
	if time.Since(when) > 2*refreshInterval {
		delete(s.ChangedAt, id)
		return false
	}
	return true
}

// TogglePin toggles the pinned state of a session
func (s *AppState) TogglePin(id string) {
	s.Pinned[id] = !s.Pinned[id]
	if !s.Pinned[id] {
		delete(s.Pinned, id)
	}
}

// ToggleSelect toggles the selected state of a session
func (s *AppState) ToggleSelect(id string) {
	s.Selected[id] = !s.Selected[id]
	if !s.Selected[id] {
		delete(s.Selected, id)
	}
}

// SelectedIDs returns the list of selected session IDs
func (s *AppState) SelectedIDs() []string {
	if len(s.Selected) == 0 {
		return nil
	}
	ids := make([]string, 0, len(s.Selected))
	for id := range s.Selected {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// ApplyPinnedFirst reorders sessions so pinned ones come first
func (s *AppState) ApplyPinnedFirst(list []SessionView) []SessionView {
	if len(s.Pinned) == 0 {
		return list
	}
	pinned := make([]SessionView, 0, len(list))
	rest := make([]SessionView, 0, len(list))
	for _, sess := range list {
		id := stripANSI(sess.ID)
		if s.Pinned[id] {
			pinned = append(pinned, sess)
		} else {
			rest = append(rest, sess)
		}
	}
	return append(pinned, rest...)
}

// BuildProjectItems builds the project list from sessions
func BuildProjectItems(sessions []SessionView) []ProjectItem {
	byName := map[string]*ProjectItem{}
	for _, sess := range sessions {
		if sess.Project == "" {
			continue
		}
		key := strings.ToLower(sess.Project)
		item := byName[key]
		if item == nil {
			item = &ProjectItem{
				Name:        sess.Project,
				StatusCount: make(map[Status]int),
				Providers:   make(map[Provider]int),
			}
			byName[key] = item
		}
		item.Count++
		item.StatusCount[sess.Status]++
		item.Providers[sess.Provider]++
		if sess.LastSeen.After(item.LastSeen) {
			item.LastSeen = sess.LastSeen
		}
	}

	items := make([]ProjectItem, 0, len(byName))
	for _, item := range byName {
		items = append(items, *item)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
		}
		return items[i].Count > items[j].Count
	})

	return items
}

// BuildRows builds table rows from filtered sessions
func (s *AppState) BuildRows(groupBy string, rowBuilder func(*SessionView) table.Row, groupRowBuilder func(string) table.Row) {
	s.RowMeta = nil

	if groupBy == "" {
		for i := range s.FilteredSessions {
			sess := &s.FilteredSessions[i]
			s.RowMeta = append(s.RowMeta, RowMeta{Kind: RowSession, Session: sess})
		}
		return
	}

	groups := groupSessions(s.FilteredSessions, groupBy)
	for _, g := range groups {
		groupLabel := g.Group
		if strings.TrimSpace(groupLabel) == "" {
			groupLabel = "unknown"
		}
		s.RowMeta = append(s.RowMeta, RowMeta{Kind: RowGroup, Group: groupLabel})
		for i := range g.Sessions {
			sess := g.Sessions[i]
			s.RowMeta = append(s.RowMeta, RowMeta{Kind: RowSession, Session: &sess})
		}
	}
}

// SessionGroup represents a group of sessions
type SessionGroup struct {
	Group    string
	Sessions []SessionView
}

// groupSessions groups sessions by the specified key
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

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	// Simple version - the full version uses regexp
	result := strings.Builder{}
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}
