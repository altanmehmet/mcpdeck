package syncer

import (
	"bytes"
	"github.com/altanmehmet/mcpdeck/internal/testutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/altanmehmet/mcpdeck/internal/model"
)

func TestRetryFailedDoesNotRewriteSuccessfulTargets(t *testing.T) {
	dir := t.TempDir()
	good, bad := filepath.Join(dir, "good.json"), filepath.Join(dir, "bad.json")
	for path, text := range map[string]string{good: `{"theme":"dark"}`, bad: `invalid`} {
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	d := &model.Deck{Servers: map[string]model.ServerConfig{"demo": {Command: "echo", Env: map[string]string{"TOKEN": "private-fixture"}}}, Profiles: map[string]model.ProfileConfig{
		"good": {TargetPath: good, EnabledServers: []string{"demo"}}, "bad": {TargetPath: bad, EnabledServers: []string{"demo"}},
	}}
	sy := Syncer{Deck: d, ConfigPath: filepath.Join(dir, "deck.json")}
	results, err := sy.SyncTargets([]string{"bad", "good"})
	if err == nil || len(results) != 2 || results[0].Status != "failed" || results[1].Status != "synced" {
		t.Fatal("partial outcome not retained", results, err)
	}
	raw, err := os.ReadFile(sy.reportPath())
	if err != nil || strings.Contains(string(raw), "private-fixture") {
		t.Fatal("report includes credentials", err)
	}
	testutil.AssertPrivateFile(t, sy.reportPath())
	before, err := os.Stat(good)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(bad, []byte(`{"fixed":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	results, err = sy.RetryFailed()
	if err != nil || len(results) != 1 || results[0].Profile != "bad" || results[0].Status != "synced" {
		t.Fatal("retry did not complete failed target", results, err)
	}
	after, err := os.Stat(good)
	if err != nil || !os.SameFile(before, after) {
		t.Fatal("successful target was rewritten", err)
	}
	results, err = sy.RetryFailed()
	if err != nil || len(results) != 0 {
		t.Fatal("completed targets retried", results, err)
	}
}

func TestRetryRefusesChangedTarget(t *testing.T) {
	dir := t.TempDir()
	sy := Syncer{ConfigPath: filepath.Join(dir, "deck.json"), Deck: &model.Deck{Profiles: map[string]model.ProfileConfig{"agent": {TargetPath: filepath.Join(dir, "new.json")}}}}
	if err := sy.record([]SyncResult{{Profile: "agent", TargetPath: filepath.Join(dir, "old.json"), Status: "failed"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := sy.RetryFailed(); err == nil {
		t.Fatal("changed target was retried")
	}
}

func TestRestoreAndUndoKeepDeckSelections(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent.json")
	original := []byte(`{"theme":"personal","mcpServers":{"external":{"command":"external"}}}`)
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	d := &model.Deck{Servers: map[string]model.ServerConfig{"demo": {Command: "echo"}}, Profiles: map[string]model.ProfileConfig{"agent": {TargetPath: path, EnabledServers: []string{"demo"}}}}
	sy := Syncer{Deck: d, ConfigPath: filepath.Join(dir, "deck.json")}
	if _, err := sy.SyncTargets([]string{"agent"}); err != nil {
		t.Fatal(err)
	}
	synced, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range [][]byte{original, synced} {
		if err = sy.Restore("agent"); err != nil {
			t.Fatal(err)
		}
		got, e := os.ReadFile(path)
		if e != nil || !bytes.Equal(got, want) {
			t.Fatal("restore failed", e)
		}
	}
	if !d.IsEnabled("agent", "demo") {
		t.Fatal("restore changed the deck")
	}
	if err = os.WriteFile(path+".mcpdeck-backup", []byte(`invalid`), 0600); err != nil {
		t.Fatal(err)
	}
	if err = sy.Restore("agent"); err == nil {
		t.Fatal("invalid backup accepted")
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, synced) {
		t.Fatal("invalid backup damaged target", err)
	}
}
