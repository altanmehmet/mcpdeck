package syncer

import (
	"encoding/json"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"os"
	"path/filepath"
	"testing"
)

func TestModesPreserveOtherSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ide.json")
	_ = os.WriteFile(path, []byte(`{"theme":"dark","mcpServers":{"external":{"command":"external"},"test":{"command":"old"}}}`), 0600)
	d := &model.Deck{Servers: map[string]model.ServerConfig{"test": {Command: "echo", Args: []string{"${MCP_TEST_VALUE}"}, CachedTools: []byte(`{"tools":[]}`)}}, Profiles: map[string]model.ProfileConfig{"cursor": {TargetPath: path, EnabledServers: []string{"test"}}}}
	t.Setenv("MCP_TEST_VALUE", "hello")
	s := Syncer{Deck: d, Executable: "/app/mcpdeck", ConfigPath: "/deck.json"}
	for _, bridge := range []bool{false, true} {
		var err error
		if bridge {
			err = s.SyncBridgeMode("cursor")
		} else {
			err = s.Sync("cursor")
		}
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := os.ReadFile(path)
		var root struct {
			Theme   string
			Servers map[string]json.RawMessage `json:"mcpServers"`
		}
		if err = json.Unmarshal(raw, &root); err != nil {
			t.Fatal(err)
		}
		if root.Theme != "dark" || root.Servers["external"] == nil {
			t.Fatal("unrelated configuration lost")
		}
		if bridge {
			if root.Servers["mcpdeck"] == nil || root.Servers["test"] != nil {
				t.Fatal("incorrect bridge config")
			}
		} else {
			var cfg model.ServerConfig
			_ = json.Unmarshal(root.Servers["test"], &cfg)
			if cfg.Args[0] != "hello" || len(cfg.CachedTools) != 0 {
				t.Fatal("incorrect direct config")
			}
		}
	}
}
func TestInvalidJSONNotOverwritten(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ide.json")
	_ = os.WriteFile(path, []byte(`invalid`), 0600)
	s := Syncer{Deck: &model.Deck{Profiles: map[string]model.ProfileConfig{"cursor": {TargetPath: path}}}}
	if err := s.Sync("cursor"); err == nil {
		t.Fatal("expected failure")
	}
	b, _ := os.ReadFile(path)
	if string(b) != "invalid" {
		t.Fatal("file changed")
	}
}
