package instructions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/altanmehmet/mcpdeck/internal/model"
)

func fixture(t *testing.T) (Manager, *model.Deck) {
	t.Helper()
	home := t.TempDir()
	m := Manager{SourcePath: filepath.Join(home, "deck", "instructions.md"), Home: home, CodexHome: filepath.Join(home, "custom-codex"), CopilotHome: filepath.Join(home, "custom-copilot")}
	d := &model.Deck{Version: 1, Servers: map[string]model.ServerConfig{}, Profiles: map[string]model.ProfileConfig{}}
	for _, name := range []string{"codex", "claude-code", "copilot-cli", "copilot", "gemini-cli", "antigravity", "qwen-code", "opencode", "cursor", "windsurf", "kiro", "cline", "claude", "trae"} {
		config := filepath.Join(home, "configs", name+".json")
		write(t, config, "{}")
		d.Profiles[name] = model.ProfileConfig{TargetPath: config}
	}
	return m, d
}

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
}
func read(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestDistributePreservesPersonalAndProjectInstructions(t *testing.T) {
	m, d := fixture(t)
	path := filepath.Join(m.CodexHome, "AGENTS.md")
	write(t, path, "# Existing personal guidance\nUse tabs.\n")
	project := filepath.Join(m.Home, "project", "AGENTS.md")
	write(t, project, "Project-specific instructions\n")
	results, err := m.Apply(d, "Never log credentials.\nVerify tests.", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, result := range results {
		if result.Agent == "claude" || result.Agent == "trae" {
			if result.Status != "manual" {
				t.Fatal(result)
			}
			continue
		}
		if result.Status != "synced" {
			t.Fatalf("%s: %s %s", result.Agent, result.Status, result.Detail)
		}
		if !strings.Contains(read(t, result.Path), "Never log credentials.") {
			t.Fatal(result.Path)
		}
	}
	if !strings.HasPrefix(read(t, path), "# Existing personal guidance\nUse tabs.\n") {
		t.Fatal("existing personal text changed")
	}
	if read(t, path+".mcpdeck-backup") != "# Existing personal guidance\nUse tabs.\n" {
		t.Fatal("original backup missing")
	}
	before := read(t, path)
	if _, err := m.Apply(d, "Never log credentials.\nVerify tests.", false, nil); err != nil {
		t.Fatal(err)
	}
	if read(t, path) != before || strings.Count(before, begin) != 1 {
		t.Fatal("sync was not idempotent")
	}
	if _, err := m.Apply(d, "Updated guidance", false, nil); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(read(t, path), "Never log credentials.") {
		t.Fatal("previous managed guidance remained")
	}
	if _, err := m.Apply(d, "", true, nil); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(read(t, path), begin) || !strings.Contains(read(t, path), "Use tabs.") {
		t.Fatal("clear damaged personal text")
	}
	if read(t, project) != "Project-specific instructions\n" {
		t.Fatal("project text changed")
	}
}

func TestMalformedBlockAndLimitArePartialFailures(t *testing.T) {
	m, d := fixture(t)
	codex := filepath.Join(m.CodexHome, "AGENTS.md")
	write(t, codex, begin+"\nMalformed")
	results, err := m.Apply(d, strings.Repeat("x", 6100), false, nil)
	if err == nil {
		t.Fatal("expected sync failures")
	}
	failed := map[string]bool{}
	for _, r := range results {
		if r.Status == "failed" {
			failed[r.Agent] = true
		}
	}
	if !failed["codex"] || !failed["windsurf"] {
		t.Fatal(results)
	}
	if read(t, codex) != begin+"\nMalformed" {
		t.Fatal("malformed file overwritten")
	}
	if !strings.Contains(read(t, filepath.Join(m.Home, ".qwen", "QWEN.md")), strings.Repeat("x", 6100)) {
		t.Fatal("one failure blocked other agents")
	}
	if text, err := m.Load(); err != nil || len(text) != 6100 {
		t.Fatal("canonical text not retained for retry", err)
	}
}

func TestRejectSymlinkToProjectAndConcurrentEdit(t *testing.T) {
	m, d := fixture(t)
	project := filepath.Join(m.Home, "project")
	write(t, filepath.Join(project, "QWEN.md"), "Project only")
	if err := os.Symlink(project, filepath.Join(m.Home, ".qwen")); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Apply(d, "Global instructions", false, nil); err == nil {
		t.Fatal("symlink parent accepted")
	}
	if read(t, filepath.Join(project, "QWEN.md")) != "Project only" {
		t.Fatal("symlink changed project")
	}
	expected := "outdated"
	if _, err := m.Apply(d, "another edit", false, &expected); err == nil {
		t.Fatal("stale editor overwrote another writer")
	}
	if text, _ := m.Load(); text != "Global instructions" {
		t.Fatal("stale edit was saved")
	}
}

func TestNotDetectedPreviewAndCodexOverride(t *testing.T) {
	m, d := fixture(t)
	write(t, filepath.Join(m.CodexHome, "AGENTS.override.md"), "Override text")
	qwen := d.Profiles["qwen-code"]
	qwen.TargetPath = filepath.Join(m.Home, "nonexistent.json")
	d.Profiles["qwen-code"] = qwen
	results, err := m.Preview(d, "Shared instructions", false)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if r.Agent == "codex" && !strings.HasSuffix(r.Path, "AGENTS.override.md") {
			t.Fatal("ignored global override")
		}
		if r.Agent == "qwen-code" && r.Status != "not detected" {
			t.Fatal("undetected agent targeted")
		}
	}
	if _, err := os.Stat(m.SourcePath); !os.IsNotExist(err) {
		t.Fatal("preview wrote central state")
	}
	if _, err := m.Apply(d, "Shared instructions", false, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(m.Home, ".qwen", "QWEN.md")); !os.IsNotExist(err) {
		t.Fatal("undetected agent written")
	}
}
