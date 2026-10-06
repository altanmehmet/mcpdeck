package tui

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestExternalMCPVisibleAndRemovable(t *testing.T) {
	m := mouseFixture(t)
	path := m.deck.Profiles["cursor"].TargetPath
	if err := os.WriteFile(path, []byte(`{"mcpServers":{"outside":{"command":"custom","secret":"never-display"},"keep":{"command":"echo"}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	m = New(m.deck, m.store)
	if len(m.external["outside"].Profiles) != 1 {
		t.Fatal("external entry missing")
	}
	for i, name := range m.servers {
		if name == "outside" {
			m.cursor = i
		}
	}
	if !strings.Contains(m.View(), "external") || strings.Contains(m.View(), "never-display") {
		t.Fatal("incorrect external view")
	}
	next, _ := m.Update(keyMessage("space"))
	m = next.(Model)
	raw, _ := os.ReadFile(path)
	if !strings.Contains(string(raw), "outside") {
		t.Fatal("toggle altered external entry")
	}
	next, _ = m.Update(keyMessage("x"))
	m = next.(Model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)
	raw, _ = os.ReadFile(path)
	if strings.Contains(string(raw), "outside") || !strings.Contains(string(raw), "keep") {
		t.Fatal("incorrect external removal", m.status)
	}
}
