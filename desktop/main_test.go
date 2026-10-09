package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// Run against the packaged executable to ensure an IDE's bridge invocation
// reaches JSON-RPC instead of launching another desktop window.
func TestPackagedBinaryBridge(t *testing.T) {
	binary := os.Getenv("MCPDECK_TEST_BINARY")
	if binary == "" {
		t.Skip("set MCPDECK_TEST_BINARY to the built desktop executable")
	}
	binary, err := filepath.Abs(binary)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "deck.json")
	raw, _ := json.Marshal(map[string]any{"version": 1, "servers": map[string]any{}, "profiles": map[string]any{"desktop-smoke": map[string]any{"target_path": filepath.Join(dir, "agent.json"), "enabled_servers": []string{}}}})
	if err = os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, "bridge", "--config", path, "--profile", "desktop-smoke")
	cmd.Stdin = bytes.NewBufferString("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{\"protocolVersion\":\"2025-11-25\"}}\n{\"jsonrpc\":\"2.0\",\"method\":\"notifications/initialized\"}\n{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/list\"}\n")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("desktop binary did not serve bridge: %v", err)
	}
	lines := bytes.Split(bytes.TrimSpace(out), []byte("\n"))
	if len(lines) != 2 {
		t.Fatalf("stdout must contain exactly two JSON-RPC responses: %d", len(lines))
	}
	for _, line := range lines {
		var r struct {
			JSONRPC string          `json:"jsonrpc"`
			Result  json.RawMessage `json:"result"`
			Error   any             `json:"error"`
		}
		if json.Unmarshal(line, &r) != nil || r.JSONRPC != "2.0" || r.Error != nil || r.Result == nil {
			t.Fatalf("invalid bridge response: %s", line)
		}
	}
}

func TestDesktopPathKeepsPrecedenceAndSkipsInvalidCandidates(t *testing.T) {
	original := t.TempDir()
	extra := t.TempDir()
	missing := filepath.Join(t.TempDir(), "missing")
	file := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(file, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	got := desktopPath(original, []string{extra, original, extra, missing, file, "relative"})
	want := original + string(os.PathListSeparator) + extra
	if got != want {
		t.Fatalf("PATH precedence or filtering changed: %q", got)
	}
}
