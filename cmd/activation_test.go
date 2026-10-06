package cmd

import (
	"bytes"
	"encoding/json"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"github.com/pelletier/go-toml/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func allProfilesFixture(t *testing.T) store.Store {
	t.Helper()
	dir := t.TempDir()
	s := store.Store{Path: filepath.Join(dir, "deck.json")}
	d := &model.Deck{Version: 1, Servers: map[string]model.ServerConfig{}, Profiles: store.DefaultProfiles()}
	for name, p := range store.AdditionalProfiles() {
		if store.IsInstalledAgent(name, p) {
			d.Profiles[name] = p
		}
	}
	for name, p := range d.Profiles {
		p.TargetPath = filepath.Join(dir, name+".config")
		d.Profiles[name] = p
		field := "mcpServers"
		if p.Format == "vscode" {
			field = "servers"
		}
		if p.Format == "zed" {
			field = "context_servers"
		}
		if p.Format == "opencode" {
			field = "mcp"
		}
		source := `{"keep":"value","` + field + `":{"external":{"command":"keep"}}}`
		if p.Format == "codex" {
			source = "keep = 'value'\n[mcp_servers.external]\ncommand = 'keep'\n"
		}
		if err := os.WriteFile(p.TargetPath, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Save(d); err != nil {
		t.Fatal(err)
	}
	return s
}

func runCommand(s store.Store, args ...string) error {
	c := NewRoot()
	c.SetArgs(append([]string{"--config", s.Path}, args...))
	c.SetOut(&bytes.Buffer{})
	c.SetErr(&bytes.Buffer{})
	return c.Execute()
}

func readAgentConfig(t *testing.T, p model.ProfileConfig) (map[string]any, map[string]any) {
	t.Helper()
	raw, err := os.ReadFile(p.TargetPath)
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]any
	field := "mcpServers"
	if p.Format == "codex" {
		err = toml.Unmarshal(raw, &root)
		field = "mcp_servers"
	} else {
		err = json.Unmarshal(raw, &root)
		if p.Format == "vscode" {
			field = "servers"
		}
		if p.Format == "zed" {
			field = "context_servers"
		}
		if p.Format == "opencode" {
			field = "mcp"
		}
	}
	if err != nil {
		t.Fatal(err)
	}
	return root, root[field].(map[string]any)
}

func TestAddEnableDisableAllAgents(t *testing.T) {
	s := allProfilesFixture(t)
	t.Setenv("MCPDECK_BULK_TOKEN", "test-value")
	if err := runCommand(s, "add", "--name", "oracle-demo", "--command", "echo", "--arg", "hello world", "--env", "AUTH=${MCPDECK_BULK_TOKEN}"); err != nil {
		t.Fatal(err)
	}
	for _, enabled := range []bool{true, false, false, true, true} {
		action := "disable"
		if enabled {
			action = "enable"
		}
		if err := runCommand(s, action, "oracle-demo"); err != nil {
			t.Fatal(err)
		}
		d, err := s.Load()
		if err != nil {
			t.Fatal(err)
		}
		for name, p := range d.Profiles {
			if d.IsEnabled(name, "oracle-demo") != enabled || len(p.EnabledServers) > 1 {
				t.Fatalf("incorrect selection in %s", name)
			}
			root, servers := readAgentConfig(t, p)
			if root["keep"] != "value" || servers["external"].(map[string]any)["command"] != "keep" {
				t.Fatalf("external settings changed in %s", name)
			}
			entry, exists := servers["oracle-demo"]
			if exists != enabled {
				t.Fatalf("incorrect agent configuration in %s", name)
			}
			envField := "env"
			if p.Format == "opencode" {
				envField = "environment"
			}
			if exists && entry.(map[string]any)[envField].(map[string]any)["AUTH"] != "test-value" {
				t.Fatalf("environment not applied in %s", name)
			}
		}
	}
}

func TestAddAllWithoutSeparateSync(t *testing.T) {
	s := allProfilesFixture(t)
	if err := runCommand(s, "add", "--name", "demo", "--command", "echo"); err != nil {
		t.Fatal(err)
	}
	d, _ := s.Load()
	for name, p := range d.Profiles {
		_, servers := readAgentConfig(t, p)
		if !d.IsEnabled(name, "demo") || servers["demo"] == nil {
			t.Fatalf("add did not automatically apply to %s", name)
		}
	}
}

func TestSingleProfileAndDisabledAdd(t *testing.T) {
	s := allProfilesFixture(t)
	if err := runCommand(s, "add", "--name", "demo", "--command", "echo", "--disabled"); err != nil {
		t.Fatal(err)
	}
	if err := runCommand(s, "enable", "demo", "--profile", "codex"); err != nil {
		t.Fatal(err)
	}
	d, _ := s.Load()
	for name, p := range d.Profiles {
		_, servers := readAgentConfig(t, p)
		if d.IsEnabled(name, "demo") != (name == "codex") || (servers["demo"] != nil) != (name == "codex") {
			t.Fatalf("unexpected selection in %s", name)
		}
	}
	before, _ := os.ReadFile(s.Path)
	for _, args := range [][]string{{"enable", "missing"}, {"disable", "demo", "--profile", "missing"}} {
		if err := runCommand(s, args...); err == nil {
			t.Fatal("invalid selection accepted")
		}
		after, _ := os.ReadFile(s.Path)
		if !bytes.Equal(before, after) {
			t.Fatal("failed selection changed deck")
		}
	}
}

func TestBulkSyncFailureCanBeRetried(t *testing.T) {
	s := allProfilesFixture(t)
	d, _ := s.Load()
	p := d.Profiles["cursor"].TargetPath
	if err := os.WriteFile(p, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	err := runCommand(s, "add", "--name", "demo", "--command", "echo")
	if err == nil || !strings.Contains(err.Error(), "server saved; sync incomplete") || !strings.Contains(err.Error(), "cursor") {
		t.Fatalf("partial failure not reported: %v", err)
	}
	raw, _ := os.ReadFile(p)
	if string(raw) != "broken" {
		t.Fatal("invalid target overwritten")
	}
	d, _ = s.Load()
	_, codex := readAgentConfig(t, d.Profiles["codex"])
	if codex["demo"] == nil || !d.IsEnabled("cursor", "demo") {
		t.Fatal("valid profile not synced or selection not saved")
	}
	if err := os.WriteFile(p, []byte(`{"mcpServers":{}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := runCommand(s, "enable", "demo"); err != nil {
		t.Fatal(err)
	}
	_, cursor := readAgentConfig(t, d.Profiles["cursor"])
	if cursor["demo"] == nil {
		t.Fatal("retry did not repair sync")
	}
}

func TestAddNoSync(t *testing.T) {
	s := allProfilesFixture(t)
	if err := runCommand(s, "add", "--name", "demo", "--command", "echo", "--no-sync"); err != nil {
		t.Fatal(err)
	}
	d, _ := s.Load()
	for name, p := range d.Profiles {
		_, servers := readAgentConfig(t, p)
		if !d.IsEnabled(name, "demo") || servers["demo"] != nil {
			t.Fatalf("no-sync did not preserve target %s", name)
		}
	}
}

func TestWizardDefaultsToAllProfiles(t *testing.T) {
	s := allProfilesFixture(t)
	c := NewRoot()
	c.SetArgs([]string{"--config", s.Path, "add"})
	c.SetIn(strings.NewReader("demo\necho\n[\"hello world\"]\n\nKEY=value=with=equals\n\n"))
	c.SetOut(&bytes.Buffer{})
	if err := c.Execute(); err != nil {
		t.Fatal(err)
	}
	d, _ := s.Load()
	for name, p := range d.Profiles {
		_, servers := readAgentConfig(t, p)
		if !d.IsEnabled(name, "demo") || servers["demo"] == nil {
			t.Fatalf("wizard did not apply to %s", name)
		}
	}
	if d.Servers["demo"].Env["KEY"] != "value=with=equals" {
		t.Fatal("environment value changed")
	}
}

func TestBulkBridgeSelection(t *testing.T) {
	s := allProfilesFixture(t)
	if err := s.Update(func(d *model.Deck) error {
		p := d.Profiles["codex"]
		p.Mode = "bridge"
		d.Profiles["codex"] = p
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := runCommand(s, "add", "--name", "demo", "--command", "echo"); err != nil {
		t.Fatal(err)
	}
	d, _ := s.Load()
	_, entries := readAgentConfig(t, d.Profiles["codex"])
	if entries["mcpdeck"] == nil || entries["demo"] != nil || !d.IsEnabled("codex", "demo") {
		t.Fatal("bridge mode not preserved")
	}
	if err := runCommand(s, "disable", "demo"); err != nil {
		t.Fatal(err)
	}
	d, _ = s.Load()
	if d.IsEnabled("codex", "demo") || d.Profiles["codex"].Mode != "bridge" {
		t.Fatal("bridge selection not disabled")
	}
}
