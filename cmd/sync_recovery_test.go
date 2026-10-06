package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
)

func TestSyncRestoreRequiresConfirmation(t *testing.T) {
	dir := t.TempDir()
	s := store.Store{Path: filepath.Join(dir, "deck.json")}
	path := filepath.Join(dir, "agent.json")
	if err := s.Save(&model.Deck{Version: 1, Servers: map[string]model.ServerConfig{}, Profiles: map[string]model.ProfileConfig{"agent": {TargetPath: path}}}); err != nil {
		t.Fatal(err)
	}
	for file, text := range map[string]string{path: `{"current":true}`, path + ".mcpdeck-backup": `{"previous":true}`} {
		if err := os.WriteFile(file, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, yes := range []bool{false, true} {
		c := NewRoot()
		args := []string{"--config", s.Path, "sync", "restore", "--profile", "agent"}
		if yes {
			args = append(args, "--yes")
		}
		c.SetArgs(args)
		c.SetOut(&bytes.Buffer{})
		if err := c.Execute(); err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(path)
		want := `{"current":true}`
		if yes {
			want = `{"previous":true}`
		}
		if err != nil || string(raw) != want {
			t.Fatal("confirmation not respected", err)
		}
	}
}
