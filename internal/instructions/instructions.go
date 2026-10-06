// Package instructions distributes personal guidance to documented global
// instruction locations. Project paths and MCP configuration are never edited.
package instructions

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
)

const MaxBytes = 24 * 1024
const begin = "<!-- mcpdeck:global-instructions:begin -->"
const end = "<!-- mcpdeck:global-instructions:end -->"

type Manager struct {
	SourcePath, Home, CodexHome, CopilotHome string
}

type Target struct {
	Agent, Path, Header, Note string
	MaxChars                  int
}

type Result struct {
	Agent, Path, Status, Detail string
}

func New(s store.Store) Manager {
	home, _ := os.UserHomeDir()
	codex := os.Getenv("CODEX_HOME")
	if codex == "" {
		codex = filepath.Join(home, ".codex")
	}
	copilot := os.Getenv("COPILOT_HOME")
	if copilot == "" {
		copilot = filepath.Join(home, ".copilot")
	}
	return Manager{filepath.Join(filepath.Dir(s.Path), "instructions.md"), home, store.ExpandUserPath(codex, home), store.ExpandUserPath(copilot, home)}
}

func Validate(text string) error {
	if len(text) > MaxBytes || !utf8.ValidString(text) || strings.ContainsRune(text, 0) {
		return fmt.Errorf("instructions must be UTF-8 text up to 24 KiB without NUL bytes")
	}
	if strings.Contains(text, begin) || strings.Contains(text, end) {
		return fmt.Errorf("instructions contain reserved MCPDeck markers")
	}
	return nil
}

func (m Manager) Load() (string, error) {
	raw, err := readRegular(m.SourcePath)
	if err != nil {
		return "", err
	}
	if err := Validate(string(raw)); err != nil {
		return "", err
	}
	return string(raw), nil
}

func (m Manager) Targets(d *model.Deck) []Target {
	var targets []Target
	for agent, profile := range d.Profiles {
		t := Target{Agent: agent}
		switch {
		case agent == "codex":
			t.Path = filepath.Join(m.CodexHome, "AGENTS.md")
			if info, err := os.Stat(filepath.Join(m.CodexHome, "AGENTS.override.md")); err == nil && !info.IsDir() {
				t.Path = filepath.Join(m.CodexHome, "AGENTS.override.md")
			}
		case agent == "claude-code":
			t.Path = filepath.Join(m.Home, ".claude", "CLAUDE.md")
		case agent == "copilot-cli":
			t.Path = filepath.Join(m.CopilotHome, "copilot-instructions.md")
		case agent == "copilot":
			t.Path = filepath.Join(m.Home, ".copilot", "copilot-instructions.md")
			t.Note = "Copilot Agent Host; legacy VS Code Local sessions require manual user instructions"
		case agent == "gemini-cli" || agent == "antigravity":
			t.Path = filepath.Join(m.Home, ".gemini", "GEMINI.md")
		case agent == "qwen-code":
			t.Path = filepath.Join(m.Home, ".qwen", "QWEN.md")
		case agent == "opencode":
			t.Path = filepath.Join(m.Home, ".config", "opencode", "AGENTS.md")
		case agent == "cursor":
			t.Path = filepath.Join(m.Home, ".cursor", "rules", "mcpdeck-global.mdc")
			t.Header = "---\ndescription: Personal instructions managed by MCPDeck\nalwaysApply: true\n---\n"
		case agent == "windsurf":
			t.Path = filepath.Join(m.Home, ".codeium", "windsurf", "memories", "global_rules.md")
			t.MaxChars = 6000
		case agent == "kiro":
			t.Path = filepath.Join(m.Home, ".kiro", "steering", "mcpdeck-global.md")
			t.Header = "---\ninclusion: always\n---\n"
		case agent == "cline":
			t.Path = filepath.Join(m.Home, ".cline", "rules", "mcpdeck-global.md")
		case strings.HasPrefix(agent, "cline-vscode-"):
			t.Path = filepath.Join(m.Home, "Documents", "Cline", "Rules", "mcpdeck-global.md")
		default:
			t.Note = "No verified automatic global instruction adapter; use this agent's personal instruction settings"
		}
		if t.Path != "" && !store.AgentDetected(agent, profile) {
			t.Note = strings.TrimSpace(t.Note + " (agent not detected)")
		}
		targets = append(targets, t)
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].Agent < targets[j].Agent })
	return targets
}

// merge replaces only our block and preserves all bytes outside it. Broken or
// duplicate markers fail closed rather than overwriting personal instructions.
func merge(existing, text string, t Target) (string, error) {
	if err := Validate(text); err != nil {
		return "", err
	}
	start, finish := strings.Index(existing, begin), strings.Index(existing, end)
	if strings.Count(existing, begin) > 1 || strings.Count(existing, end) > 1 || (start < 0) != (finish < 0) || (start >= 0 && finish < start) {
		return "", fmt.Errorf("invalid managed instruction block; repair the markers before syncing")
	}
	block := ""
	if strings.TrimSpace(text) != "" {
		block = begin + "\n" + strings.TrimSpace(text) + "\n" + end
	}
	var result string
	if start >= 0 {
		result = existing[:start] + block + existing[finish+len(end):]
	} else if block == "" {
		result = existing
	} else {
		prefix := existing
		if prefix == "" {
			prefix = t.Header
		} else if !strings.HasSuffix(prefix, "\n\n") {
			prefix += "\n\n"
		}
		result = prefix + block + "\n"
	}
	if t.MaxChars > 0 && utf8.RuneCountInString(result) > t.MaxChars {
		return "", fmt.Errorf("combined global instructions exceed this agent's %d character limit", t.MaxChars)
	}
	return result, nil
}

func readRegular(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("instruction path is not a regular file: %s", path)
	}
	if info.Size() > 1024*1024 {
		return nil, fmt.Errorf("instruction file exceeds 1 MiB")
	}
	return os.ReadFile(path)
}

func (m Manager) Status(d *model.Deck, all bool) ([]Result, error) {
	text, err := m.Load()
	if err != nil {
		return nil, err
	}
	return m.distribute(d, text, all, false), nil
}

func (m Manager) Preview(d *model.Deck, text string, all bool) ([]Result, error) {
	if err := Validate(text); err != nil {
		return nil, err
	}
	return m.distribute(d, text, all, false), nil
}

// Apply saves the canonical text under a cross-process lock, then syncs each
// eligible agent independently. Partial failures remain visible and retryable.
func (m Manager) Apply(d *model.Deck, text string, all bool, expected *string) ([]Result, error) {
	if err := Validate(text); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(m.SourcePath), 0700); err != nil {
		return nil, err
	}
	lock := m.SourcePath + ".lock"
	if err := os.Mkdir(lock, 0700); err != nil {
		return nil, fmt.Errorf("instructions are locked by another writer; try again")
	}
	defer os.Remove(lock)
	previous, err := m.Load()
	if err != nil {
		return nil, err
	}
	if expected != nil && previous != *expected {
		return nil, fmt.Errorf("instructions changed in another session; reopen the editor")
	}
	if previous != text {
		if previous != "" {
			if err := store.AtomicWrite(m.SourcePath+".mcpdeck-backup", []byte(previous)); err != nil {
				return nil, err
			}
		}
		if err := store.AtomicWrite(m.SourcePath, []byte(text)); err != nil {
			return nil, err
		}
	}
	results := m.distribute(d, text, all, true)
	var failures []error
	for _, r := range results {
		if r.Status == "failed" {
			failures = append(failures, fmt.Errorf("%s: %s", r.Agent, r.Detail))
		}
	}
	return results, errors.Join(failures...)
}

func (m Manager) distribute(d *model.Deck, text string, all, write bool) []Result {
	var results []Result
	seen := map[string]Result{}
	for _, t := range m.Targets(d) {
		r := Result{Agent: t.Agent, Path: t.Path, Detail: t.Note}
		if t.Path == "" {
			r.Status = "manual"
			results = append(results, r)
			continue
		}
		if !all && !store.AgentDetected(t.Agent, d.Profiles[t.Agent]) {
			r.Status = "not detected"
			results = append(results, r)
			continue
		}
		if previous, ok := seen[t.Path]; ok {
			r.Status = previous.Status
			if previous.Status == "failed" {
				r.Detail = previous.Detail
			}
			results = append(results, r)
			continue
		}
		err := m.checkParents(t.Path)
		var raw []byte
		if err == nil {
			raw, err = readRegular(t.Path)
		}
		var next string
		if err == nil {
			next, err = merge(string(raw), text, t)
		}
		if err == nil {
			r.Status = "up to date"
			if next != string(raw) {
				r.Status = "pending"
				if write {
					if len(raw) > 0 {
						err = store.AtomicWrite(t.Path+".mcpdeck-backup", raw)
					}
					if err == nil {
						err = store.AtomicWrite(t.Path, []byte(next))
					}
					if err == nil {
						r.Status = "synced"
					}
				}
			}
		}
		if err != nil {
			r.Status = "failed"
			r.Detail = err.Error()
		}
		seen[t.Path] = r
		results = append(results, r)
	}
	return results
}

// A global rules directory may be symlinked into a repository. Refuse such
// redirects so distribution cannot turn into an accidental project edit.
func (m Manager) checkParents(path string) error {
	for parent := filepath.Dir(path); parent != m.Home && parent != filepath.Dir(parent); parent = filepath.Dir(parent) {
		rel, err := filepath.Rel(m.Home, parent)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			break
		}
		info, err := os.Lstat(parent)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("global instruction directory is a symlink; refusing to edit a possible project location: %s", parent)
		}
	}
	return nil
}
