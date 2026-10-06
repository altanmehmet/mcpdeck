package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/altanmehmet/mcpdeck/internal/model"
)

func TestRedactedConnectionDoesNotExposePrivateValues(t *testing.T) {
	cfg := model.ServerConfig{
		URL:       "https://user:password@example.test/mcp?token=password",
		Variables: map[string]string{"TOKEN": "password"},
		Headers:   map[string]string{"Authorization": "Bearer password"},
		Env:       map[string]string{"SECRET": "password"},
		Command:   "node",
		Args:      []string{"server.js"},
	}
	raw, err := redactedConnection(cfg, "oracle")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(raw, "password") {
		t.Fatalf("redacted connection leaked a private value: %s", raw)
	}
	var root map[string]any
	if err := json.Unmarshal([]byte(raw), &root); err != nil {
		t.Fatal(err)
	}
}

func TestEnabledTargetsOnlyIncludesEnabledProfiles(t *testing.T) {
	d := &model.Deck{Profiles: map[string]model.ProfileConfig{
		"codex":  {EnabledServers: []string{"oracle"}},
		"cursor": {EnabledServers: []string{}},
	}}
	got := enabledTargets(d, "oracle")
	if len(got) != 1 || got[0] != "codex" {
		t.Fatalf("targets = %#v", got)
	}
}
