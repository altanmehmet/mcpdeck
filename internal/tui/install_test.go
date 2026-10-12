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

func TestNewMCPNameAloneContinues(t *testing.T) {
	m := mouseFixture(t)
	m.form = &importForm{name: "sentry"}
	updated, command := m.updateForm(tea.KeyMsg{Type: tea.KeyEnter})
	if command == nil || updated.(Model).form != nil {
		t.Fatal("name-only form did not continue")
	}
	m.form = &importForm{name: "  "}
	updated, command = m.startAgenticAdd()
	if command != nil || updated.(Model).form.message == "" {
		t.Fatal("empty MCP name accepted")
	}
}
