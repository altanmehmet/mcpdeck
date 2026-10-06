package syncer

import (
	"encoding/json"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestIntegrationFormats(t *testing.T) {
	for _, format := range []string{"", "mcpServers", "vscode", "copilot-cli", "codex", "qwen", "trae"} {
		t.Run(format, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config")
			field := serverField(format)
			source := `{"unrelated":{"large":9007199254740993},"` + field + `":{"external":{"url":"https://example.com"},"demo":{"command":"old"}},"projects":{"/work":{"mcpServers":{"local":{"command":"keep"}}}}}`
			if format == "vscode" {
				source = "// user comment\n" + source[:len(source)-1] + ",}"
			}
			if format == "codex" {
				source = "model = 'existing'\n[projects.'/work']\ntrust_level = 'trusted'\n[mcp_servers.external]\nurl = 'https://example.com'\n[mcp_servers.demo]\ncommand = 'old'\n"
			}
			if err := os.WriteFile(path, []byte(source), 0600); err != nil {
				t.Fatal(err)
			}
			before, err := decodeConfig([]byte(source), format)
			if err != nil {
				t.Fatal(err)
			}
			d := &model.Deck{Servers: map[string]model.ServerConfig{"demo": {Command: "echo", Args: []string{"hello"}, Env: map[string]string{"TEST": "value"}, CachedTools: []byte("hidden")}}, Profiles: map[string]model.ProfileConfig{"agent": {TargetPath: path, Format: format, EnabledServers: []string{"demo"}}}}
			sy := Syncer{Deck: d, ConfigPath: "/deck.json", Executable: "/bin/mcpdeck"}
			for _, bridge := range []bool{false, true, false} {
				var err error
				if bridge {
					err = sy.SyncBridgeMode("agent")
				} else {
					err = sy.Sync("agent")
				}
				if err != nil {
					t.Fatal(err)
				}
				raw, _ := os.ReadFile(path)
				after, err := decodeConfig(raw, format)
				if err != nil {
					t.Fatal(err)
				}
				for k, v := range before {
					if k != field && !reflect.DeepEqual(v, after[k]) {
						t.Fatalf("unrelated setting %s changed", k)
					}
				}
				servers := after[field].(map[string]any)
				if !reflect.DeepEqual(servers["external"], before[field].(map[string]any)["external"]) {
					t.Fatal("external server changed")
				}
				name := "demo"
				if bridge {
					name = "mcpdeck"
					if _, ok := servers["demo"]; ok {
						t.Fatal("direct entry remains")
					}
				} else if _, ok := servers["mcpdeck"]; ok {
					t.Fatal("bridge remains")
				}
				entry := servers[name].(map[string]any)
				if _, ok := entry["cached_tools"]; ok {
					t.Fatal("cache leaked")
				}
				if format == "vscode" && entry["type"] != "stdio" {
					t.Fatal(entry)
				}
				if format == "copilot-cli" {
					if entry["type"] != "local" {
						t.Fatal(entry)
					}
					b, _ := json.Marshal(entry["tools"])
					if string(b) != `["*"]` {
						t.Fatal(string(b))
					}
				}
			}
		})
	}
}
func TestInvalidFormatsPreserved(t *testing.T) {
	for _, tc := range []struct{ format, source string }{{"codex", "[broken"}, {"codex", "mcp_servers = 42"}, {"vscode", `{"servers":[]}`}, {"vscode", `{"servers":{}} /* unclosed`}, {"vscode", `{"servers":{},,}`}, {"copilot-cli", `{"mcpServers":null}`}} {
		t.Run(tc.format+tc.source, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "config")
			_ = os.WriteFile(p, []byte(tc.source), 0600)
			s := Syncer{Deck: &model.Deck{Profiles: map[string]model.ProfileConfig{"test": {TargetPath: p, Format: tc.format}}}}
			if err := s.Sync("test"); err == nil {
				t.Fatal("invalid config accepted")
			}
			b, _ := os.ReadFile(p)
			if string(b) != tc.source {
				t.Fatal("config overwritten")
			}
		})
	}
}
func TestJSONCStringsPreserved(t *testing.T) {
	raw := []byte(`{"servers":{},"url":"https://example.com/*a*/,}","quote":"a\\\"//b",/* comment */"list":[1,2,],}`)
	root, err := decodeConfig(raw, "vscode")
	if err != nil {
		t.Fatal(err)
	}
	if root["url"] != "https://example.com/*a*/,}" {
		t.Fatal(root)
	}
}
