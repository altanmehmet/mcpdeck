package client

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/altanmehmet/mcpdeck/internal/instructions"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
)

// NewDemo uses isolated sample files, without changing the process HOME or any
// real agent configuration. It is intended for native UI acceptance testing.
func NewDemo() (*App, func(), error) {
	home, err := os.MkdirTemp("", "mcpdeck-native-demo-")
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { os.RemoveAll(home) }
	fail := func(e error) (*App, func(), error) { cleanup(); return nil, nil, e }
	d := &model.Deck{Version: 1, Servers: map[string]model.ServerConfig{"sample-git": {Command: "git", Args: []string{"--version"}}}, Profiles: map[string]model.ProfileConfig{}}
	for _, key := range []string{"codex", "claude-code", "copilot-cli", "cursor"} {
		path := filepath.Join(home, "agents", key+".json")
		if err = store.AtomicWrite(path, []byte(`{}`)); err != nil {
			return fail(err)
		}
		d.Profiles[key] = model.ProfileConfig{TargetPath: path, EnabledServers: []string{}}
	}
	s := store.Store{Path: filepath.Join(home, "deck.json"), DisableDiscovery: true}
	if err = s.Save(d); err != nil {
		return fail(err)
	}
	a := New(s)
	a.manager = instructions.Manager{SourcePath: filepath.Join(home, "instructions.md"), Home: home, CodexHome: filepath.Join(home, ".codex"), CopilotHome: filepath.Join(home, ".copilot")}
	if _, err = a.manager.Apply(d, "# Working preferences\n\n- Follow existing project conventions.\n- Never log credentials or tokens.", false, nil); err != nil {
		return fail(fmt.Errorf("Cannot prepare isolated sample instructions: %w", err))
	}
	a.demo = true
	return a, cleanup, nil
}
