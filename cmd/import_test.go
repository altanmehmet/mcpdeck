package cmd

import (
	"bytes"
	"encoding/json"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"strings"
	"testing"
)

func runImport(t *testing.T, s store.Store, input string, args ...string) (string, error) {
	t.Helper()
	c := NewRoot()
	c.SetArgs(append([]string{"--config", s.Path, "import"}, args...))
	c.SetIn(strings.NewReader(input))
	out := &bytes.Buffer{}
	c.SetOut(out)
	c.SetErr(&bytes.Buffer{})
	err := c.Execute()
	return out.String(), err
}
func TestImportBatchAllAgentsAndDisabled(t *testing.T) {
	s := allProfilesFixture(t)
	source := `{"mcpServers":{"remote":{"serverUrl":"https://example.com/mcp"},"local":{"command":"echo","args":["hello"]},"off":{"command":"echo","disabled":true}}}`
	if _, err := runImport(t, s, source, "--preview"); err != nil {
		t.Fatal(err)
	}
	d, _ := s.Load()
	if len(d.Servers) != 0 {
		t.Fatal("preview mutated store")
	}
	if _, err := runImport(t, s, source); err != nil {
		t.Fatal(err)
	}
	d, _ = s.Load()
	for name, p := range d.Profiles {
		if !d.IsEnabled(name, "remote") || !d.IsEnabled(name, "local") || d.IsEnabled(name, "off") {
			t.Fatal("wrong selections")
		}
		root, servers := readAgentConfig(t, p)
		if root["keep"] != "value" || servers["external"] == nil || servers["remote"] == nil || servers["local"] == nil || servers["off"] != nil {
			t.Fatal("wrong output")
		}
	}
	// Reject an entire batch when one name already exists.
	if _, err := runImport(t, s, `{"local":{"command":"echo"},"fresh":{"command":"echo"}}`); err == nil {
		t.Fatal("duplicate accepted")
	}
	d, _ = s.Load()
	if _, ok := d.Servers["fresh"]; ok {
		t.Fatal("partial batch saved")
	}
}
func TestImportDesktopMissingValues(t *testing.T) {
	s := allProfilesFixture(t)
	t.Setenv("IMPORT_DESKTOP_SECRET", "")
	source := `{"demo":{"url":"https://example.com/mcp","headers":{"Authorization":"Bearer ${IMPORT_DESKTOP_SECRET}"}}}`
	if _, err := runImport(t, s, source); err == nil {
		t.Fatal("missing value accepted")
	}
	d, _ := s.Load()
	if len(d.Servers) != 0 {
		t.Fatal("invalid server saved")
	}
	raw, _ := json.Marshal(map[string]any{"source": source, "values": map[string]string{"IMPORT_DESKTOP_SECRET": "test$literal"}})
	if _, err := runImport(t, s, string(raw), "--json-stdin"); err != nil {
		t.Fatal(err)
	}
	d, _ = s.Load()
	_, servers := readAgentConfig(t, d.Profiles["codex"])
	headers := servers["demo"].(map[string]any)["http_headers"].(map[string]any)
	if headers["Authorization"] != "Bearer test$literal" {
		t.Fatal("secret altered")
	}
}
