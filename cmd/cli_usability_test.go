package cmd

import (
	"bytes"
	"encoding/json"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"path/filepath"
	"strings"
	"testing"
)

func TestHumanListAndExistingJSONContract(t *testing.T) {
	s := store.Store{Path: filepath.Join(t.TempDir(), "deck.json"), DisableDiscovery: true}
	d := &model.Deck{Version: 1, Servers: map[string]model.ServerConfig{"demo": {Command: "echo", Args: []string{"secret-proof"}, Env: map[string]string{"TOKEN": "secret-proof"}}}, Profiles: map[string]model.ProfileConfig{"cursor": {TargetPath: filepath.Join(t.TempDir(), "mcp.json"), EnabledServers: []string{"demo"}}}}
	if err := s.Save(d); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) (string, error) {
		t.Helper()
		c := NewRoot()
		c.SetArgs(append([]string{"--config", s.Path}, args...))
		var out bytes.Buffer
		c.SetOut(&out)
		c.SetErr(&out)
		err := c.Execute()
		return out.String(), err
	}
	for _, name := range []string{"list", "ls"} {
		out, err := run(name)
		if err != nil || !strings.Contains(out, "SERVER") || !strings.Contains(out, "demo") || strings.Contains(out, "secret-proof") {
			t.Fatal("inventory invalid", out, err)
		}
	}
	human, err := run("agents")
	if err != nil || !strings.Contains(human, "AGENT") || !strings.Contains(human, "cursor") {
		t.Fatal(human, err)
	}
	status, err := run("status")
	if err != nil {
		t.Fatal(err)
	}
	listing, err := run("list", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var before, after any
	if json.Unmarshal([]byte(status), &before) != nil || json.Unmarshal([]byte(listing), &after) != nil || status != listing {
		t.Fatal("status JSON contract changed")
	}
	if _, err = run("list", "--profile", "missing"); err == nil {
		t.Fatal("unknown agent accepted")
	}
}
func TestGroupedHelpAndCompatibilityAliases(t *testing.T) {
	c := NewRoot()
	c.SetArgs([]string{"--help"})
	var out bytes.Buffer
	c.SetOut(&out)
	if err := c.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, section := range []string{"Everyday commands:", "Maintenance:", "Advanced and scripting:", "mcpdeck agents"} {
		if !strings.Contains(out.String(), section) {
			t.Fatal("help missing", section)
		}
	}
	for _, name := range []string{"add", "profiles", "status", "connect", "update", "bridge", "cache-clear"} {
		found, _, err := NewRoot().Find([]string{name})
		if err != nil || found == nil || found.Name() == "mcpdeck" {
			t.Fatal("command missing", name)
		}
	}
}
