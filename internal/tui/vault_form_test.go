package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"strings"
	"testing"
)

func TestNewMCPVaultToggle(t *testing.T) {
	m := Model{form: &importForm{}, width: 100, height: 26}
	next, _ := m.updateForm(tea.KeyMsg{Type: tea.KeyCtrlK})
	m = next.(Model)
	if !m.form.vault || !strings.Contains(m.formView(), "OS VAULT: on") {
		t.Fatal("vault toggle not visible")
	}
	next, _ = m.updateForm(tea.KeyMsg{Type: tea.KeyCtrlK})
	m = next.(Model)
	if m.form.vault {
		t.Fatal("vault toggle did not restore existing behavior")
	}
}
