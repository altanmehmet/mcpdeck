package syncer

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/vault"
)

func TestVaultSyncKeepsReferencesOutOfAgentConfigs(t *testing.T) {
	for _, format := range []string{"", "vscode", "copilot-cli", "codex", "zed", "gemini-cli", "opencode", "cline", "continue", "roo-code", "qwen", "trae"} {
		t.Run(format, func(t *testing.T) {
			reference := vault.Reference("fixture-must-not-resolve")
			d := &model.Deck{Version: 1, Servers: map[string]model.ServerConfig{
				"private": {URL: "https://example.com/mcp", Headers: map[string]string{"Authorization": "Bearer ${TOKEN}"}, Variables: map[string]string{"TOKEN": reference}},
				"public":  {Command: "echo", Args: []string{"safe"}},
			}, Profiles: map[string]model.ProfileConfig{"test": {Format: format, TargetPath: filepath.Join(t.TempDir(), "config"), EnabledServers: []string{"private", "public"}}}}
			s := Syncer{Deck: d, ConfigPath: "/test/deck.json", Executable: "/test/mcpdeck"}
			raw, err := s.render("test", false, nil)
			if err != nil {
				t.Fatal(err)
			}
			for _, excluded := range []string{reference, "Authorization", "TOKEN", "https://example.com/mcp"} {
				if bytes.Contains(raw, []byte(excluded)) {
					t.Fatalf("private connection material reached %s", format)
				}
			}
			if !bytes.Contains(raw, []byte("vault")) || !bytes.Contains(raw, []byte("exec")) || !bytes.Contains(raw, []byte("safe")) {
				t.Fatal("runtime wrapper or public MCP missing")
			}
		})
	}
}
func TestVaultArgumentRejectedBeforeSync(t *testing.T) {
	d := &model.Deck{Servers: map[string]model.ServerConfig{"private": {Command: "echo", Args: []string{"${TOKEN}"}, Variables: map[string]string{"TOKEN": vault.Reference("fixture")}}}, Profiles: map[string]model.ProfileConfig{"test": {EnabledServers: []string{"private"}}}}
	if _, err := (Syncer{Deck: d}).render("test", false, nil); err == nil {
		t.Fatal("vault secret allowed in argv")
	}
}
