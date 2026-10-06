package store

import (
	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/testutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestCopilotCLIUsesCopilotHomeOverride(t *testing.T) {
	override := filepath.Join(t.TempDir(), "copilot-home")
	t.Setenv("COPILOT_HOME", override)
	profile, ok := DefaultProfiles()["copilot-cli"]
	if !ok {
		t.Fatal("copilot-cli profile missing")
	}
	want := filepath.Join(override, "mcp-config.json")
	if profile.TargetPath != want {
		t.Fatalf("copilot-cli target = %q, want %q", profile.TargetPath, want)
	}
}

func TestCopilotCLIUsesDefaultHomeWhenOverrideBlank(t *testing.T) {
	t.Setenv("COPILOT_HOME", "")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	profile := DefaultProfiles()["copilot-cli"]
	want := filepath.Join(home, ".copilot", "mcp-config.json")
	if profile.TargetPath != want {
		t.Fatalf("copilot-cli target = %q, want %q", profile.TargetPath, want)
	}
}

func TestLoadMigratesLegacyCopilotCLIPathWhenHomeOverrideIsSet(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	legacy := filepath.Join(home, ".copilot", "mcp-config.json")
	override := filepath.Join(t.TempDir(), "copilot-home")
	t.Setenv("COPILOT_HOME", override)
	path := filepath.Join(t.TempDir(), "deck.json")
	d := &model.Deck{Version: 1, Servers: map[string]model.ServerConfig{}, Profiles: map[string]model.ProfileConfig{
		"copilot-cli": {TargetPath: legacy, EnabledServers: []string{}, Format: "copilot-cli"},
	}}
	s := Store{Path: path}
	if err := s.Save(d); err != nil {
		t.Fatal(err)
	}
	loaded, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(override, "mcp-config.json")
	if loaded.Profiles["copilot-cli"].TargetPath != want {
		t.Fatalf("migrated target = %q, want %q", loaded.Profiles["copilot-cli"].TargetPath, want)
	}
}

func TestLoadPreservesCustomCopilotCLIPath(t *testing.T) {
	t.Setenv("COPILOT_HOME", filepath.Join(t.TempDir(), "copilot-home"))
	custom := filepath.Join(t.TempDir(), "custom-mcp-config.json")
	s := Store{Path: filepath.Join(t.TempDir(), "deck.json")}
	d := &model.Deck{Version: 1, Servers: map[string]model.ServerConfig{}, Profiles: map[string]model.ProfileConfig{
		"copilot-cli": {TargetPath: custom, EnabledServers: []string{}, Format: "copilot-cli"},
	}}
	if err := s.Save(d); err != nil {
		t.Fatal(err)
	}
	loaded, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Profiles["copilot-cli"].TargetPath != custom {
		t.Fatal("custom Copilot CLI target was changed")
	}
}

func TestLoadRepairsWorldReadableDeckPermissions(t *testing.T) {
	s := Store{Path: filepath.Join(t.TempDir(), "deck.json")}
	d := &model.Deck{Version: 1, Servers: map[string]model.ServerConfig{}, Profiles: map[string]model.ProfileConfig{}}
	if err := s.Save(d); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(s.Path, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load(); err != nil {
		t.Fatal(err)
	}
	testutil.AssertPrivateFile(t, s.Path)
}

func TestAdditionalProfilesIncludesQwenAndOptionalTraeProject(t *testing.T) {
	profiles := AdditionalProfiles()
	qwen, ok := profiles["qwen-code"]
	if !ok || qwen.Format != "qwen" || !strings.HasSuffix(qwen.TargetPath, filepath.Join(".qwen", "settings.json")) {
		t.Fatalf("Qwen profile missing or incorrect: %#v", qwen)
	}
	project := t.TempDir()
	if err := os.Mkdir(filepath.Join(project, ".trae"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TRAE_MCP_CONFIG", filepath.Join(project, ".trae", "mcp.json"))
	trae, ok := AdditionalProfiles()["trae"]
	if !ok || trae.Format != "trae" || trae.TargetPath != filepath.Join(project, ".trae", "mcp.json") {
		t.Fatalf("Trae profile missing or incorrect: %#v", trae)
	}
}

func TestAtomicPermissionsAndUpdates(t *testing.T) {
	s := Store{Path: filepath.Join(t.TempDir(), "deck.json")}
	d, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Save(d); err != nil {
		t.Fatal(err)
	}
	testutil.AssertPrivateFile(t, s.Path)
	var wg sync.WaitGroup
	for _, key := range []string{"cursor", "claude"} {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			if e := s.Update(func(d *model.Deck) error { d.Toggle(key, "fetch"); return nil }); e != nil {
				t.Error(e)
			}
		}(key)
	}
	wg.Wait()
	d, err = s.Load()
	if err != nil || !d.IsEnabled("cursor", "fetch") || !d.IsEnabled("claude", "fetch") {
		t.Fatal("lost update", err)
	}
}
func TestInvalidConfigPreserved(t *testing.T) {
	s := Store{Path: filepath.Join(t.TempDir(), "deck.json")}
	_ = os.WriteFile(s.Path, []byte(`broken`), 0600)
	if err := s.Save(Defaults()); err == nil {
		t.Fatal("overwrote corrupt config")
	}
}
