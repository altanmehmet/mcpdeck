package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/altanmehmet/mcpdeck/internal/store"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

type keyBinding struct {
	ID, Label string
	Default   string
}

var keyBindings = []keyBinding{
	{"quit", "Exit", "q"},
	{"help", "Help", "?"},
	{"assign_keys", "Assign keys", "a"},
	{"install", "Install MCP", "i"},
	{"repair", "Repair/update selected MCP", "u"},
	{"instructions", "Personal instructions for all agents", "g"},
	{"add", "New MCP with agent", "n"},
	{"reload", "Refresh", "r"},
	{"remove", "Remove MCP from all agents", "x"},
	{"sync", "Sync agent settings", "s"},
	{"retry_sync", "Retry failed sync targets", "ctrl+r"},
	{"switch_profile", "Next agent", "tab"},
	{"move_up", "Previous MCP", "up"},
	{"move_down", "Next MCP", "down"},
	{"toggle", "Toggle selected agent", "space"},
	{"enable_all", "Enable all agents", "e"},
	{"disable_all", "Disable all agents", "d"},
	{"mode", "Direct/Bridge mode", "b"},
	{"select_text", "Select text with mouse", "v"},
}

type keyPreferences struct {
	Bindings map[string]string `json:"bindings"`
}

type keyEditor struct {
	selected  int
	offset    int
	capturing bool
	message   string
}

func defaultKeyPreferences() keyPreferences {
	p := keyPreferences{Bindings: map[string]string{}}
	for _, b := range keyBindings {
		p.Bindings[b.ID] = b.Default
	}
	return p
}

func keyPreferencesPath(s store.Store) string { return filepath.Join(filepath.Dir(s.Path), "ui.json") }

func readKeyPreferences(s store.Store) (keyPreferences, error) {
	p := defaultKeyPreferences()
	raw, err := os.ReadFile(keyPreferencesPath(s))
	if os.IsNotExist(err) {
		return p, nil
	}
	if err != nil {
		return p, err
	}
	if len(raw) > 64*1024 {
		return p, fmt.Errorf("UI settings exceed 64 KiB")
	}
	var saved keyPreferences
	if err = json.Unmarshal(raw, &saved); err != nil || saved.Bindings == nil {
		return p, fmt.Errorf("invalid UI settings")
	}
	for id, key := range saved.Bindings {
		known := false
		for _, b := range keyBindings {
			if b.ID == id {
				known = true
				break
			}
		}
		if !known {
			return p, fmt.Errorf("unknown key binding")
		}
		p.Bindings[id] = normalizeKey(key)
	}
	// Preserve existing custom shortcuts when a newly added action uses their key.
	if _, savedRemove := saved.Bindings["remove"]; !savedRemove {
		for _, candidate := range []string{"x", "delete", "ctrl+x", "ctrl+d", "ctrl+e", "ctrl+f", "ctrl+g", "ctrl+h", "ctrl+j", "ctrl+k", "ctrl+l", "ctrl+n", "ctrl+o", "ctrl+p", "ctrl+r", "ctrl+s", "ctrl+t", "ctrl+u", "ctrl+w", "ctrl+y", "ctrl+z"} {
			used := false
			for id, key := range p.Bindings {
				if id != "remove" && key == candidate {
					used = true
				}
			}
			if !used {
				p.Bindings["remove"] = candidate
				break
			}
		}
	}
	// New actions must not take an existing custom shortcut.
	for _, binding := range keyBindings {
		if _, exists := saved.Bindings[binding.ID]; exists {
			continue
		}
		used := false
		for id, key := range p.Bindings {
			if id != binding.ID && key == p.Bindings[binding.ID] {
				used = true
			}
		}
		if !used {
			continue
		}
		for _, candidate := range []string{"g", "u", "x", "ctrl+g", "ctrl+u", "ctrl+x", "ctrl+j", "ctrl+k", "ctrl+l", "ctrl+n", "ctrl+o", "ctrl+p", "ctrl+s", "ctrl+t", "ctrl+w", "ctrl+y", "ctrl+z"} {
			available := true
			for id, key := range p.Bindings {
				if id != binding.ID && key == candidate {
					available = false
				}
			}
			if available {
				p.Bindings[binding.ID] = candidate
				break
			}
		}
	}
	if err = validateKeyPreferences(p); err != nil {
		return defaultKeyPreferences(), err
	}
	return p, nil
}

func saveKeyPreferences(s store.Store, p keyPreferences) error {
	if err := validateKeyPreferences(p); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return store.AtomicWrite(keyPreferencesPath(s), append(raw, '\n'))
}

func validateKeyPreferences(p keyPreferences) error {
	seen := map[string]bool{}
	for _, b := range keyBindings {
		key := normalizeKey(p.Bindings[b.ID])
		if !validKey(key) {
			return fmt.Errorf("invalid key")
		}
		if key == "ctrl+c" {
			return fmt.Errorf("Ctrl+C is reserved for quitting")
		}
		if seen[key] {
			return fmt.Errorf("a key cannot be assigned to more than one action")
		}
		seen[key] = true
	}
	return nil
}

func normalizeKey(key string) string {
	switch strings.ToLower(key) {
	case " ":
		return "space"
	case "space", "spacebar":
		return "space"
	case "esc", "escape":
		return "esc"
	case "return":
		return "enter"
	default:
		return strings.ToLower(key)
	}
}

func validKey(key string) bool {
	if key == "" || key == "ctrl+c" {
		return false
	}
	switch key {
	case "tab", "enter", "space", "up", "down", "left", "right", "backspace", "delete", "home", "end", "pgup", "pgdown", "ctrl+a", "ctrl+b", "ctrl+d", "ctrl+e", "ctrl+f", "ctrl+g", "ctrl+h", "ctrl+i", "ctrl+j", "ctrl+k", "ctrl+l", "ctrl+m", "ctrl+n", "ctrl+o", "ctrl+p", "ctrl+q", "ctrl+s", "ctrl+t", "ctrl+u", "ctrl+v", "ctrl+w", "ctrl+x", "ctrl+y", "ctrl+z":
		return true
	}
	return len([]rune(key)) == 1 && key != " " && !strings.ContainsAny(key, "\x00\n\r")
}

func displayKey(key string) string {
	switch key {
	case "space":
		return "Space"
	case "tab":
		return "Tab"
	case "enter":
		return "Enter"
	case "esc":
		return "Esc"
	case "up":
		return "↑"
	case "down":
		return "↓"
	case "left":
		return "←"
	case "right":
		return "→"
	default:
		return key
	}
}

func (m Model) boundAction(key string) string {
	key = normalizeKey(key)
	for _, b := range keyBindings {
		if m.keys.Bindings[b.ID] == key {
			return b.ID
		}
	}
	return ""
}

func keyMessage(key string) tea.KeyMsg {
	switch key {
	case "space":
		return tea.KeyMsg{Type: tea.KeySpace}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case "delete":
		return tea.KeyMsg{Type: tea.KeyDelete}
	case "home":
		return tea.KeyMsg{Type: tea.KeyHome}
	case "end":
		return tea.KeyMsg{Type: tea.KeyEnd}
	case "pgup":
		return tea.KeyMsg{Type: tea.KeyPgUp}
	case "pgdown":
		return tea.KeyMsg{Type: tea.KeyPgDown}
	default:
		if strings.HasPrefix(key, "ctrl+") && len(key) == 6 {
			types := []tea.KeyType{tea.KeyCtrlA, tea.KeyCtrlB, tea.KeyCtrlC, tea.KeyCtrlD, tea.KeyCtrlE, tea.KeyCtrlF, tea.KeyCtrlG, tea.KeyCtrlH, tea.KeyCtrlI, tea.KeyCtrlJ, tea.KeyCtrlK, tea.KeyCtrlL, tea.KeyCtrlM, tea.KeyCtrlN, tea.KeyCtrlO, tea.KeyCtrlP, tea.KeyCtrlQ, tea.KeyCtrlR, tea.KeyCtrlS, tea.KeyCtrlT, tea.KeyCtrlU, tea.KeyCtrlV, tea.KeyCtrlW, tea.KeyCtrlX, tea.KeyCtrlY, tea.KeyCtrlZ}
			letter := key[5]
			if letter >= 'a' && letter <= 'z' && letter != 'c' {
				return tea.KeyMsg{Type: types[letter-'a']}
			}
		}
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
}

func (m Model) keyEditorView() string {
	lines := make([]string, m.height)
	lines[0] = title.Render(" MCPDECK / TUS ATAMA")
	lines[1] = " Select an action, then press Enter or click it to record a key."
	lines[2] = " [ Back ] [ Restore defaults ]"
	lines[3] = " " + plain(m.keyEditor.message)
	if m.keyEditor.capturing {
		lines[3] = " Press the key to assign. Esc cancels."
	}
	start := 4
	rows := max(1, m.height-9)
	for m.keyEditor.selected < m.keyEditor.offset {
		m.keyEditor.offset = m.keyEditor.selected
	}
	if m.keyEditor.selected >= m.keyEditor.offset+rows {
		m.keyEditor.offset = m.keyEditor.selected - rows + 1
	}
	for i := 0; i < rows; i++ {
		index := m.keyEditor.offset + i
		if index >= len(keyBindings) {
			break
		}
		b := keyBindings[index]
		line := fmt.Sprintf(" %-28s [ %s ]", b.Label, displayKey(m.keys.Bindings[b.ID]))
		if index == m.keyEditor.selected {
			line = selected.Render(">" + line)
		}
		lines[start+i] = line
	}
	lines[m.height-2] = muted.Render(" ↑/↓: select action  Enter: record key  Esc: back  Ctrl+R: restore defaults")
	lines[m.height-1] = muted.Render(" Key bindings are stored in this computer's MCPDeck settings.")
	for i, line := range lines {
		lines[i] = ansi.Truncate(line, m.width, "")
	}
	return strings.Join(lines, "\n")
}

func (m Model) updateKeys(msg tea.Msg) (tea.Model, tea.Cmd) {
	if mouse, ok := msg.(tea.MouseMsg); ok {
		if mouse.Button == tea.MouseButtonWheelUp || mouse.Button == tea.MouseButtonWheelDown {
			delta := 1
			if mouse.Button == tea.MouseButtonWheelUp {
				delta = -1
			}
			m.keyEditor.selected = max(0, min(len(keyBindings)-1, m.keyEditor.selected+delta))
			return m, nil
		}
		if mouse.Action != tea.MouseActionPress || mouse.Button != tea.MouseButtonLeft {
			return m, nil
		}
		if mouse.Y == 2 && mouse.X < 10 {
			m.keyEditor = nil
			return m, nil
		}
		if mouse.Y == 2 && mouse.X >= 11 && mouse.X < 33 {
			return m.resetKeys()
		}
		visibleRow := mouse.Y - 4
		row := visibleRow + m.keyEditor.offset
		if visibleRow >= 0 && visibleRow < max(1, m.height-9) && row < len(keyBindings) {
			m.keyEditor.selected = row
			m.keyEditor.capturing = true
			m.keyEditor.message = ""
		}
		return m, nil
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	if m.keyEditor.capturing {
		if key.String() == "esc" {
			m.keyEditor.capturing = false
			m.keyEditor.message = "Key recording cancelled."
			return m, nil
		}
		candidate := normalizeKey(key.String())
		if candidate == "ctrl+c" {
			m.keyEditor.capturing = false
			m.keyEditor.message = "Ctrl+C is reserved for quitting."
			return m, nil
		}
		if candidate == "esc" || candidate == "ctrl+r" {
			m.keyEditor.capturing = false
			m.keyEditor.message = "Esc cancels; Ctrl+R restores defaults."
			return m, nil
		}
		m.keyEditor.capturing = false
		b := keyBindings[m.keyEditor.selected]
		updated := keyPreferences{Bindings: map[string]string{}}
		for id, value := range m.keys.Bindings {
			updated.Bindings[id] = value
		}
		updated.Bindings[b.ID] = candidate
		if err := validateKeyPreferences(updated); err != nil {
			m.keyEditor.message = "This key is unsupported or already assigned to another action."
			return m, nil
		}
		if err := saveKeyPreferences(m.store, updated); err != nil {
			m.keyEditor.message = "Could not save the key binding."
			return m, nil
		}
		m.keys = updated
		m.keyEditor.message = b.Label + ": " + displayKey(candidate) + " saved."
		return m, nil
	}
	switch key.String() {
	case "esc", "q":
		m.keyEditor = nil
	case "up", "k":
		m.keyEditor.selected = max(0, m.keyEditor.selected-1)
	case "down", "j":
		m.keyEditor.selected = min(len(keyBindings)-1, m.keyEditor.selected+1)
	case "enter":
		m.keyEditor.capturing = true
		m.keyEditor.message = ""
	case "ctrl+r":
		return m.resetKeys()
	case "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) resetKeys() (tea.Model, tea.Cmd) {
	p := defaultKeyPreferences()
	if err := saveKeyPreferences(m.store, p); err != nil {
		m.keyEditor.message = "Could not save the default keys."
		return m, nil
	}
	m.keys = p
	m.keyEditor.capturing = false
	m.keyEditor.message = "Default key bindings restored."
	return m, nil
}

func keyBindingIDs() []string {
	ids := make([]string, 0, len(keyBindings))
	for _, b := range keyBindings {
		ids = append(ids, b.ID)
	}
	sort.Strings(ids)
	return ids
}
