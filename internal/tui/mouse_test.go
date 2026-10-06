package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"path/filepath"
	"strings"
	"testing"
)

func mouseFixture(t *testing.T) Model {
	t.Helper()
	dir := t.TempDir()
	s := store.Store{Path: filepath.Join(dir, "deck.json")}
	d := &model.Deck{Version: 1, Servers: map[string]model.ServerConfig{"demo": {Command: "echo"}}, Profiles: map[string]model.ProfileConfig{"cursor": {TargetPath: filepath.Join(dir, "cursor.json")}, "codex": {TargetPath: filepath.Join(dir, "codex.toml"), Format: "codex"}}}
	if err := s.Save(d); err != nil {
		t.Fatal(err)
	}
	return New(d, s)
}
func press(m Model, x, y int) Model {
	next, _ := m.Update(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	return next.(Model)
}
func TestMouseSelectionToggleAndToolbar(t *testing.T) {
	m := mouseFixture(t)
	m = press(m, 3, firstRow+1)
	if m.tab != 1 {
		t.Fatal("profile click ignored")
	}
	m = press(m, sidebarWidth+2, firstRow)
	d, err := m.store.Load()
	if err != nil || !d.IsEnabled("codex", "demo") || d.IsEnabled("cursor", "demo") {
		t.Fatal("wrong single-profile toggle")
	}
	next, _ := m.Update(tea.MouseMsg{X: sidebarWidth + 2, Y: firstRow, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease})
	m = next.(Model)
	if !m.deck.IsEnabled("codex", "demo") {
		t.Fatal("release toggled twice")
	}
	for _, button := range m.buttons() {
		if button.action == "Enable all" {
			m = press(m, button.x+1, button.y)
		}
	}
	d, _ = m.store.Load()
	for name := range d.Profiles {
		if !d.IsEnabled(name, "demo") {
			t.Fatal("global button did not sync")
		}
	}
	for _, button := range m.buttons() {
		if button.action == "+ New MCP" {
			m = press(m, button.x+1, button.y)
		}
	}
	if m.form == nil {
		t.Fatal("add button ignored")
	}
}
func TestMouseScrollAndResize(t *testing.T) {
	m := mouseFixture(t)
	for i := 0; i < 50; i++ {
		m.servers = append(m.servers, fmt.Sprintf("server-%02d", i))
	}
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 18})
	m = next.(Model)
	for i := 0; i < 20; i++ {
		next, _ = m.Update(tea.MouseMsg{X: 30, Y: 10, Button: tea.MouseButtonWheelDown})
		m = next.(Model)
	}
	if m.cursor != 20 || m.offset == 0 {
		t.Fatal("scroll did not keep selection visible")
	}
	for _, width := range []int{64, 80, 120} {
		next, _ = m.Update(tea.WindowSizeMsg{Width: width, Height: 18})
		m = next.(Model)
		for _, line := range strings.Split(m.View(), "\n") {
			if lipgloss.Width(line) > width {
				t.Fatal("view overflow")
			}
		}
	}
	next, _ = m.Update(tea.WindowSizeMsg{Width: 30, Height: 10})
	m = next.(Model)
	before := m.tab
	m = press(m, 2, 6)
	if m.tab != before {
		t.Fatal("hidden controls active in small terminal")
	}
}

func TestTextSelectionModeCanBeToggledWithoutLosingKeyboardControl(t *testing.T) {
	m := mouseFixture(t)
	if m.textSelection {
		t.Fatal("mouse controls should start enabled")
	}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = next.(Model)
	if !m.textSelection || cmd != nil {
		t.Fatal("v should enable text selection without stopping mouse capture")
	}
	next, _ = m.Update(tea.MouseMsg{X: 1, Y: 0, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	m = next.(Model)
	next, cmd = m.Update(tea.MouseMsg{X: 6, Y: 0, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease})
	m = next.(Model)
	if cmd == nil || !strings.Contains(m.status, "Copying selection") {
		t.Fatal("dragging in text mode should copy the selected text")
	}
	next, _ = m.Update(clipboardResultMsg{})
	m = next.(Model)
	if !strings.Contains(m.status, "Copy sent to terminal clipboard.") {
		t.Fatal("successful clipboard copy was not reported")
	}
	next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = next.(Model)
	if m.textSelection || cmd != nil {
		t.Fatal("v should restore app mouse controls")
	}
}

func TestSelectionTextHandlesANSIAndMultilineRanges(t *testing.T) {
	got := selectionText("\x1b[31mhello\x1b[0m\nworld", mousePoint{1, 0}, mousePoint{2, 1}, 10, 2)
	if got != "ello\nwor" {
		t.Fatalf("selectionText() = %q, want %q", got, "ello\nwor")
	}
}

func TestNewMCPFormCapturesNameAndNaturalLanguageRequest(t *testing.T) {
	m := mouseFixture(t)
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = next.(Model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("sentry")})
	m = next.(Model)
	if m.form.name != "sentry" {
		t.Fatal("MCP name was not captured")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Track errors in my Sentry project")})
	m = next.(Model)
	if !strings.Contains(m.View(), "Track errors in my Sentry project") {
		t.Fatal("natural-language request was not shown")
	}
	if strings.Contains(plain("bad\x1b[31m\nvalue"), "\x1b") {
		t.Fatal("terminal escape not removed")
	}
}
