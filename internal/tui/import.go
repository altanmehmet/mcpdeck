package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// importForm is the primary, agent-led MCP creation flow. Configuration paste
// remains available through the import CLI command; the UI starts from intent.
type importForm struct {
	name, request, message string
	field                  int
}

func (m Model) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	f := m.form
	if mouse, ok := msg.(tea.MouseMsg); ok {
		if mouse.Action != tea.MouseActionPress || mouse.Button != tea.MouseButtonLeft {
			return m, nil
		}
		if mouse.Y == 2 {
			if mouse.X < 13 {
				m.form = nil
				return m, nil
			}
			if mouse.X >= 14 {
				return m.startAgenticAdd()
			}
		}
		if mouse.Y >= 6 && mouse.Y <= 8 {
			f.field = 0
		} else if mouse.Y >= 10 && mouse.Y < m.height-5 {
			f.field = 1
		}
		return m, nil
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.form = nil
		return m, nil
	case "ctrl+s":
		return m.startAgenticAdd()
	case "tab", "shift+tab":
		f.field = 1 - f.field
		return m, nil
	case "enter":
		if f.field == 0 {
			f.field = 1
		} else {
			f.request += "\n"
		}
		return m, nil
	case "ctrl+u":
		if f.field == 0 {
			f.name = ""
		} else {
			f.request = ""
		}
		return m, nil
	case "backspace", "ctrl+h":
		if f.field == 0 {
			f.name = trimLastRune(f.name)
		} else {
			f.request = trimLastRune(f.request)
		}
		return m, nil
	}
	if key.Type == tea.KeyRunes || key.Type == tea.KeySpace {
		text := string(key.Runes)
		if key.Type == tea.KeySpace {
			text = " "
		}
		if strings.ContainsAny(text, "\r\n") {
			text = strings.NewReplacer("\r", " ", "\n", " ").Replace(text)
		}
		if f.field == 0 {
			if len([]rune(f.name))+len([]rune(text)) <= 40 {
				f.name += text
			}
		} else if len([]rune(f.request))+len([]rune(text)) <= 4096 {
			f.request += text
		}
	}
	return m, nil
}

func trimLastRune(s string) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return s
	}
	return string(runes[:len(runes)-1])
}

func (m Model) startAgenticAdd() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.form.name)
	request := strings.TrimSpace(m.form.request)
	if name == "" || request == "" {
		m.form.message = "Enter an MCP name and describe what you want to connect."
		return m, nil
	}
	return m.startInstallRequest(name, request)
}

func (m Model) formView() string {
	f := m.form
	lines := make([]string, m.height)
	lines[0] = title.Render(" MCPDECK / NEW MCP")
	lines[1] = muted.Render("Describe the connection you need. No setup URL required.")
	lines[2] = " [ Cancel ] [ Continue · choose agent ]"
	lines[3] = " " + plain(f.message)
	inner := max(20, m.width-8)
	lines[4] = muted.Render(" MCP NAME  ·  Example: sentry, oracle, figma")
	lines[5] = ""
	lines[6] = " ┌" + strings.Repeat("─", inner+2) + "┐"
	name := plain(f.name)
	if f.field == 0 {
		name += "▌"
	}
	nameLine := " │ " + fit(name, inner) + " │"
	if f.field == 0 {
		lines[7] = selected.Render(nameLine)
	} else {
		lines[7] = nameLine
	}
	lines[8] = " └" + strings.Repeat("─", inner+2) + "┘"
	lines[9] = muted.Render(" WHAT SHOULD THIS MCP DO?")
	lines[10] = " ┌" + strings.Repeat("─", inner+2) + "┐"
	requestLines := wrapRequest(plain(f.request), inner)
	visible := max(2, m.height-16)
	start := max(0, len(requestLines)-visible)
	for row := 0; row < visible; row++ {
		line := ""
		if start+row < len(requestLines) {
			line = requestLines[start+row]
		}
		if f.field == 1 && row == min(visible-1, len(requestLines)-1-start) {
			line += "▌"
		}
		value := " │ " + fit(line, inner) + " │"
		if f.field == 1 {
			lines[11+row] = selected.Render(value)
		} else {
			lines[11+row] = value
		}
	}
	bottom := 11 + visible
	lines[bottom] = " └" + strings.Repeat("─", inner+2) + "┘"
	lines[m.height-4] = muted.Render(" NEXT  Choose a planner agent, review its sources and setup steps, then approve.")
	lines[m.height-3] = muted.Render(" Tab: switch field   Enter: next line   Ctrl+U: clear   Esc: cancel")
	lines[m.height-2] = muted.Render(" Ctrl+S: continue  ·  Request is sent to the planner; secrets are entered later.")
	for i, line := range lines {
		lines[i] = ansi.Truncate(line, m.width, "")
	}
	return strings.Join(lines, "\n")
}

func wrapRequest(value string, width int) []string {
	if value == "" {
		return []string{""}
	}
	var result []string
	for _, paragraph := range strings.Split(value, "\n") {
		if paragraph == "" {
			result = append(result, "")
			continue
		}
		line := ""
		for _, word := range strings.Fields(paragraph) {
			candidate := word
			if line != "" {
				candidate = line + " " + word
			}
			if line != "" && ansi.StringWidth(candidate) > width {
				result = append(result, line)
				line = word
			} else {
				line = candidate
			}
		}
		result = append(result, line)
	}
	return result
}
