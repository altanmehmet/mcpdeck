package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/altanmehmet/mcpdeck/internal/syncer"
)

func (m Model) removeView() string {
	lines := make([]string, m.height)
	lines[0] = title.Render(" MCPDECK / REMOVE MCP")
	lines[2] = " Remove " + plain(m.removing) + "?"
	lines[4] = " Removes its MCPDeck record and settings from every configured agent."
	lines[5] = " Installed files and shared software are kept."
	if item, external := m.external[m.removing]; external {
		lines[4] = " Removes the external MCP from: " + plain(strings.Join(item.Profiles, ", "))
		lines[5] = " Original agent settings are backed up. Installed files are kept."
		if m.discoveryWarning != "" {
			lines[6] = " Discovery incomplete. Refresh after fixing unreadable agent settings."
		}
	}
	lines[7] = " [ Remove ] [ Cancel ]"
	lines[9] = muted.Render(" y or Enter: remove   Esc or n: cancel")
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], m.width, "")
	}
	return strings.Join(lines, "\n")
}

func (m Model) updateRemove(msg tea.Msg) (tea.Model, tea.Cmd) {
	confirmed, cancelled := false, false
	switch event := msg.(type) {
	case tea.KeyMsg:
		switch event.String() {
		case "enter", "y":
			confirmed = true
		case "esc", "n":
			cancelled = true
		case "ctrl+c":
			return m, tea.Quit
		}
	case tea.MouseMsg:
		if event.Action == tea.MouseActionPress && event.Button == tea.MouseButtonLeft && event.Y == 7 {
			confirmed = event.X >= 1 && event.X < 11
			cancelled = event.X >= 12 && event.X < 22
		}
	}
	if cancelled {
		m.removing = ""
		m.status = "Removal cancelled."
	}
	if confirmed {
		name := m.removing
		m.removing = ""
		var err error
		if item, external := m.external[name]; external {
			if m.discoveryWarning != "" {
				err = fmt.Errorf("discovery incomplete; fix unreadable agent settings and refresh")
			} else {
				err = syncer.RemoveDiscovered(m.store, item)
			}
		} else {
			err = syncer.Remove(m.store, name)
		}
		m.reload()
		if err != nil {
			m.status = err.Error()
		} else {
			m.status = fmt.Sprintf("%s removed from all configured agents. Restart or reload MCP connections.", name)
		}
	}
	return m, nil
}
