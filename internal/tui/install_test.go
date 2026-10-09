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
		if b.action == "More" {
			found = true
			m = press(m, b.x+1, b.y)
		}
	}
	if !found || !m.actionsOpen {
		t.Fatal("more actions button absent")
	}
	for i, a := range additionalActions {
		if a.id == "install" {
			_, cmd = m.Update(tea.MouseMsg{X: 2, Y: 4 + i, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
			if cmd == nil {
				t.Fatal("install menu click absent")
			}
			return
		}
	}
	t.Fatal("install menu action absent")
}
