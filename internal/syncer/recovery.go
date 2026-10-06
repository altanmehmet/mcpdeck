package syncer

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/altanmehmet/mcpdeck/internal/store"
)

// SyncResult contains no connection settings or diagnostics that may hold secrets.
type SyncResult struct {
	Profile    string `json:"profile"`
	TargetPath string `json:"target_path"`
	Status     string `json:"status"`
}

type syncReport struct {
	Version int                   `json:"version"`
	Targets map[string]SyncResult `json:"targets"`
}

func (s Syncer) reportPath() string { return s.ConfigPath + ".sync-status.json" }

func decodeReport(raw []byte) (syncReport, error) {
	r := syncReport{Version: 1, Targets: map[string]SyncResult{}}
	if len(raw) == 0 {
		return r, nil
	}
	if json.Unmarshal(raw, &r) != nil || r.Version != 1 || r.Targets == nil {
		return r, fmt.Errorf("invalid sync report; run a normal sync after moving the report aside")
	}
	return r, nil
}

func (s Syncer) record(results []SyncResult) error {
	if s.ConfigPath == "" || len(results) == 0 {
		return nil
	}
	return store.UpdateFile(s.reportPath(), "", func(raw []byte, exists bool) ([]byte, error) {
		r, err := decodeReport(raw)
		if err != nil {
			return nil, err
		}
		for _, result := range results {
			r.Targets[result.Profile] = result
		}
		return json.MarshalIndent(r, "", "  ")
	})
}

func (s Syncer) Results() ([]SyncResult, error) {
	if s.ConfigPath == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(s.reportPath())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read sync report")
	}
	r, err := decodeReport(raw)
	if err != nil {
		return nil, err
	}
	results := make([]SyncResult, 0, len(r.Targets))
	for _, result := range r.Targets {
		results = append(results, result)
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Profile < results[j].Profile })
	return results, nil
}

// SyncTargets continues after individual failures and saves a retryable report.
func (s Syncer) SyncTargets(keys []string) ([]SyncResult, error) {
	var results []SyncResult
	var failures []error
	seen := map[string]bool{}
	for _, key := range keys {
		if seen[key] {
			continue
		}
		seen[key] = true
		p, ok := s.Deck.Profiles[key]
		if !ok {
			failures = append(failures, fmt.Errorf("unknown profile %q", key))
			continue
		}
		result := SyncResult{Profile: key, TargetPath: ExpandPath(p.TargetPath), Status: "synced"}
		if err := s.Sync(key); err != nil {
			result.Status = "failed"
			failures = append(failures, fmt.Errorf("%s: %w", key, err))
		}
		results = append(results, result)
	}
	if err := s.record(results); err != nil {
		failures = append(failures, fmt.Errorf("agent operations completed, but sync report could not be saved: %w", err))
	}
	return results, errors.Join(failures...)
}

func (s Syncer) RetryFailed() ([]SyncResult, error) {
	results, err := s.Results()
	if err != nil {
		return nil, err
	}
	var keys []string
	for _, result := range results {
		if result.Status != "failed" {
			continue
		}
		p, ok := s.Deck.Profiles[result.Profile]
		if !ok || ExpandPath(p.TargetPath) != result.TargetPath {
			return nil, fmt.Errorf("failed target %s changed; review it and run a normal sync", result.Profile)
		}
		keys = append(keys, result.Profile)
	}
	return s.SyncTargets(keys)
}

// Restore restores the last changed configuration, retaining the current file
// as a backup. It does not change the deck or reconnect an agent.
func (s Syncer) Restore(key string) error {
	p, ok := s.Deck.Profiles[key]
	if !ok {
		return fmt.Errorf("unknown profile %q", key)
	}
	path := ExpandPath(p.TargetPath)
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Errorf("cannot resolve target; restore requires an existing configuration")
	}
	if err = store.UpdateFile(path, ".mcpdeck-backup", func([]byte, bool) ([]byte, error) {
		backup, err := os.ReadFile(resolved + ".mcpdeck-backup")
		if err != nil {
			return nil, fmt.Errorf("no readable sync backup for %s", key)
		}
		if _, err = decodeConfig(backup, p.Format); err != nil {
			return nil, fmt.Errorf("backup configuration is invalid; refusing restore")
		}
		return backup, nil
	}); err != nil {
		return err
	}
	return s.record([]SyncResult{{Profile: key, TargetPath: path, Status: "restored"}})
}
