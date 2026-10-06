package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/altanmehmet/mcpdeck/internal/install"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"github.com/altanmehmet/mcpdeck/internal/testutil"
)

func TestHelper(t *testing.T) { testutil.Serve() }
func installFixture(t *testing.T) (store.Store, install.Plan, string) {
	t.Helper()
	s := allProfilesFixture(t)
	cfg := testutil.Config()
	cfg.Env["AUTH"] = "${INSTALL_SECRET}"
	connection, _ := json.Marshal(map[string]any{"command": cfg.Command, "args": cfg.Args, "env": cfg.Env})
	p := install.Plan{Version: 1, Name: "installed-demo", Connection: string(connection), Steps: []install.Step{{Command: "touch", Args: []string{"installed-marker"}}}}
	raw, _ := json.Marshal(p)
	path := filepath.Join(filepath.Dir(s.Path), "plan.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return s, p, path
}
func TestInstallVerifiedAllProfilesAndLifecycle(t *testing.T) {
	s, p, path := installFixture(t)
	c := NewRoot()
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&out)
	c.SetIn(strings.NewReader(`{"INSTALL_SECRET":"local$literal"}`))
	c.SetArgs([]string{"--config", s.Path, "install", "--plan", path, "--approve", p.Digest(), "--values-stdin"})
	if err := c.Execute(); err != nil {
		t.Fatal(err, out.String())
	}
	if strings.Contains(out.String(), "local$literal") {
		t.Fatal("secret leaked")
	}
	if !strings.Contains(out.String(), "Verified: 2 tools.") {
		t.Fatal("real tools probe missing", out.String())
	}
	d, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	for key, profile := range d.Profiles {
		if !d.IsEnabled(key, p.Name) {
			t.Fatal("not selected", key)
		}
		_, servers := readAgentConfig(t, profile)
		if servers[p.Name] == nil || servers["external"] == nil {
			t.Fatal("config not preserved", key)
		}
		before, e := os.ReadFile(profile.TargetPath + ".mcpdeck-backup")
		if e != nil || bytes.Contains(before, []byte(p.Name)) {
			t.Fatal("bad backup", key, e)
		}
	}
	if err := runCommand(s, "disable", p.Name); err != nil {
		t.Fatal(err)
	}
	d, _ = s.Load()
	for _, profile := range d.Profiles {
		_, servers := readAgentConfig(t, profile)
		if servers[p.Name] != nil {
			t.Fatal("not disabled")
		}
	}
	if err := runCommand(s, "enable", p.Name); err != nil {
		t.Fatal(err)
	}
}
func TestInstallFailureLeavesAgentFilesUntouched(t *testing.T) {
	for _, failure := range []string{"approval", "probe", "conflict", "missing-input"} {
		t.Run(failure, func(t *testing.T) {
			s, p, path := installFixture(t)
			d, _ := s.Load()
			if failure == "probe" {
				p.Connection = `{"command":"/nonexistent/mcpdeck-server","args":[]}`
			}
			if failure == "conflict" {
				profile := d.Profiles["cursor"]
				os.WriteFile(profile.TargetPath, []byte(`{"mcpServers":{"installed-demo":{"command":"keep"}}}`), 0600)
			}
			before := map[string][]byte{}
			for _, profile := range d.Profiles {
				before[profile.TargetPath], _ = os.ReadFile(profile.TargetPath)
			}
			raw, _ := json.Marshal(p)
			os.WriteFile(path, raw, 0600)
			approval := p.Digest()
			if failure == "approval" {
				approval = "incorrect"
			}
			c := NewRoot()
			c.SetOut(&bytes.Buffer{})
			c.SetErr(&bytes.Buffer{})
			value := `{"INSTALL_SECRET":"test"}`
			if failure == "missing-input" {
				value = `{}`
			}
			c.SetIn(strings.NewReader(value))
			c.SetArgs([]string{"--config", s.Path, "install", "--plan", path, "--approve", approval, "--values-stdin"})
			if err := c.Execute(); err == nil {
				t.Fatal("failure accepted")
			}
			after, _ := s.Load()
			if _, ok := after.Servers[p.Name]; ok {
				t.Fatal("failed server saved")
			}
			for path, original := range before {
				raw, _ := os.ReadFile(path)
				if !bytes.Equal(raw, original) {
					t.Fatal("agent modified before verification")
				}
			}
		})
	}
}
func TestPlanOnlyDoesNotInstall(t *testing.T) {
	s, _, path := installFixture(t)
	if err := runCommand(s, "install", "--plan", path, "--plan-only"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(s.Path), "installations")); !os.IsNotExist(err) {
		t.Fatal("plan mutated installation")
	}
}
func TestInstallationTargets(t *testing.T) {
	d := &model.Deck{Profiles: map[string]model.ProfileConfig{"absent": {TargetPath: filepath.Join(t.TempDir(), "missing")}}}
	if len(installationTargets(d, false)) != 0 || len(installationTargets(d, true)) != 1 {
		t.Fatal("target selection wrong")
	}
}
