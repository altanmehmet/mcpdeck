package client

import (
	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"os"
	"strings"
	"testing"
)

func TestAgentReviewScopesChangesAndBridge(t *testing.T) {
	a, d := fixture(t)
	d.Servers["demo"] = model.ServerConfig{Command: "echo", Args: []string{"ok"}}
	if err := a.s.Save(d); err != nil {
		t.Fatal(err)
	}
	r, err := a.ReviewAgent("cursor", "enable", "demo")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := a.s.Load()
	if before.IsEnabled("cursor", "demo") {
		t.Fatal("review mutated deck")
	}
	if len(r.Results) != 1 || r.Results[0].Agent != "cursor" {
		t.Fatal("review not scoped")
	}
	result, err := a.ApplyReview(r.Token)
	if err != nil || result.Partial {
		t.Fatal(result, err)
	}
	after, _ := a.s.Load()
	if !after.IsEnabled("cursor", "demo") || after.IsEnabled("codex", "demo") {
		t.Fatal("wrong profiles changed")
	}
	r, err = a.ReviewAgent("cursor", "bridge", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = a.ApplyReview(r.Token); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(d.Profiles["cursor"].TargetPath)
	if !strings.Contains(string(raw), `"bridge"`) || !strings.Contains(string(raw), `"mcpdeck"`) {
		t.Fatal("bridge not synced")
	}
	snapshot, err := a.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	for _, agent := range snapshot.Agents {
		if agent.Name == "cursor" && (agent.Mode != "bridge" || agent.SyncStatus != "synced") {
			t.Fatal("status not refreshed", agent)
		}
	}
	if _, err = a.ReviewAgent("unknown", "sync", ""); err == nil {
		t.Fatal("unknown agent accepted")
	}
	if _, err = a.ReviewAgent("cursor", "enable", "unknown"); err == nil {
		t.Fatal("unknown server accepted")
	}
}
func TestRetryFailedOnlyAndRestoreStaleProtection(t *testing.T) {
	a, d := fixture(t)
	target := d.Profiles["cursor"].TargetPath
	if err := store.AtomicWrite(target, []byte("broken")); err != nil {
		t.Fatal(err)
	}
	r, err := a.ReviewAgent("cursor", "sync", "")
	if err != nil {
		t.Fatal(err)
	}
	result, err := a.ApplyReview(r.Token)
	if err != nil || !result.Partial {
		t.Fatal("failure not represented", result, err)
	}
	if err = store.AtomicWrite(target, []byte(`{"keep":"original"}`)); err != nil {
		t.Fatal(err)
	}
	r, err = a.ReviewRecovery("retry", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Results) != 1 || r.Results[0].Agent != "cursor" {
		t.Fatal("retry targets wrong")
	}
	result, err = a.ApplyReview(r.Token)
	if err != nil || result.Partial {
		t.Fatal(result, err)
	}
	r, err = a.ReviewRecovery("restore", "cursor")
	if err != nil {
		t.Fatal(err)
	}
	if err = store.AtomicWrite(target, []byte(`{"keep":"external change"}`)); err != nil {
		t.Fatal(err)
	}
	if _, err = a.ApplyReview(r.Token); err == nil {
		t.Fatal("stale restore overwrote target")
	}
	r, err = a.ReviewRecovery("restore", "cursor")
	if err != nil {
		t.Fatal(err)
	}
	if err = store.AtomicWrite(target+".mcpdeck-backup", []byte(`{"changed":"backup"}`)); err != nil {
		t.Fatal(err)
	}
	if _, err = a.ApplyReview(r.Token); err == nil {
		t.Fatal("stale backup applied")
	}
	r, err = a.ReviewRecovery("restore", "cursor")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = a.ApplyReview(r.Token); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(target)
	if string(raw) != `{"changed":"backup"}` {
		t.Fatal("restore failed")
	}
	raw, _ = os.ReadFile(target + ".mcpdeck-backup")
	if string(raw) != `{"keep":"external change"}` {
		t.Fatal("current config not backed up")
	}
}

func TestDemoNeverDiscoversRealProfiles(t *testing.T) {
	a, cleanup, err := NewDemo()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	defer a.Close()
	check := func() {
		state, err := a.Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		if len(state.Agents) != 4 {
			t.Fatalf("demo discovered %d profiles", len(state.Agents))
		}
		for _, agent := range state.Agents {
			if !strings.HasPrefix(agent.Path, a.manager.Home+string(os.PathSeparator)) {
				t.Fatal("demo target escaped", agent.Name)
			}
		}
	}
	check()
	r, err := a.ReviewAgent("cursor", "enable", "sample-git")
	if err != nil {
		t.Fatal(err)
	}
	result, err := a.ApplyReview(r.Token)
	if err != nil || result.Partial {
		t.Fatal(result, err)
	}
	check()
}
