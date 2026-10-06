package cmd

import (
	"bytes"
	"github.com/altanmehmet/mcpdeck/internal/testutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
)

func TestInstructionsCLISetStatusClear(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	t.Setenv("COPILOT_HOME", filepath.Join(home, ".copilot"))
	s := store.Store{Path: filepath.Join(home, "deck", "deck.json")}
	target := filepath.Join(home, ".codex", "config.toml")
	if err := store.AtomicWrite(target, []byte("")); err != nil {
		t.Fatal(err)
	}
	d := &model.Deck{Version: 1, Servers: map[string]model.ServerConfig{}, Profiles: map[string]model.ProfileConfig{"codex": {TargetPath: target, Format: "codex"}}}
	if err := s.Save(d); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) (string, error) {
		c := instructionsCommand(func() store.Store { return s })
		var out bytes.Buffer
		c.SetOut(&out)
		c.SetErr(&out)
		c.SetArgs(args)
		err := c.Execute()
		return out.String(), err
	}
	if _, err := run("set", "Use readable code."); err != nil {
		t.Fatal(err)
	}
	if _, err := run("append", "Never log credentials."); err != nil {
		t.Fatal(err)
	}
	if _, err := run("append", "Never log credentials."); err != nil {
		t.Fatal(err)
	}
	if out, err := run("show"); err != nil || strings.Count(out, "Never log credentials.") != 1 {
		t.Fatal(out, err)
	}
	if out, err := run("show", "--agent", "codex"); err != nil || !strings.Contains(out, "Use readable code.") || !strings.Contains(out, "mcpdeck:global-instructions") {
		t.Fatal("current native instructions were not shown", out, err)
	}
	if out, err := run("files"); err != nil || !strings.Contains(out, "AGENTS.md") {
		t.Fatal(out, err)
	}
	if out, err := run("show"); err != nil || !strings.Contains(out, "Use readable code.") {
		t.Fatal(out, err)
	}
	if out, err := run("status"); err != nil || !strings.Contains(out, "codex: up to date") {
		t.Fatal(out, err)
	}
	if _, err := run("clear"); err == nil {
		t.Fatal("clear lacked explicit confirmation")
	}
	if _, err := run("clear", "--yes"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(home, ".codex", "AGENTS.md"))
	if err != nil || strings.Contains(string(raw), "Use readable code.") {
		t.Fatal(string(raw), err)
	}
}
