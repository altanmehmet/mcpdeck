package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"github.com/altanmehmet/mcpdeck/internal/vault"
)

func TestVaultProtectionRejectsArgumentsBeforeSaving(t *testing.T) {
	cfg := model.ServerConfig{Command: "echo", Args: []string{"${TOKEN}"}, Variables: map[string]string{"TOKEN": "fixture-never-store"}}
	if err := protectInputs(store.Store{Path: "unused"}, "demo", &cfg); err == nil {
		t.Fatal("unsafe argument accepted")
	}
	if cfg.Variables["TOKEN"] != "fixture-never-store" {
		t.Fatal("credential modified before validation")
	}
}
func TestVaultProtectionCapturesEnvironmentBeforeValidation(t *testing.T) {
	t.Setenv("VAULT_FIXTURE_TOKEN", "fixture-not-for-argv")
	cfg := model.ServerConfig{Command: "echo", Args: []string{"${VAULT_FIXTURE_TOKEN}"}}
	if err := protectInputs(store.Store{Path: "unused"}, "demo", &cfg); err == nil {
		t.Fatal("environment-sourced secret permitted in argv")
	}
}
func TestRepairPromptContainsNoVaultValue(t *testing.T) {
	cfg := model.ServerConfig{Command: "fixture", Env: map[string]string{"TOKEN": "${TOKEN}"}, Variables: map[string]string{"TOKEN": vault.Reference("private-key-id")}}
	raw, err := redactedConnection(cfg, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(raw, "private-key-id") || strings.Contains(raw, "vault://") {
		t.Fatal("vault reference leaked into planner prompt")
	}
	if !strings.Contains(raw, "TOKEN") {
		t.Fatal("planner lost the input name")
	}
}
func TestVaultCLIRejectsUnboundVariableWithoutReadingSecret(t *testing.T) {
	s := store.Store{Path: t.TempDir() + "/deck.json"}
	d := &model.Deck{Version: 1, Servers: map[string]model.ServerConfig{"fixture": {Command: "echo"}}, Profiles: map[string]model.ProfileConfig{}}
	if err := s.Save(d); err != nil {
		t.Fatal(err)
	}
	root := vaultCommand(func() store.Store { return s })
	root.SetContext(context.Background())
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetErr(&output)
	root.SetIn(strings.NewReader("fixture-secret"))
	root.SetArgs([]string{"bind", "fixture", "UNKNOWN", "--stdin"})
	if err := root.Execute(); err == nil {
		t.Fatal("unbound input accepted")
	}
	if strings.Contains(output.String(), "fixture-secret") {
		t.Fatal("secret leaked")
	}
}

func TestVaultProtectionRejectsInsecureTransportBeforeSaving(t *testing.T) {
	cfg := model.ServerConfig{URL: "http://example.com/mcp", Headers: map[string]string{"Authorization": "${TOKEN}"}, Variables: map[string]string{"TOKEN": "fixture-never-store"}}
	if err := protectInputs(store.Store{Path: "unused"}, "demo", &cfg); err == nil {
		t.Fatal("insecure remote transport accepted")
	}
	if cfg.Variables["TOKEN"] != "fixture-never-store" {
		t.Fatal("credential modified before rejecting transport")
	}
}
