package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func TestToolbarDoesNotAddTerminalRows(t *testing.T) {
	m := mouseFixture(t)
	for _, width := range []int{64, 80, 120} {
		m.width, m.height = width, 30
		lines := strings.Split(m.View(), "\n")
		if len(lines) != m.height {
			t.Fatalf("toolbar overflow: %d rows, expected %d", len(lines), m.height)
		}
		if !strings.Contains(ansi.Strip(lines[0]), "MCPDECK") || !strings.Contains(ansi.Strip(lines[4]), "AGENTS") {
			t.Fatal("toolbar moved surrounding rows")
		}
	}
	for _, style := range []struct {
		name  string
		width int
	}{{primaryButton.Render("+ New MCP"), len("+ New MCP") + 4}, {quietButton.Render("Retry"), len("Retry") + 4}} {
		if strings.Contains(style.name, "\n") || lipgloss.Width(style.name) != style.width {
			t.Fatal("button does not match single-row hit box")
		}
	}
}

func TestCompactToolbarAndActionMenu(t *testing.T) {
	m := mouseFixture(t)
	for _, width := range []int{64, 80, 120} {
		m.width = width
		for _, b := range m.buttons() {
			if b.x+b.w > width {
				t.Fatal("partially visible button remains clickable")
			}
		}
	}
	m.width = 80
	next, _ := m.Update(keyMessage(m.keys.Bindings["actions"]))
	m = next.(Model)
	if !m.actionsOpen || !strings.Contains(m.View(), "MORE ACTIONS") {
		t.Fatal("menu did not open")
	}
	for i, a := range additionalActions {
		if a.id == "remove" {
			m.actionCursor = i
		}
	}
	next, _ = m.Update(keyMessage("enter"))
	m = next.(Model)
	if m.actionsOpen || m.removing != "demo" {
		t.Fatal("removal confirmation bypassed")
	}
	if err := validateKeyPreferences(defaultKeyPreferences()); err != nil {
		t.Fatal("default shortcuts cannot be saved", err)
	}
}
