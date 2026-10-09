package instructions

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf8"

	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
)

const MaxDocumentBytes = 1024 * 1024

type Document struct {
	Agent, Path, Note string
}

func (m Manager) SelectDocument(d *model.Deck, agent, path string) (Document, error) {
	documents, err := m.Documents(d)
	if err != nil {
		return Document{}, err
	}
	var matches []Document
	for _, doc := range documents {
		if doc.Agent == agent && (path == "" || filepath.Clean(path) == doc.Path) {
			matches = append(matches, doc)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 && path == "" {
		// Prefer the established main file when it contains guidance or cannot
		// be read. An absent/empty main file must not hide modular instructions.
		var primary *Document
		for _, target := range m.Targets(d) {
			if target.Agent == agent {
				for _, doc := range matches {
					if target.Path == doc.Path {
						primary = &doc
						text, err := m.ReadDocument(d, doc)
						if err != nil || strings.TrimSpace(text) != "" {
							return doc, nil
						}
					}
				}
			}
		}
		var populated []Document
		for _, doc := range matches {
			text, err := m.ReadDocument(d, doc)
			if err != nil {
				return Document{}, err
			}
			if strings.TrimSpace(text) != "" {
				populated = append(populated, doc)
			}
		}
		if len(populated) == 1 {
			return populated[0], nil
		}
		if len(populated) == 0 && primary != nil {
			return *primary, nil
		}
		return Document{}, fmt.Errorf("multiple global files exist for %s; list instructions files and select one with --path", agent)
	}
	return Document{}, fmt.Errorf("no verified global instruction file for %s; list instructions files to see supported targets", agent)
}

func directoryRules(t Target) bool {
	return t.Agent == "cursor" || t.Agent == "kiro" || t.Agent == "cline" || strings.HasPrefix(t.Agent, "cline-vscode-")
}

// Documents inventories only documented global locations, including rules
// written outside MCPDeck. It never scans project instruction directories.
func (m Manager) Documents(d *model.Deck) ([]Document, error) {
	var documents []Document
	for _, target := range m.Targets(d) {
		if target.Path == "" {
			continue
		}
		paths := []string{target.Path}
		// Antigravity's documented standalone global alternatives are read sources,
		// while shared distribution keeps its established GEMINI.md target.
		if target.Agent == "gemini-cli" {
			settings := filepath.Join(m.Home, ".gemini", "settings.json")
			if err := m.checkParents(settings); err == nil {
				if raw, err := readRegular(settings); err == nil && len(raw) > 0 {
					var cfg struct {
						Context struct {
							FileName json.RawMessage `json:"fileName"`
						} `json:"context"`
					}
					if json.Unmarshal(raw, &cfg) == nil && len(cfg.Context.FileName) > 0 {
						var names []string
						var single string
						if json.Unmarshal(cfg.Context.FileName, &single) == nil {
							names = []string{single}
						} else {
							_ = json.Unmarshal(cfg.Context.FileName, &names)
						}
						if len(names) <= 32 {
							for _, name := range names {
								if name == "" || name == "." || name == ".." || !strings.HasSuffix(strings.ToLower(name), ".md") || strings.ContainsAny(name, "/\\\x00") {
									continue
								}
								path := filepath.Join(m.Home, ".gemini", name)
								if info, e := os.Lstat(path); e == nil && info.Mode().IsRegular() {
									paths = append(paths, path)
								}
							}
						}
					}
				}
			}
		}
		if target.Agent == "antigravity" {
			for _, path := range []string{filepath.Join(m.Home, ".gemini", "AGENTS.md"), filepath.Join(m.Home, ".gemini", "config", "AGENTS.md"), filepath.Join(m.Home, ".gemini", "config", "GEMINI.md")} {
				if info, e := os.Lstat(path); e == nil && info.Mode().IsRegular() {
					paths = append(paths, path)
				}
			}
		}

		if target.Agent == "codex" && filepath.Base(target.Path) == "AGENTS.override.md" {
			paths = append(paths, filepath.Join(m.CodexHome, "AGENTS.md"))
		}
		if directoryRules(target) {
			paths = nil
			root := filepath.Dir(target.Path)
			if err := m.checkParents(target.Path); err != nil {
				documents = append(documents, Document{target.Agent, target.Path, "unreadable: " + err.Error()})
				continue
			}
			err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
				if os.IsNotExist(err) && path == root {
					return nil
				}
				if err != nil {
					return err
				}
				if entry.Type()&os.ModeSymlink != 0 {
					return nil
				}
				if !entry.IsDir() && (strings.HasSuffix(path, ".md") || strings.HasSuffix(path, ".mdc")) {
					paths = append(paths, path)
				}
				if len(paths) > 1000 {
					return fmt.Errorf("too many global rule files for %s", target.Agent)
				}
				return nil
			})
			if err != nil {
				target.Note = strings.TrimSpace(target.Note + " (scan incomplete: " + err.Error() + ")")
			}
			if len(paths) == 0 {
				paths = []string{target.Path}
			}
		}
		// Some clients also read modular personal rules alongside their main
		// global document. Include those files in the existing-file browser.
		var extraRoots []string
		if target.Agent == "claude-code" {
			extraRoots = []string{filepath.Join(m.Home, ".claude", "rules")}
		}
		if target.Agent == "copilot-cli" || target.Agent == "copilot" {
			extraRoots = []string{filepath.Join(filepath.Dir(target.Path), "instructions")}
		}
		if target.Agent == "cline" || strings.HasPrefix(target.Agent, "cline-vscode-") {
			extraRoots = []string{filepath.Join(m.Home, "Documents", "Cline", "Rules"), filepath.Join(m.Home, ".cline", "rules"), filepath.Join(m.Home, "Cline", "Rules")}
		}
		if target.Agent == "antigravity" {
			extraRoots = append(extraRoots, filepath.Join(m.Home, ".gemini", "config", "rules"), filepath.Join(m.Home, ".gemini", "antigravity-cli", "rules"))
		}
		if target.Agent == "copilot" {
			// Local VS Code stores user instruction files in profile storage. Never
			// derive these paths from arbitrary workspace MCP configuration paths.
			var user string
			switch runtime.GOOS {
			case "darwin":
				user = filepath.Join(m.Home, "Library", "Application Support", "Code", "User")
			case "windows":
				if appData := os.Getenv("APPDATA"); appData != "" {
					user = filepath.Join(appData, "Code", "User")
				}
			default:
				user = filepath.Join(m.Home, ".config", "Code", "User")
			}
			if user != "" {
				extraRoots = append(extraRoots, filepath.Join(user, "prompts"), filepath.Join(user, "profiles"))
			}
		}
		for _, root := range extraRoots {
			if root == filepath.Dir(target.Path) {
				continue
			}
			if err := m.checkParents(filepath.Join(root, "placeholder.md")); err != nil {
				target.Note += " (additional rules unreadable)"
				continue
			}
			err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
				if os.IsNotExist(err) && path == root {
					return nil
				}
				if err != nil {
					return err
				}
				if entry.Type()&os.ModeSymlink != 0 {
					return nil
				}
				if !entry.IsDir() && strings.HasSuffix(path, ".md") {
					if (target.Agent == "copilot-cli" || target.Agent == "copilot") && !strings.HasSuffix(path, ".instructions.md") {
						return nil
					}
					if target.Agent == "antigravity" && filepath.Dir(path) != root {
						return nil
					}
					if target.Agent == "copilot" && (filepath.Base(root) == "prompts" || filepath.Base(root) == "profiles") && !strings.HasSuffix(path, ".instructions.md") {
						return nil
					}
					paths = append(paths, path)
				}
				if len(paths) > 1000 {
					return fmt.Errorf("too many global rules")
				}
				return nil
			})
			if err != nil {
				target.Note += " (additional rules scan incomplete)"
			}
		}
		seen := map[string]bool{}
		for _, path := range paths {
			if seen[path] {
				continue
			}
			seen[path] = true
			note := target.Note
			if _, err := os.Lstat(path); os.IsNotExist(err) {
				note = strings.TrimSpace(note + " (not created)")
			}
			documents = append(documents, Document{target.Agent, path, note})
		}
	}
	return documents, nil
}

func (m Manager) documentTarget(d *model.Deck, doc Document) (Target, error) {
	documents, err := m.Documents(d)
	if err != nil {
		return Target{}, err
	}
	for _, allowed := range documents {
		if allowed.Agent == doc.Agent && allowed.Path == doc.Path {
			for _, target := range m.Targets(d) {
				if target.Agent == doc.Agent {
					target.Path = doc.Path
					return target, nil
				}
			}
		}
	}
	return Target{}, fmt.Errorf("not a registered global instruction file")
}

func (m Manager) ReadDocument(d *model.Deck, doc Document) (string, error) {
	if _, err := m.documentTarget(d, doc); err != nil {
		return "", err
	}
	if err := m.checkParents(doc.Path); err != nil {
		return "", err
	}
	raw, err := readRegular(doc.Path)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(raw) || strings.ContainsRune(string(raw), 0) {
		return "", fmt.Errorf("global instructions are not valid UTF-8 text")
	}
	return string(raw), nil
}

func managedBlock(text string) (string, error) {
	start, finish := strings.Index(text, begin), strings.Index(text, end)
	if strings.Count(text, begin) > 1 || strings.Count(text, end) > 1 || (start < 0) != (finish < 0) || (start >= 0 && finish < start) {
		return "", fmt.Errorf("invalid managed instruction markers")
	}
	if start < 0 {
		return "", nil
	}
	return text[start : finish+len(end)], nil
}

// ReplacePersonalText replaces a document's personal text while retaining the
// exact managed shared block. The normal document checks still run on save.
func ReplacePersonalText(current, replacement string) (string, error) {
	block, err := managedBlock(current)
	if err != nil {
		return "", err
	}
	if strings.Contains(replacement, begin) || strings.Contains(replacement, end) {
		return "", fmt.Errorf("edit shared instructions in the central editor; replacement personal text cannot contain managed markers")
	}
	// Rule-file front matter controls activation, rather than instruction text.
	// Keep it when clearing or replacing the visible document in one operation.
	if strings.HasPrefix(current, "---\n") {
		if finish := strings.Index(current[4:], "\n---\n"); finish >= 0 {
			replacement = current[:4+finish+5] + replacement
		}
	}
	if block == "" {
		return replacement, nil
	}
	if replacement != "" && !strings.HasSuffix(replacement, "\n\n") {
		replacement += "\n\n"
	}
	return replacement + block + "\n", nil
}

// PersonalText hides the managed shared block and rule activation metadata from
// a personal-text-only editor. ReplacePersonalText restores both when saving.
func PersonalText(raw string) (string, error) {
	block, err := managedBlock(raw)
	if err != nil {
		return "", err
	}
	text := strings.Replace(raw, block, "", 1)
	if strings.HasPrefix(text, "---\n") {
		if finish := strings.Index(text[4:], "\n---\n"); finish >= 0 {
			text = text[4+finish+5:]
		}
	}
	return strings.TrimSpace(text), nil
}

// SaveDocument edits an existing agent's personal text. Shared instructions
// remain owned by the central editor and cannot silently drift in one agent.
func (m Manager) SaveDocument(d *model.Deck, doc Document, text, expected string) error {
	if err := os.MkdirAll(filepath.Dir(m.SourcePath), 0700); err != nil {
		return err
	}
	lock := m.SourcePath + ".lock"
	if err := os.Mkdir(lock, 0700); err != nil {
		return fmt.Errorf("instructions are locked by another writer; try again")
	}
	defer os.Remove(lock)
	if err := m.CheckDocument(d, doc, text, expected); err != nil {
		return err
	}
	if expected == text {
		return nil
	}
	if expected != "" {
		if err := store.AtomicWrite(doc.Path+".mcpdeck-backup", []byte(expected)); err != nil {
			return err
		}
	}
	return store.AtomicWrite(doc.Path, []byte(text))
}

func (m Manager) CheckDocument(d *model.Deck, doc Document, text, expected string) error {
	if len(text) > MaxDocumentBytes || !utf8.ValidString(text) || strings.ContainsRune(text, 0) {
		return fmt.Errorf("global instruction document must be valid UTF-8 up to 1 MiB")
	}
	target, err := m.documentTarget(d, doc)
	if err != nil {
		return err
	}
	if target.MaxChars > 0 && utf8.RuneCountInString(text) > target.MaxChars {
		return fmt.Errorf("this agent's global instructions exceed %d characters", target.MaxChars)
	}
	current, err := m.ReadDocument(d, doc)
	if err != nil {
		return err
	}
	if current != expected {
		return fmt.Errorf("this file changed since it was opened; reload it before saving")
	}
	beforeBlock, err := managedBlock(current)
	if err != nil {
		return err
	}
	afterBlock, err := managedBlock(text)
	if err != nil {
		return err
	}
	if beforeBlock != afterBlock {
		return fmt.Errorf("edit shared instructions in the central editor to update all agents; keep this file's MCPDeck block unchanged")
	}
	return nil
}

// SharedText removes file-specific metadata and markers when the user explicitly
// chooses an agent's current guidance as the shared text for all agents.
func SharedText(raw string) (string, error) {
	if _, err := managedBlock(raw); err != nil {
		return "", err
	}
	text := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(raw, begin, ""), end, ""))
	if strings.HasPrefix(text, "---\n") {
		if finish := strings.Index(text[4:], "\n---"); finish >= 0 {
			text = strings.TrimSpace(text[4+finish+4:])
		}
	}
	return text, Validate(text)
}

func AppendText(current, addition string) (string, error) {
	addition = strings.TrimSpace(addition)
	if addition == "" {
		return "", fmt.Errorf("instruction is empty")
	}
	text := strings.TrimSpace(current)
	if !strings.Contains("\n"+text+"\n", "\n"+addition+"\n") {
		if text != "" {
			text += "\n\n"
		}
		text += addition
	}
	return text, Validate(text)
}

// CurrentText displays the saved guidance in this one file, including shared
// guidance. Personal editing still uses PersonalText to protect the shared block.
func CurrentText(raw string) (string, error) {
	if _, err := managedBlock(raw); err != nil {
		return "", err
	}
	text := strings.ReplaceAll(strings.ReplaceAll(raw, begin, ""), end, "")
	if strings.HasPrefix(text, "---\n") {
		if finish := strings.Index(text[4:], "\n---\n"); finish >= 0 {
			text = text[4+finish+5:]
		}
	}
	return strings.TrimSpace(text), nil
}
