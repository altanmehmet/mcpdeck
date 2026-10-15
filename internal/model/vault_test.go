package model

import (
	"github.com/altanmehmet/mcpdeck/internal/vault"
	"testing"
)

func TestVaultTransport(t *testing.T) {
	for _, url := range []string{"", "https://example.com/mcp", "http://localhost:3000/mcp", "http://127.0.0.1:3000/mcp", "http://[::1]/mcp"} {
		if err := (ServerConfig{URL: url}).CheckVaultTransport(); err != nil {
			t.Fatalf("valid vault transport rejected: %s", url)
		}
	}
	for _, url := range []string{"http://example.com/mcp", "http://localhost.example.com/mcp", "http://127.0.0.1.example.com/mcp", "http://192.168.1.1/mcp", "http://0.0.0.0/mcp", "http://${HOST}/mcp"} {
		cfg := ServerConfig{URL: url, Headers: map[string]string{"Authorization": "${TOKEN}"}, Variables: map[string]string{"TOKEN": vault.Reference("fixture-must-not-read")}}
		if cfg.CheckVaultTransport() == nil {
			t.Fatalf("insecure transport accepted: %s", url)
		}
		if _, err := Resolve(cfg); err == nil {
			t.Fatal("runtime accepted insecure transport")
		}
	}
	// Existing non-vault configurations retain their connection contract.
	cfg := ServerConfig{URL: "http://example.com/mcp", Headers: map[string]string{"Authorization": "fixture-legacy"}}
	if _, err := Resolve(cfg); err != nil {
		t.Fatal("legacy transport behavior changed")
	}
}
func TestVaultInputCannotAppearInProcessArguments(t *testing.T) {
	for _, value := range []string{"${TOKEN}", "$TOKEN", "--token=${TOKEN}", "$TOKEN/suffix"} {
		cfg := ServerConfig{Command: "fixture", Args: []string{value}, Variables: map[string]string{"TOKEN": vault.Reference("fixture")}}
		if cfg.CheckVaultInput("TOKEN") == nil {
			t.Fatal("vault input permitted in argv")
		}
	}
	if err := (ServerConfig{Command: "fixture", Args: []string{"${TOKEN_OTHER}", "$TOKEN_OTHER"}}).CheckVaultInput("TOKEN"); err != nil {
		t.Fatal("distinct variable incorrectly blocked")
	}
}
