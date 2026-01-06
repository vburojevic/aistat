package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func ensureAppDirs() error {
	ad, err := appDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(ad, 0o700); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(ad, "sessions"), 0o700); err != nil {
		return err
	}
	return nil
}

func appDir() (string, error) {
	if v := os.Getenv("AISTAT_HOME"); strings.TrimSpace(v) != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	// macOS conventional location
	return filepath.Join(home, "Library", "Application Support", appName), nil
}

func sessionsDir() (string, error) {
	ad, err := appDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(ad, "sessions"), nil
}

func configFilePath() (string, error) {
	ad, err := appDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(ad, "config.json"), nil
}

func recordPath(provider Provider, id string) (string, error) {
	if err := ensureAppDirs(); err != nil {
		return "", err
	}
	sd, err := sessionsDir()
	if err != nil {
		return "", err
	}
	safeID := fileSafeRe.ReplaceAllString(id, "_")
	return filepath.Join(sd, fmt.Sprintf("%s_%s.json", provider, safeID)), nil
}

func withLock(lockPath string, fn func() error) error {
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	// Exclusive lock
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }()

	return fn()
}

func loadRecord(p string) (SessionRecord, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return SessionRecord{}, err
	}
	var rec SessionRecord
	if err := json.Unmarshal(b, &rec); err != nil {
		return SessionRecord{}, err
	}
	return rec, nil
}

func saveRecord(p string, rec SessionRecord) error {
	tmp := p + ".tmp"
	b, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func updateRecord(provider Provider, id string, mutate func(*SessionRecord)) error {
	p, err := recordPath(provider, id)
	if err != nil {
		return err
	}
	lock := p + ".lock"

	return withLock(lock, func() error {
		rec := SessionRecord{Provider: provider, ID: id}
		if existing, err := loadRecord(p); err == nil {
			rec = existing
		}
		mutate(&rec)
		rec.Provider = provider
		rec.ID = id
		rec.UpdatedAt = time.Now().UTC()
		return saveRecord(p, rec)
	})
}

func loadAllRecords() ([]SessionRecord, error) {
	sd, err := sessionsDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(sd)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []SessionRecord{}, nil
		}
		return nil, err
	}
	var out []SessionRecord
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		p := filepath.Join(sd, e.Name())
		rec, err := loadRecord(p)
		if err != nil {
			continue
		}
		out = append(out, rec)
	}
	return out, nil
}
