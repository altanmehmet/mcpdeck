package tui

import (
	"encoding/json"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
	tea "github.com/charmbracelet/bubbletea"
	"os"
	"path/filepath"
	"testing"
)

func TestToggleAndModeSaveAndSync(t *testing.T) {
	dir := t.TempDir()
	s := store.Store{Path: filepath.Join(dir, "deck.json")}
	d := &model.Deck{Version: 1, Servers: map[string]model.ServerConfig{"demo": {Command: "echo"}}, Profiles: map[string]model.ProfileConfig{"cursor": {TargetPath: filepath.Join(dir, "ide.json")}}}
	m := New(d, s)
	if err := s.Save(d); err != nil {
		t.Fatal(err)
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m = next.(Model)
	saved, err := s.Load()
	if err != nil || !saved.IsEnabled("cursor", "demo") {
		t.Fatal("toggle not saved", err)
	}
	if _, err = os.Stat(d.Profiles["cursor"].TargetPath); err != nil {
		t.Fatal("not synced", err)
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	m = next.(Model)
	if m.deck.Profiles["cursor"].Mode != "bridge" {
		t.Fatal("mode not switched")
	}
	saved, err = s.Load()
	if err != nil || saved.Profiles["cursor"].Mode != "bridge" {
		t.Fatal("mode not persisted", err)
	}
}

func TestEnableDisableAllProfiles(t *testing.T) {
	dir := t.TempDir()
	s := store.Store{Path: filepath.Join(dir, "deck.json")}
	d := &model.Deck{Version: 1, Servers: map[string]model.ServerConfig{"demo": {Command: "echo"}}, Profiles: map[string]model.ProfileConfig{
		"cursor": {TargetPath: filepath.Join(dir, "cursor.json"), EnabledServers: []string{"demo"}},
		"claude": {TargetPath: filepath.Join(dir, "claude.json")},
	}}
	// Keep discovered agents inside the fixture; never write personal settings.
	for name, p := range store.AdditionalProfiles() {
		if store.IsInstalledAgent(name, p) {
			p.TargetPath = filepath.Join(dir, name+".json")
			d.Profiles[name] = p
		}
	}
	if err := s.Save(d); err != nil {
		t.Fatal(err)
	}
	m := New(d, s)
	for _, key := range []rune{'e', 'e', 'd', 'd', 'e'} {
		next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}})
		m = next.(Model)
		saved, err := s.Load()
		if err != nil {
			t.Fatal(err)
		}
		for name, p := range saved.Profiles {
			if saved.IsEnabled(name, "demo") != (key == 'e') || len(p.EnabledServers) > 1 {
				t.Fatalf("incorrect global selection in %s", name)
			}
			raw, err := os.ReadFile(p.TargetPath)
			if err != nil {
				t.Fatal(err)
			}
			var root map[string]map[string]any
			if err := json.Unmarshal(raw, &root); err != nil {
				t.Fatal(err)
			}
			field := "mcpServers"
			switch p.Format {
			case "opencode":
				field = "mcp"
			case "zed":
				field = "context_servers"
			case "vscode":
				field = "servers"
			}
			if (root[field]["demo"] != nil) != (key == 'e') {
				t.Fatalf("global selection not synced to %s", name)
			}
		}
	}
}
