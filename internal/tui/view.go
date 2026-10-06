package tui

import (
	"fmt"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"unicode"
)

const sidebarWidth = 21
const firstRow = 5

type hit struct {
	x, y, w int
	action  string
	index   int
}

func (m Model) rows() int { return max(1, m.height-10) }
func (m Model) listWidth() int {
	if m.width >= 110 {
		return 39
	}
	return max(1, m.width-sidebarWidth)
}
func (m Model) buttons() []hit {
	result := []hit{}
	x := 1
	for _, a := range []string{"+ New MCP", "Instructions", "Refresh", "Sync", "Help", "Exit", "Remove", "Retry"} {
		result = append(result, hit{x, 2, len(a) + 4, a, 0})
		x += len(a) + 5
	}
	x = 1
	controls := []string{"Toggle", "Enable all", "Disable all", "Mode", "Install MCP", "Repair"}
	if m.width < 76 {
		controls = []string{"Toggle", "All on", "All off", "Mode", "Install", "Repair"}
	}
	for _, a := range controls {
		result = append(result, hit{x, 3, len(a) + 4, a, 0})
		x += len(a) + 5
	}
	return result
}
func plain(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, value)
}
func fit(value string, width int) string {
	value = ansi.Truncate(value, max(0, width), "")
	return value + strings.Repeat(" ", max(0, width-lipgloss.Width(value)))
}
func (m Model) View() string {
	if m.width < 64 || m.height < 18 {
		return "MCPDeck: resize terminal to at least 64 x 18.\nq: quit"
	}
	if m.instructionEditor != nil {
		return m.instructionsView()
	}
	if m.removing != "" {
		return m.removeView()
	}
	if m.form != nil {
		return m.formView()
	}
	if m.keyEditor != nil {
		return m.keyEditorView()
	}
	if m.help {
		return m.helpView()
	}
	lines := make([]string, m.height)
	lines[0] = title.Render(" MCPDECK") + muted.Render("  /  one connection, every coding agent")
	lines[1] = muted.Render(" Click to select · scroll to browse   ● detected  ○ unavailable")
	for _, b := range m.buttons() {
		if len(lines[b.y]) == 0 {
			lines[b.y] = strings.Repeat(" ", b.x)
		} else {
			lines[b.y] += " "
		}
		if b.action == "+ New MCP" || b.action == "Install" {
			lines[b.y] += primaryButton.Render(b.action)
		} else {
			lines[b.y] += quietButton.Render(b.action)
		}
	}
	lines[4] = muted.Render(fit(" AGENTS", sidebarWidth) + fit(" MCP SERVERS", m.listWidth()))
	if m.width >= 110 {
		lines[4] += muted.Render(" DETAILS")
	}
	for row := 0; row < m.rows(); row++ {
		left := ""
		pi := m.profileOffset + row
		if pi < len(m.profiles) {
			indicator := "○"
			if store.AgentDetected(m.profiles[pi], m.deck.Profiles[m.profiles[pi]]) {
				indicator = "●"
			}
			left = " " + indicator + " " + plain(m.profiles[pi])
			if pi == m.tab {
				left = selected.Render(">" + indicator + " " + plain(m.profiles[pi]))
			}
		}
		center := ""
		si := m.offset + row
		if si < len(m.servers) {
			name := m.servers[si]
			mark := "[ ]"
			if len(m.profiles) > 0 && m.deck.IsEnabled(m.profiles[m.tab], name) {
				mark = "[x]"
			}
			center = " " + mark + " " + plain(name)
			if item, external := m.external[name]; external {
				mark = "[ ]"
				for _, profile := range item.Profiles {
					if len(m.profiles) > 0 && profile == m.profiles[m.tab] {
						mark = "[x]"
					}
				}
				center = " " + mark + " " + plain(name) + " · external"
			}
			if si == m.cursor {
				center = selected.Render(center)
			}
		} else if row == 0 && len(m.servers) == 0 {
			center = " No MCP servers yet. Select + New MCP to start."
		}
		lines[firstRow+row] = fit(left, sidebarWidth-1) + muted.Render("│") + fit(center, m.listWidth())
	}
	if m.width >= 110 && len(m.servers) > 0 && m.cursor < len(m.servers) {
		name := m.servers[m.cursor]
		cfg := m.deck.Servers[name]
		kind := "Local process"
		if cfg.URL != "" {
			kind = "Remote MCP"
		}
		count := 0
		for p := range m.deck.Profiles {
			if m.deck.IsEnabled(p, name) {
				count++
			}
		}
		details := []string{plain(name), kind, "", fmt.Sprintf("Enabled for %d of %d agents", count, len(m.profiles)), "", "Toggle: selected agent", "Enable/disable all: every agent", "", "Enabled means configured;", "it does not confirm a live connection."}
		if item, external := m.external[name]; external {
			details = []string{plain(name), "External MCP", "", "Found in:", plain(strings.Join(item.Profiles, ", ")), "", "Added outside MCPDeck.", "Remove: original agents", "Settings are backed up.", "Presence does not confirm", "a live connection."}
		}
		for i, line := range details {
			if i < m.rows() {
				lines[firstRow+i] += muted.Render("│ ") + line
			}
		}
	}
	mode := "direct"
	profile := "no agent"
	if len(m.profiles) > 0 {
		profile = m.profiles[m.tab]
		if m.deck.Profiles[profile].Mode == "bridge" {
			mode = "bridge"
		}
	}
	lines[m.height-5] = muted.Render(strings.Repeat("─", m.width))
	lines[m.height-4] = fmt.Sprintf(" %s | %s mode | %d MCP servers | %d agents", plain(profile), mode, len(m.servers), len(m.profiles))
	lines[m.height-3] = " " + plain(m.status)
	if m.status == "" && m.discoveryWarning != "" {
		lines[m.height-3] = " " + m.discoveryWarning
	}
	if m.busy {
		lines[m.height-3] = " Working..."
	}
	lines[m.height-2] = muted.Render(fmt.Sprintf(" %s: agents  %s/%s: servers  %s: toggle  %s/%s: all", displayKey(m.keys.Bindings["switch_profile"]), displayKey(m.keys.Bindings["move_up"]), displayKey(m.keys.Bindings["move_down"]), displayKey(m.keys.Bindings["toggle"]), displayKey(m.keys.Bindings["enable_all"]), displayKey(m.keys.Bindings["disable_all"])))
	lines[m.height-1] = muted.Render(fmt.Sprintf(" %s: new MCP  %s: instructions  %s: repair  %s: refresh  %s: sync  %s: help  %s: quit", displayKey(m.keys.Bindings["add"]), displayKey(m.keys.Bindings["instructions"]), displayKey(m.keys.Bindings["repair"]), displayKey(m.keys.Bindings["reload"]), displayKey(m.keys.Bindings["sync"]), displayKey(m.keys.Bindings["help"]), displayKey(m.keys.Bindings["quit"])))
	for i, line := range lines {
		lines[i] = ansi.Truncate(line, m.width, "")
	}
	return strings.Join(lines, "\n")
}
func (m Model) helpView() string {
	lines := []string{" MCPDECK / HELP", "", " [ Guide ] [ Shortcuts ] [ Rebind ] [ Back ]", ""}
	if m.helpPage == 0 {
		lines = append(lines, " ADD AN MCP", "  Select + New MCP and enter its name, such as Sentry or Oracle.", "  Describe what you need in plain English. A connected planner agent", "  researches public documentation and prepares a setup plan.", "  Review sources, commands, and configuration. Approve to install.", "  MCPDeck probes initialize and tools/list, then offers agent distribution.", "  API keys are entered privately when the plan requires them.", "", " OTHER WAYS TO ADD", "  Import an existing config with: mcpdeck import --preview", "  Run the app with: mcpdeck", "", " The left list shows agents; the center shows MCP servers. Click a row", " to enable or disable it for the selected agent. Scroll to browse.", " Press v, then drag across text to copy it to your clipboard.", " Press v again to restore MCPDeck mouse controls.", " Select Remove or press x to remove an MCP from all agents.", " MCPs added directly to agent settings appear as external.", " Enabled means configured; use connection verification to test it.")
	} else {
		lines = []string{" MCPDECK / HELP", " [ Guide ] [ Shortcuts ] [ Rebind ] [ Back ]", " KEYBOARD SHORTCUTS"}
		for _, b := range keyBindings {
			lines = append(lines, "  "+displayKey(m.keys.Bindings[b.ID])+": "+b.Label)
		}
		lines = append(lines, " a: rebind a shortcut | Esc: return to the main screen")
	}
	if m.helpPage == 0 {
		lines = append(lines, "", " ←/→ switch help page  a: rebind shortcut  Esc: return")
	}
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], m.width, "")
	}
	return strings.Join(lines[:min(len(lines), m.height)], "\n")
}
