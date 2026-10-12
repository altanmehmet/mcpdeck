package cmd

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"github.com/altanmehmet/mcpdeck/internal/testutil"
)

func TestDoctorOnlyRequiresEnabledExecutables(t *testing.T) {
	testutil.SetHome(t, t.TempDir())
	t.Setenv("PATH", t.TempDir())
	dir := t.TempDir()
	s := store.Store{Path: filepath.Join(dir, "deck.json")}
	d := &model.Deck{Version: 1, Servers: map[string]model.ServerConfig{"fixture": {Command: "missing-fixture-executable"}}, Profiles: map[string]model.ProfileConfig{"fixture": {TargetPath: filepath.Join(dir, "agent.json")}}}
	if err := s.Save(d); err != nil {
		t.Fatal(err)
	}
	for _, enabled := range []bool{false, true} {
		if err := s.Update(func(d *model.Deck) error { return d.SetEnabled("fixture", "fixture", enabled) }); err != nil {
			t.Fatal(err)
		}
		c := NewRoot()
		c.SetArgs([]string{"--config", s.Path, "doctor"})
		c.SetOut(&bytes.Buffer{})
		c.SetErr(&bytes.Buffer{})
		err := c.Execute()
		if (err != nil) != enabled {
			t.Fatal("optional executables incorrectly affected diagnosis", enabled, err)
		}
	}
}
