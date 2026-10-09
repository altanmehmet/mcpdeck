package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"strings"
)

var additionalActions = []struct{ id, label string }{
	{"repair", "Repair / update selected MCP"},
	{"install", "Install from documentation"},
	{"mode", "Switch selected agent's Direct / Bridge mode"},
	{"reload", "Refresh server inventory"},
	{"retry_sync", "Retry failed synchronization"},
	{"assign_keys", "Customize keyboard shortcuts"},
	{"select_text", "Select and copy terminal text"},
	{"remove", "Remove selected MCP from its agents"},
}

func (m Model) actionsView() string {
	lines := make([]string, m.height)
	lines[0] = title.Render(" MCPDECK / MORE ACTIONS")
	lines[1] = muted.Render(" Select an action. Existing keyboard shortcuts also work from the main panel.")
	lines[2] = " [ Back ]"
	for i, action := range additionalActions {
		line := fmt.Sprintf(" %-46s %s", action.label, displayKey(m.keys.Bindings[action.id]))
		if i == m.actionCursor {
			line = selected.Render(">" + line[1:])
		}
		lines[4+i] = line
	}
	lines[m.height-2] = muted.Render(" ↑/↓: select  Enter: open  Esc: back  Ctrl+C: quit")
	for i, line := range lines {
		lines[i] = ansi.Truncate(line, m.width, "")
	}
	return strings.Join(lines, "\n")
}
func (m Model) updateActions(msg tea.Msg) (tea.Model, tea.Cmd) {
	open := func() (tea.Model, tea.Cmd) {
		id := additionalActions[m.actionCursor].id
		m.actionsOpen = false
		return m.Update(keyMessage(m.keys.Bindings[id]))
	}
	if mouse, ok := msg.(tea.MouseMsg); ok {
		if mouse.Button == tea.MouseButtonWheelUp {
			m.actionCursor = max(0, m.actionCursor-1)
		}
		if mouse.Button == tea.MouseButtonWheelDown {
			m.actionCursor = min(len(additionalActions)-1, m.actionCursor+1)
		}
		if mouse.Action == tea.MouseActionPress && mouse.Button == tea.MouseButtonLeft {
			if mouse.Y == 2 {
				m.actionsOpen = false
			} else if i := mouse.Y - 4; i >= 0 && i < len(additionalActions) {
				m.actionCursor = i
				return open()
			}
		}
	}
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc", "q":
			m.actionsOpen = false
		case "up", "k":
			m.actionCursor = max(0, m.actionCursor-1)
		case "down", "j":
			m.actionCursor = min(len(additionalActions)-1, m.actionCursor+1)
		case "enter":
			return open()
		}
	}
	return m, nil
}
