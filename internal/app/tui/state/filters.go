package state

import (
	"sort"
	"strings"
)

// FilterState holds all filter settings
type FilterState struct {
	ProviderFilter map[Provider]bool
	StatusFilter   map[Status]bool
	ProjectFilter  map[string]bool
	TextQuery      string
	QueryMode      string // "all", "project", "status"
}

// NewFilterState creates a new FilterState with initialized maps
func NewFilterState() *FilterState {
	return &FilterState{
		ProviderFilter: make(map[Provider]bool),
		StatusFilter:   make(map[Status]bool),
		ProjectFilter:  make(map[string]bool),
		QueryMode:      "all",
	}
}

// ToggleProvider toggles a provider filter
func (f *FilterState) ToggleProvider(p Provider) {
	if f.ProviderFilter[p] {
		delete(f.ProviderFilter, p)
	} else {
		f.ProviderFilter[p] = true
	}
}

// ToggleStatus toggles a status filter
func (f *FilterState) ToggleStatus(s Status) {
	if f.StatusFilter[s] {
		delete(f.StatusFilter, s)
	} else {
		f.StatusFilter[s] = true
	}
}

// ToggleProject toggles a project filter
func (f *FilterState) ToggleProject(name string) {
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" {
		return
	}
	if f.ProjectFilter[key] {
		delete(f.ProjectFilter, key)
	} else {
		f.ProjectFilter[key] = true
	}
}

// SetProject sets a single project filter (clears others)
func (f *FilterState) SetProject(name string) {
	f.ProjectFilter = map[string]bool{
		strings.ToLower(strings.TrimSpace(name)): true,
	}
}

// Clear resets all filters
func (f *FilterState) Clear() {
	f.ProviderFilter = make(map[Provider]bool)
	f.StatusFilter = make(map[Status]bool)
	f.ProjectFilter = make(map[string]bool)
	f.TextQuery = ""
	f.QueryMode = "all"
}

// IsActive returns true if any filter is active
func (f *FilterState) IsActive() bool {
	return len(f.ProviderFilter) > 0 ||
		len(f.StatusFilter) > 0 ||
		len(f.ProjectFilter) > 0 ||
		strings.TrimSpace(f.TextQuery) != ""
}

// MatchesProvider checks if a provider passes the filter
func (f *FilterState) MatchesProvider(p Provider) bool {
	if len(f.ProviderFilter) == 0 {
		return true
	}
	return f.ProviderFilter[p]
}

// MatchesStatus checks if a status passes the filter
func (f *FilterState) MatchesStatus(s Status) bool {
	if len(f.StatusFilter) == 0 {
		return true
	}
	return f.StatusFilter[s]
}

// MatchesProject checks if a project passes the filter
func (f *FilterState) MatchesProject(project string) bool {
	if len(f.ProjectFilter) == 0 {
		return true
	}
	return f.ProjectFilter[strings.ToLower(project)]
}

// MatchesQuery checks if a session matches the text query
func (f *FilterState) MatchesQuery(s SessionView) bool {
	q := strings.TrimSpace(f.TextQuery)
	if q == "" {
		return true
	}

	needle := strings.ToLower(q)
	var hay string

	switch f.QueryMode {
	case "project":
		hay = strings.ToLower(s.Project)
	case "status":
		hay = strings.ToLower(string(s.Status))
	default:
		hay = strings.ToLower(strings.Join([]string{
			string(s.Provider), s.ID, s.Project, s.Dir, s.Model,
		}, " "))
	}

	return fuzzyMatch(needle, hay)
}

// ParseQueryMode parses the query string and extracts the mode
func (f *FilterState) ParseQueryMode() {
	q := strings.TrimSpace(f.TextQuery)
	if strings.HasPrefix(strings.ToLower(q), "p:") {
		f.QueryMode = "project"
		f.TextQuery = strings.TrimSpace(q[2:])
	} else if strings.HasPrefix(strings.ToLower(q), "s:") {
		f.QueryMode = "status"
		f.TextQuery = strings.TrimSpace(q[2:])
	} else {
		f.QueryMode = "all"
	}
}

// ApplyToSessions filters sessions based on all active filters
func (f *FilterState) ApplyToSessions(sessions []SessionView) []SessionView {
	filtered := make([]SessionView, 0, len(sessions))
	for _, s := range sessions {
		if !f.MatchesProvider(s.Provider) {
			continue
		}
		if !f.MatchesStatus(s.Status) {
			continue
		}
		if !f.MatchesProject(s.Project) {
			continue
		}
		if !f.MatchesQuery(s) {
			continue
		}
		filtered = append(filtered, s)
	}
	return filtered
}

// ActiveProviders returns the list of active provider filters
func (f *FilterState) ActiveProviders() []Provider {
	providers := make([]Provider, 0, len(f.ProviderFilter))
	for p := range f.ProviderFilter {
		providers = append(providers, p)
	}
	return providers
}

// ActiveStatuses returns the list of active status filters
func (f *FilterState) ActiveStatuses() []Status {
	statuses := make([]Status, 0, len(f.StatusFilter))
	for s := range f.StatusFilter {
		statuses = append(statuses, s)
	}
	return statuses
}

// ActiveProjects returns the list of active project filters
func (f *FilterState) ActiveProjects() []string {
	projects := make([]string, 0, len(f.ProjectFilter))
	for p := range f.ProjectFilter {
		projects = append(projects, p)
	}
	sort.Strings(projects)
	return projects
}

// fuzzyMatch performs a simple fuzzy match
func fuzzyMatch(needle, hay string) bool {
	if needle == "" {
		return true
	}
	n := []rune(needle)
	h := []rune(hay)
	idx := 0
	for _, r := range h {
		if r == n[idx] {
			idx++
			if idx == len(n) {
				return true
			}
		}
	}
	return false
}

// FilterProjectItems filters project items by a query
func FilterProjectItems(items []ProjectItem, query string) []ProjectItem {
	q := strings.TrimSpace(strings.ToLower(query))
	if q == "" {
		return items
	}
	var out []ProjectItem
	for _, it := range items {
		if fuzzyMatch(q, strings.ToLower(it.Name)) {
			out = append(out, it)
		}
	}
	return out
}

// FilterDashboardItems filters project items for the dashboard (excludes ended/stale)
func FilterDashboardItems(sessions []SessionView, filters *FilterState) []ProjectItem {
	byName := map[string]*ProjectItem{}
	for _, s := range sessions {
		if s.Project == "" {
			continue
		}
		if !filters.MatchesProvider(s.Provider) {
			continue
		}
		if !filters.MatchesStatus(s.Status) {
			continue
		}
		// Dashboard excludes ended/stale
		if s.Status == StatusEnded || s.Status == StatusStale {
			continue
		}
		key := strings.ToLower(s.Project)
		item := byName[key]
		if item == nil {
			item = &ProjectItem{
				Name:        s.Project,
				StatusCount: make(map[Status]int),
				Providers:   make(map[Provider]int),
			}
			byName[key] = item
		}
		item.Count++
		item.StatusCount[s.Status]++
		item.Providers[s.Provider]++
		if s.LastSeen.After(item.LastSeen) {
			item.LastSeen = s.LastSeen
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
