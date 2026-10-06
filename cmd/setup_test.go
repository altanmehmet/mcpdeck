package cmd

import (
	"bytes"
	"encoding/json"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"testing"
)

func TestSetupOnlyRequiresSelectedRuntimes(t *testing.T) {
	s := allProfilesFixture(t)
	if err := s.Update(func(d *model.Deck) error {
		d.Servers["unused"] = model.ServerConfig{Command: "unused-program"}
		d.Servers["local"] = model.ServerConfig{Command: "mcpdeck-nonexistent-test-command"}
		d.Servers["remote"] = model.ServerConfig{URL: "https://example.com/mcp", Headers: map[string]string{"Authorization": "secret-not-for-output"}}
		p := d.Profiles["codex"]
		p.EnabledServers = []string{"local", "remote"}
		d.Profiles["codex"] = p
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	before, _ := s.Load()
	out := &bytes.Buffer{}
	c := NewRoot()
	c.SetArgs([]string{"--config", s.Path, "setup"})
	c.SetOut(out)
	if err := c.Execute(); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out.Bytes(), []byte("secret-not-for-output")) {
		t.Fatal("secret leaked")
	}
	var result struct {
		Runtimes []struct {
			Name      string
			Required  bool
			Available bool
		}
		Profiles []struct {
			Name   string
			Exists bool
		}
	}
	if json.Unmarshal(out.Bytes(), &result) != nil {
		t.Fatal("invalid report")
	}
	found := false
	for _, runtime := range result.Runtimes {
		if runtime.Name == "go" || runtime.Name == "unused-program" {
			t.Fatal("unneeded runtime checked")
		}
		if runtime.Name == "npx" && runtime.Required {
			t.Fatal("native HTTP does not need npx")
		}
		if runtime.Name == "mcpdeck-nonexistent-test-command" {
			found = true
			if !runtime.Required || runtime.Available {
				t.Fatal("missing executable not reported")
			}
		}
	}
	if !found || len(result.Profiles) < 8 {
		t.Fatal("incomplete setup report")
	}
	after, _ := s.Load()
	a, _ := json.Marshal(before)
	b, _ := json.Marshal(after)
	if !bytes.Equal(a, b) {
		t.Fatal("setup changed configuration")
	}
}
