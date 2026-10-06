package tui

import (
	"fmt"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/syncer"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.instructionEditor != nil {
		if _, sizing := msg.(tea.WindowSizeMsg); !sizing {
			return m.updateInstructions(msg)
		}
	}
	switch event := msg.(type) {
	case clipboardResultMsg:
		if event.err != nil {
			m.status = "Clipboard copy failed; check whether your terminal allows OSC 52 clipboard access."
		} else {
			m.status = "Copy sent to terminal clipboard. Allow OSC 52 in terminal settings if paste does not work."
		}
		return m, nil
	case installFinishedMsg:
		m.reload()
		m.status = "Setup finished. Restart or reload the agent MCP connections."
		if event.err != nil {
			m.status = "Setup was cancelled or failed. Review the planner output and try again."
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.width, m.height = event.Width, event.Height
		m.keepVisible()
		return m, nil
	}
	if m.width < 64 || m.height < 18 {
		if key, ok := msg.(tea.KeyMsg); ok && (key.String() == "q" || key.String() == "ctrl+c") {
			return m, tea.Quit
		}
		return m, nil
	}
	if m.busy {
		return m, nil
	}
	if m.removing != "" {
		return m.updateRemove(msg)
	}
	if m.form != nil {
		return m.updateForm(msg)
	}
	if m.keyEditor != nil {
		return m.updateKeys(msg)
	}
	if mouse, ok := msg.(tea.MouseMsg); ok {
		return m.mouse(mouse)
	}
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "ctrl+c" {
		return m, tea.Quit
	}
	if m.help {
		if key, ok := msg.(tea.KeyMsg); ok {
			if m.boundAction(key.String()) == "select_text" {
				return m.toggleTextSelection()
			}
			if key.String() == "a" {
				m.keyEditor = &keyEditor{}
				return m, nil
			}
			if key.String() == "ctrl+c" || key.String() == "q" {
				return m, tea.Quit
			}
			if key.String() == "esc" || key.String() == "?" {
				m.help = false
			}
			if key.String() == "right" || key.String() == "l" {
				m.helpPage = 1
			}
			if key.String() == "left" || key.String() == "h" {
				m.helpPage = 0
			}
		}
		return m, nil
	}
	if key, ok := msg.(tea.KeyMsg); ok {
		switch action := m.boundAction(key.String()); action {
		case "remove":
			if len(m.servers) > 0 {
				m.removing = m.servers[m.cursor]
			}
		case "install":
			return m.startInstall()
		case "repair":
			if len(m.servers) == 0 {
				m.status = "Select an MCP to repair."
				break
			}
			name := m.servers[m.cursor]
			if _, external := m.external[name]; external {
				m.status = "External MCP: import it into MCPDeck before repair."
				break
			}
			return m.startRepairRequest(name, "")
		case "add":
			m.form = &importForm{}
		case "instructions":
			m.openInstructions()
		case "help":
			m.help = true
		case "assign_keys":
			m.keyEditor = &keyEditor{}
		case "reload":
			m.reload()
		case "quit":
			return m, tea.Quit
		case "switch_profile":
			if len(m.profiles) > 0 {
				m.tab = (m.tab + 1) % len(m.profiles)
			}
		case "move_down":
			if m.cursor < len(m.servers)-1 {
				m.cursor++
			}
		case "move_up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "toggle":
			if len(m.servers) > 0 {
				if _, external := m.external[m.servers[m.cursor]]; external {
					m.status = "External MCP: settings stay in its original agents. Use Remove to delete it."
					break
				}
			}
			if len(m.profiles) > 0 && len(m.servers) > 0 {
				m.change(func(d *model.Deck) error { d.Toggle(m.profiles[m.tab], m.servers[m.cursor]); return nil })
			}
		case "enable_all", "disable_all":
			if len(m.servers) > 0 {
				if _, external := m.external[m.servers[m.cursor]]; external {
					m.status = "External MCP: use Remove to remove it from its original agents."
					break
				}
			}
			if len(m.profiles) > 0 && len(m.servers) > 0 {
				enabled := m.boundAction(key.String()) == "enable_all"
				m.changeProfiles(func(d *model.Deck) error {
					return d.SetEnabled("", m.servers[m.cursor], enabled)
				}, true)
			}
		case "mode":
			if len(m.profiles) > 0 {
				m.change(func(d *model.Deck) error {
					k := m.profiles[m.tab]
					p := d.Profiles[k]
					if p.Mode == "bridge" {
						p.Mode = "direct"
					} else {
						p.Mode = "bridge"
					}
					d.Profiles[k] = p
					return nil
				})
			}
		case "select_text":
			return m.toggleTextSelection()
		case "sync", "retry_sync":
			d, err := m.store.Load()
			if err == nil {
				m.deck = d
				sy := syncer.Syncer{Deck: d, ConfigPath: m.store.Path}
				if action == "retry_sync" {
					var results []syncer.SyncResult
					results, err = sy.RetryFailed()
					m.status = fmt.Sprintf("Retried %d failed targets.", len(results))
				} else {
					err = sy.SyncAll()
					m.status = "All agent profiles synced."
				}
			}
			if err != nil {
				m.status = err.Error()
			}
		}
	}
	m.keepVisible()
	return m, nil
}

func (m Model) toggleTextSelection() (tea.Model, tea.Cmd) {
	m.textSelection = !m.textSelection
	m.copyStart = nil
	if m.textSelection {
		m.status = "Text selection on: drag with the mouse to copy. Press v to restore mouse controls."
		if m.mouseCapture {
			return m, nil
		}
		m.mouseCapture = true
		return m, func() tea.Msg { return tea.EnableMouseCellMotion() }
	}
	if !m.mouseCapture {
		m.mouseCapture = true
		m.status = "Mouse controls on."
		return m, func() tea.Msg { return tea.EnableMouseCellMotion() }
	}
	m.status = "Mouse controls on."
	return m, nil
}

func (m *Model) reload() {
	d, err := m.store.Load()
	if err != nil {
		m.status = err.Error()
		return
	}
	fresh := New(d, m.store)
	m.deck = d
	m.servers = fresh.servers
	m.external = fresh.external
	m.discoveryWarning = fresh.discoveryWarning
	m.profiles = fresh.profiles
	m.cursor = min(m.cursor, max(0, len(m.servers)-1))
	m.tab = min(m.tab, max(0, len(m.profiles)-1))
	m.keepVisible()
}
func (m *Model) keepVisible() {
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+m.rows() {
		m.offset = m.cursor - m.rows() + 1
	}
	if m.tab < m.profileOffset {
		m.profileOffset = m.tab
	}
	if m.tab >= m.profileOffset+m.rows() {
		m.profileOffset = m.tab - m.rows() + 1
	}
}
func (m Model) mouse(event tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.width < 64 || m.height < 18 {
		return m, nil
	}
	if m.textSelection {
		if event.Action == tea.MouseActionPress && event.Button == tea.MouseButtonLeft {
			m.copyStart = &mousePoint{x: event.X, y: event.Y}
			return m, nil
		}
		if event.Action == tea.MouseActionRelease && m.copyStart != nil {
			end := mousePoint{x: event.X, y: event.Y}
			text := selectionText(m.View(), *m.copyStart, end, m.width, m.height)
			m.copyStart = nil
			if text == "" {
				m.status = "No text selected. Drag across text to copy it."
				return m, nil
			}
			m.status = "Copying selection..."
			return m, copyToTerminalClipboard(text)
		}
		return m, nil
	}
	if m.help {
		if event.Action == tea.MouseActionPress && event.Button == tea.MouseButtonLeft && event.Y == 2 {
			switch {
			case event.X < 13:
				m.helpPage = 0
			case event.X < 24:
				m.helpPage = 1
			case event.X < 39:
				m.keyEditor = &keyEditor{}
			default:
				m.help = false
			}
		}
		return m, nil
	}
	if event.Button == tea.MouseButtonWheelUp || event.Button == tea.MouseButtonWheelDown {
		delta := 1
		if event.Button == tea.MouseButtonWheelUp {
			delta = -1
		}
		if event.X < sidebarWidth {
			m.tab = max(0, min(len(m.profiles)-1, m.tab+delta))
		} else {
			m.cursor = max(0, min(len(m.servers)-1, m.cursor+delta))
		}
		m.keepVisible()
		return m, nil
	}
	if event.Action != tea.MouseActionPress || event.Button != tea.MouseButtonLeft {
		return m, nil
	}
	for _, button := range m.buttons() {
		if event.Y == button.y && event.X >= button.x && event.X < button.x+button.w && event.X < m.width {
			actions := map[string]string{"Install MCP": "install", "Install": "install", "Repair": "repair", "+ New MCP": "add", "Instructions": "instructions", "Refresh": "reload", "Sync": "sync", "Retry": "retry_sync", "Help": "help", "Exit": "quit", "Toggle": "toggle", "Enable all": "enable_all", "Disable all": "disable_all", "All on": "enable_all", "All off": "disable_all", "Mode": "mode", "Remove": "remove"}
			if id := actions[button.action]; id != "" {
				return m.Update(keyMessage(m.keys.Bindings[id]))
			}
			return m, nil
		}
	}
	row := event.Y - firstRow
	if row < 0 || row >= m.rows() {
		return m, nil
	}
	if event.X < sidebarWidth-1 {
		index := m.profileOffset + row
		if index < len(m.profiles) {
			m.tab = index
		}
	} else if event.X >= sidebarWidth && event.X < sidebarWidth+m.listWidth() {
		index := m.offset + row
		if index < len(m.servers) {
			m.cursor = index
			if event.X >= sidebarWidth+1 && event.X <= sidebarWidth+3 {
				return m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
			}
		}
	}
	return m, nil
}
