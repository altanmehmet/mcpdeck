package cmd

import (
	"bytes"
	"encoding/json"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddAndSyncCommands(t *testing.T) {
	dir := t.TempDir()
	s := store.Store{Path: filepath.Join(dir, "deck.json")}
	d := &model.Deck{Version: 1, Servers: map[string]model.ServerConfig{}, Profiles: map[string]model.ProfileConfig{"cursor": {TargetPath: filepath.Join(dir, "ide.json"), EnabledServers: []string{}}}}
	if err := s.Save(d); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) error {
		c := NewRoot()
		c.SetArgs(append([]string{"--config", s.Path}, args...))
		c.SetOut(&bytes.Buffer{})
		c.SetErr(&bytes.Buffer{})
		return c.Execute()
	}
	if err := run("add", "--name", "demo", "--command", "echo", "--arg", "hello", "--profile", "cursor"); err != nil {
		t.Fatal(err)
	}
	if err := run("add", "--name", "demo", "--command", "bad"); err == nil {
		t.Fatal("duplicate overwritten")
	}
	if err := run("sync", "--profile", "cursor", "--mode", "bridge"); err != nil {
		t.Fatal(err)
	}
	d, err := s.Load()
	if err != nil || d.Profiles["cursor"].Mode != "bridge" || !d.IsEnabled("cursor", "demo") {
		t.Fatal("mode or enable not saved", err)
	}
	raw, err := os.ReadFile(d.Profiles["cursor"].TargetPath)
	if err != nil {
		t.Fatal(err)
	}
	var root struct {
		Servers map[string]model.ServerConfig `json:"mcpServers"`
	}
	if err = json.Unmarshal(raw, &root); err != nil {
		t.Fatal(err)
	}
	if len(root.Servers) != 1 || root.Servers["mcpdeck"].Command == "" {
		t.Fatal("bridge entry missing")
	}
	if err := run("sync", "--profile", "cursor", "--mode", "direct"); err != nil {
		t.Fatal(err)
	}
	if err := run("bridge", "--request-timeout", "0s"); err == nil {
		t.Fatal("invalid timeout accepted")
	}
}
func TestWizard(t *testing.T) {
	s := store.Store{Path: filepath.Join(t.TempDir(), "deck.json")}
	d := &model.Deck{Version: 1, Servers: map[string]model.ServerConfig{}, Profiles: map[string]model.ProfileConfig{"cursor": {TargetPath: filepath.Join(t.TempDir(), "ide.json")}}}
	if err := s.Save(d); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MCPDECK_WIZARD_TOKEN", "test-value")
	c := NewRoot()
	c.SetArgs([]string{"--config", s.Path, "add"})
	c.SetIn(bytes.NewBufferString("demo\necho\n[\"hello world\"]\ncursor\nAUTH=${MCPDECK_WIZARD_TOKEN}\n\n"))
	c.SetOut(&bytes.Buffer{})
	if err := c.Execute(); err != nil {
		t.Fatal(err)
	}
	d, err := s.Load()
	if err != nil || d.Servers["demo"].Args[0] != "hello world" {
		t.Fatal("wizard failed", err)
	}
	if d.Servers["demo"].Env["AUTH"] != "${MCPDECK_WIZARD_TOKEN}" {
		t.Fatal("environment reference not preserved")
	}
	raw, err := os.ReadFile(d.Profiles["cursor"].TargetPath)
	if err != nil || !strings.Contains(string(raw), "test-value") {
		t.Fatal("wizard did not sync resolved environment", err)
	}
}

func TestAddDefaultProfilesPreservesExisting(t *testing.T) {
	s := store.Store{Path: filepath.Join(t.TempDir(), "deck.json")}
	d := &model.Deck{Version: 1, Servers: map[string]model.ServerConfig{}, Profiles: map[string]model.ProfileConfig{"cursor": {TargetPath: "custom.json", Mode: "bridge"}, "custom": {TargetPath: "another.json"}}}
	if err := s.Save(d); err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 2; n++ {
		c := NewRoot()
		c.SetArgs([]string{"--config", s.Path, "profiles", "add-defaults"})
		c.SetOut(&bytes.Buffer{})
		if err := c.Execute(); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Profiles["cursor"].TargetPath != "custom.json" || got.Profiles["cursor"].Mode != "bridge" || got.Profiles["custom"].TargetPath != "another.json" {
		t.Fatal("existing profiles changed")
	}
	if len(got.Profiles) < 9 || got.Profiles["codex"].Format != "codex" || got.Profiles["copilot"].Format != "vscode" {
		t.Fatal("defaults missing")
	}
}
func TestDoctorTOMLAndJSONC(t *testing.T) {
	for _, tc := range []struct{ format, source string }{{"codex", "model = 'existing'\n[mcp_servers.demo]\ncommand='echo'\n"}, {"vscode", "// comment\n{\"servers\":{},}"}} {
		p := filepath.Join(t.TempDir(), "config")
		_ = os.WriteFile(p, []byte(tc.source), 0600)
		if err := checkProfilePath(p, tc.format); err != nil {
			t.Fatal(err)
		}
	}
}
