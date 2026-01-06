package app

import (
	"errors"
	"strings"
)

func resolveSourcePath(providerFilter string, id string) (string, Provider, error) {
	if strings.TrimSpace(id) == "" {
		return "", "", errors.New("missing session id")
	}
	providerFilter = strings.TrimSpace(strings.ToLower(providerFilter))
	if providerFilter != "" && providerFilter != string(ProviderClaude) && providerFilter != string(ProviderCodex) {
		return "", "", errors.New("invalid provider (use claude or codex)")
	}

	recs, err := loadAllRecords()
	if err != nil {
		return "", "", err
	}

	for _, r := range recs {
		if providerFilter != "" && string(r.Provider) != providerFilter {
			continue
		}
		if !matchID(r.ID, id) {
			continue
		}
		p := r.TranscriptPath
		if r.Provider == ProviderCodex {
			p = r.RolloutPath
		}
		if p == "" {
			continue
		}
		return p, r.Provider, nil
	}

	return "", "", errors.New("could not resolve source path (try --redact=false)")
}

func matchID(actual, query string) bool {
	if actual == query {
		return true
	}
	if strings.HasPrefix(actual, query) {
		return true
	}
	redacted := redactIDIfNeeded(actual, true)
	if redacted == query {
		return true
	}
	if strings.HasPrefix(redacted, query) {
		return true
	}
	return false
}
