package syncer

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/altanmehmet/mcpdeck/internal/model"
)

func TestInstallIntoEmptyAgentConfiguration(t *testing.T) {
	for _, format := range []string{"", "codex", "gemini-cli", "vscode", "opencode", "zed"} {
		for _, empty := range []string{"", " \t\r\n"} {
			t.Run(format+"/"+empty, func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "config")
				if err := os.WriteFile(path, []byte(empty), 0600); err != nil {
					t.Fatal(err)
				}
				d := &model.Deck{Servers: map[string]model.ServerConfig{}, Profiles: map[string]model.ProfileConfig{"agent": {TargetPath: path, Format: format, RemoteFormat: "serverUrl"}}}
				sy := Syncer{Deck: d}
				if err := sy.CheckInstallTarget("agent", "supabase"); err != nil {
					t.Fatal("empty placeholder blocked installation", err)
				}
				if err := ValidateConfig([]byte(empty), format); err != nil {
					t.Fatal("doctor rejects empty placeholder", err)
				}
				d.Servers["supabase"] = model.ServerConfig{URL: "https://example.com/mcp", Transport: "http"}
				profile := d.Profiles["agent"]
				profile.EnabledServers = []string{"supabase"}
				d.Profiles["agent"] = profile
				if err := sy.Sync("agent"); err != nil {
					t.Fatal("empty placeholder blocked sync", err)
				}
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				root, err := decodeConfig(raw, format)
				if err != nil {
					t.Fatal(err)
				}
				servers, err := profileServers(root, format, false)
				if err != nil || servers["supabase"] == nil {
					t.Fatal("remote MCP missing", err)
				}
				backup, err := os.ReadFile(path + ".mcpdeck-backup")
				if err != nil || string(backup) != empty {
					t.Fatal("original placeholder not backed up", err)
				}
			})
		}
	}
}

func TestInvalidNonemptyConfigurationStillBlocksInstallationAndSync(t *testing.T) {
	for _, invalid := range []string{`{"token":"private-fixture",`, `null`, `[]`, `"private-fixture"`} {
		path := filepath.Join(t.TempDir(), "config.json")
		raw := []byte(invalid)
		if err := os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
		d := &model.Deck{Servers: map[string]model.ServerConfig{"supabase": {URL: "https://example.com/mcp"}}, Profiles: map[string]model.ProfileConfig{"antigravity": {TargetPath: path, EnabledServers: []string{"supabase"}, RemoteFormat: "serverUrl"}}}
		sy := Syncer{Deck: d}
		if err := sy.CheckInstallTarget("antigravity", "supabase"); err == nil {
			t.Fatal("invalid file accepted")
		}
		if err := sy.Sync("antigravity"); err == nil {
			t.Fatal("invalid file overwritten")
		}
		after, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(after, raw) {
			t.Fatal("invalid file changed", err)
		}
		if _, err := os.Stat(path + ".mcpdeck-backup"); !os.IsNotExist(err) {
			t.Fatal("backup created for rejected write")
		}
	}
}
