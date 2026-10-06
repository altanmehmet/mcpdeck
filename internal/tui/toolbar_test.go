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
