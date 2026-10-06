package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"github.com/altanmehmet/mcpdeck/internal/install"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"github.com/altanmehmet/mcpdeck/internal/testutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstallUsesSavedPlannerAndDocumentation(t *testing.T) {
	s := store.Store{Path: filepath.Join(t.TempDir(), "deck.json")}
	dir := filepath.Dir(s.Path)
	exe := filepath.Join(dir, "claude-fixture")
	raw, _ := json.Marshal(map[string]any{"structured_output": install.Plan{Version: 1, Name: "test-plan", Connection: `{"command":"echo","args":[]}`}})
	script := "#!/bin/sh\nrequest=$(cat)\ncase \"$request\" in *documentation-marker*) ;; *) exit 4 ;; esac\ncat <<'RESULT'\n" + string(raw) + "\nRESULT\n"
	if err := os.WriteFile(exe, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		exe = testutil.PlannerExecutable(t, string(raw), "", 0, "documentation-marker", false)
	}
	docs := filepath.Join(dir, "README.md")
	if err := os.WriteFile(docs, []byte("documentation-marker"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := runCommand(s, "planner", "set", "--provider", "claude", "--executable", exe); err != nil {
		t.Fatal(err)
	}
	if err := runCommand(s, "install", "fixture request", "--docs", docs, "--plan-only"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(s.Path); !os.IsNotExist(err) {
		t.Fatal("planning changed deck")
	}
}

func TestInteractivePlannerShowsProgressAndCompletion(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "claude-fixture")
	plan := install.Plan{Version: 1, Name: "oracle", Connection: `{"command":"echo","args":[]}`}
	raw, _ := json.Marshal(map[string]any{"structured_output": plan})
	if err := os.WriteFile(exe, []byte("#!/bin/sh\ncat >/dev/null\nprintf '%s' '"+string(raw)+"'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		exe = testutil.PlannerExecutable(t, string(raw), "", 0, "", false)
	}
	var output bytes.Buffer
	got, err := planWithProgress(context.Background(), install.PlannerOptions{Provider: "claude", Executable: exe}, "Oracle MCP", &output, true)
	if err != nil || got.Name != "oracle" {
		t.Fatal(got, err)
	}
	for _, expected := range []string{"checking public documentation", "No installation starts without your approval", "Plan ready after", "Showing it for review"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("missing progress text %q in %q", expected, output.String())
		}
	}
}

func TestManualPrerequisitesRequireActionableInteractiveConfirmation(t *testing.T) {
	p := install.Plan{Manual: []string{"Ask DBA to verify the account is read-only"}}
	var output bytes.Buffer
	if _, err := confirmManualPrerequisites(p, bufio.NewReader(strings.NewReader("")), &output, false); err == nil || !strings.Contains(err.Error(), "complete them") {
		t.Fatal("noninteractive manual prerequisites were not explained", err)
	}
	if reviewed, err := confirmManualPrerequisites(p, bufio.NewReader(strings.NewReader("\n")), &output, true); err == nil || reviewed {
		t.Fatal("empty response should cancel manual prerequisites", reviewed, err)
	}
	if reviewed, err := confirmManualPrerequisites(p, bufio.NewReader(strings.NewReader("DONE\n")), &output, true); err != nil || !reviewed {
		t.Fatal("explicit confirmation should continue", reviewed, err)
	}
	if !strings.Contains(output.String(), "Only continue after each item is completed or independently verified") {
		t.Fatal("confirmation prompt did not explain the required action")
	}
}

func TestDecliningColimaStartupLeavesRuntimeStopped(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Colima permission flow is macOS-only")
	}
	dir := t.TempDir()
	marker := filepath.Join(dir, "colima-started")
	docker := filepath.Join(dir, "docker")
	dockerScript := "#!/bin/sh\nif [ \"$1\" = context ]; then echo colima; exit 0; fi\nexit 1\n"
	if err := os.WriteFile(docker, []byte(dockerScript), 0700); err != nil {
		t.Fatal(err)
	}
	colima := filepath.Join(dir, "colima")
	if err := os.WriteFile(colima, []byte("#!/bin/sh\ntouch '"+marker+"'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	p := install.Plan{Requirements: []string{"docker"}}
	var output bytes.Buffer
	err := ensureRequirements(context.Background(), p, bufio.NewReader(strings.NewReader("NO\n")), &output, true)
	if err == nil || !strings.Contains(err.Error(), "cancelled") {
		t.Fatal("declining did not cancel startup", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("Colima started without permission")
	}
	if !strings.Contains(output.String(), "local Linux VM") {
		t.Fatal("startup prompt did not explain the action")
	}
}

func TestApprovingColimaStartupStartsRuntimeAndContinues(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Colima permission flow is macOS-only")
	}
	dir := t.TempDir()
	marker := filepath.Join(dir, "colima-started")
	docker := filepath.Join(dir, "docker")
	dockerScript := "#!/bin/sh\nif [ \"$1\" = info ] && [ -f '" + marker + "' ]; then exit 0; fi\nif [ \"$1\" = context ]; then echo colima; exit 0; fi\nexit 1\n"
	if err := os.WriteFile(docker, []byte(dockerScript), 0700); err != nil {
		t.Fatal(err)
	}
	colima := filepath.Join(dir, "colima")
	if err := os.WriteFile(colima, []byte("#!/bin/sh\n: > '"+marker+"'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	p := install.Plan{Requirements: []string{"docker"}}
	var output bytes.Buffer
	err := ensureRequirements(context.Background(), p, bufio.NewReader(strings.NewReader("YES\n")), &output, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("Colima did not start after permission", err)
	}
	if !strings.Contains(output.String(), "Docker is ready") {
		t.Fatal("installation did not resume after Docker became ready")
	}
}

func TestPlannerSettingsAndOverride(t *testing.T) {
	s := store.Store{Path: filepath.Join(t.TempDir(), "deck.json")}
	o, err := loadPlanner(s)
	if err != nil || o.Provider != "codex" {
		t.Fatal("default changed", err)
	}
	t.Setenv("ANTHROPIC_API_KEY", "private-setting-key")
	if err := runCommand(s, "planner", "set", "--provider", "anthropic-api", "--model", "test-model"); err != nil {
		t.Fatal(err)
	}
	o, err = loadPlanner(s)
	if err != nil || o.Provider != "anthropic-api" || o.Model != "test-model" {
		t.Fatal(o, err)
	}
	raw, _ := os.ReadFile(plannerSettingsPath(s))
	if bytes.Contains(raw, []byte("private-setting-key")) {
		t.Fatal("key written to disk")
	}
	if err := runCommand(s, "planner", "set", "--provider", "gemini-api"); err == nil {
		t.Fatal("missing model accepted")
	}
	o, _ = loadPlanner(s)
	if o.Provider != "anthropic-api" {
		t.Fatal("invalid update mutated settings")
	}
	c := NewRoot()
	c.SetOut(&bytes.Buffer{})
	c.SetErr(&bytes.Buffer{})
	c.SetArgs([]string{"--config", s.Path, "install", "test docs", "--provider", "unknown", "--plan-only"})
	if err := c.Execute(); err == nil || !strings.Contains(err.Error(), "unknown planner provider") {
		t.Fatal("override ignored", err)
	}
}
