package cmd

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoteAllAgentLifecycle(t *testing.T) {
	s := allProfilesFixture(t)
	t.Setenv("MCPDECK_REMOTE_SECRET", "Bearer test-secret")
	if err := runCommand(s, "add", "--name", "remote-demo", "--url", "https://example.com/mcp", "--header", "Authorization=${MCPDECK_REMOTE_SECRET}"); err != nil {
		t.Fatal(err)
	}
	for _, enabled := range []bool{true, false, true} {
		action := "disable"
		if enabled {
			action = "enable"
		}
		if err := runCommand(s, action, "remote-demo"); err != nil {
			t.Fatal(err)
		}
		d, _ := s.Load()
		for name, p := range d.Profiles {
			root, servers := readAgentConfig(t, p)
			if root["keep"] != "value" || servers["external"] == nil {
				t.Fatal("unrelated setting changed")
			}
			raw, exists := servers["remote-demo"]
			if exists != enabled {
				t.Fatalf("wrong activation for %s", name)
			}
			if !enabled {
				continue
			}
			entry := raw.(map[string]any)
			if name == "claude" {
				if strings.TrimSuffix(strings.ToLower(filepath.Base(entry["command"].(string))), ".cmd") != "npx" {
					t.Fatal("desktop adapter missing")
				}
				args, _ := json.Marshal(entry["args"])
				if bytes.Contains(args, []byte("test-secret")) {
					t.Fatal("header leaked in arguments")
				}
			} else {
				field := "url"
				if name == "antigravity" || name == "windsurf" {
					field = "serverUrl"
				} else if name == "gemini-cli" || name == "qwen-code" {
					field = "httpUrl"
				}
				if entry[field] != "https://example.com/mcp" || entry["command"] != nil {
					t.Fatalf("wrong remote entry for %s", name)
				}
				header := "headers"
				if name == "codex" {
					header = "http_headers"
				}
				if entry[header].(map[string]any)["Authorization"] != "Bearer test-secret" {
					t.Fatal("missing header")
				}
				if name == "copilot" || name == "copilot-cli" || name == "claude-code" {
					if entry["type"] != "http" {
						t.Fatal("wrong transport")
					}
				}
			}
		}
	}
}

func TestDesktopJSONInputAndRedactedStatus(t *testing.T) {
	s := allProfilesFixture(t)
	c := NewRoot()
	c.SetArgs([]string{"--config", s.Path, "add", "--json-stdin", "--disabled"})
	c.SetIn(strings.NewReader(`{"name":"private-server","server":{"url":"https://example.com/secret-url?token=private-token","headers":{"Authorization":"private-secret"}}}`))
	c.SetOut(&bytes.Buffer{})
	if err := c.Execute(); err != nil {
		t.Fatal(err)
	}
	out := &bytes.Buffer{}
	c = NewRoot()
	c.SetArgs([]string{"--config", s.Path, "status"})
	c.SetOut(out)
	if err := c.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"secret-url", "private-token", "private-secret", "Authorization"} {
		if strings.Contains(out.String(), s) {
			t.Fatal("status leaked connection data")
		}
	}
	if !strings.Contains(out.String(), "private-server") || !json.Valid(out.Bytes()) {
		t.Fatal("invalid desktop status")
	}
}

func TestJSONInputRejectsUnknownAndTrailingData(t *testing.T) {
	for _, input := range []string{`{"name":"x","server":{"url":"https://example.com","typo":true}}`, `{"name":"x","server":{"url":"https://example.com"}} {}`, `{"name":"","server":{"command":"echo"}}`} {
		s := allProfilesFixture(t)
		c := NewRoot()
		c.SetArgs([]string{"--config", s.Path, "add", "--json-stdin"})
		c.SetIn(strings.NewReader(input))
		c.SetOut(&bytes.Buffer{})
		c.SetErr(&bytes.Buffer{})
		if c.Execute() == nil {
			t.Fatal("invalid input accepted")
		}
		d, _ := s.Load()
		if len(d.Servers) != 0 {
			t.Fatal("invalid input persisted")
		}
	}
}
