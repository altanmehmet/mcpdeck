package syncer

import (
	"github.com/altanmehmet/mcpdeck/internal/model"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoteFallbackAndSSE(t *testing.T) {
	cfg := model.ServerConfig{URL: "https://example.com/sse", Transport: "sse"}
	for _, p := range []model.ProfileConfig{{}, {Format: "mcpServers"}, {Format: "codex"}, {RemoteFormat: "stdio"}} {
		entry := profileServerEntry(cfg, p)
		if strings.TrimSuffix(strings.ToLower(filepath.Base(entry["command"].(string))), ".cmd") != "npx" {
			t.Fatal("missing local adapter")
		}
		args := entry["args"].([]string)
		found := false
		for _, arg := range args {
			if arg == "sse-only" {
				found = true
			}
		}
		if !found {
			t.Fatal("wrong adapter transport")
		}
	}
	for _, format := range []string{"vscode", "copilot-cli"} {
		if profileServerEntry(cfg, model.ProfileConfig{Format: format})["type"] != "sse" {
			t.Fatal("wrong native transport")
		}
	}
}

func TestQwenAndTraeRemoteSchemas(t *testing.T) {
	cfg := model.ServerConfig{URL: "https://example.com/mcp", Transport: "http", Headers: map[string]string{"X-Token": "${TOKEN}"}}
	qwen := profileServerEntry(cfg, model.ProfileConfig{Format: "qwen", RemoteFormat: "url"})
	if qwen["httpUrl"] != cfg.URL || qwen["url"] != nil || qwen["type"] != nil {
		t.Fatalf("unexpected Qwen remote entry: %#v", qwen)
	}
	trae := profileServerEntry(cfg, model.ProfileConfig{Format: "trae", RemoteFormat: "url"})
	if trae["url"] != cfg.URL || trae["httpUrl"] != nil || trae["type"] != nil {
		t.Fatalf("unexpected Trae remote entry: %#v", trae)
	}
}
