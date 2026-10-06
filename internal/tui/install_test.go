package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"testing"
)

func TestInstallButtonAndKeyboard(t *testing.T) {
	m := mouseFixture(t)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if cmd == nil {
		t.Fatal("installation keyboard action absent")
	}
	found := false
	for _, b := range m.buttons() {
		if b.action == "Install MCP" {
			found = true
			_, cmd = m.Update(tea.MouseMsg{X: b.x + 1, Y: b.y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
			if cmd == nil {
				t.Fatal("install click absent")
			}
		}
	}
	if !found {
		t.Fatal("install button absent")
	}
}
