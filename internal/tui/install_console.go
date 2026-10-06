package tui

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// ConsolePlan intentionally contains no connection credentials or input values.
type ConsolePlan struct {
	Name, Summary, Digest, Transport                         string
	Steps                                                    []ConsoleStep
	Requirements, Inputs, Manual, FollowUp, Sources, Targets []string
}
type ConsoleStep struct{ Description, Command, Directory string }
type ConsoleEvent struct {
	Kind, Text string
	Plan       *ConsolePlan
	Secret     bool
	Err        error
}
type consoleEntry struct{ role, text string }
type consoleTick time.Time

// InstallConsole presents the existing CLI workflow; it never executes recipes.
type InstallConsole struct {
	events                                              <-chan ConsoleEvent
	stopped                                             <-chan struct{}
	submit                                              func(string, bool)
	cancel                                              func()
	width, height                                       int
	provider, stage, prompt                             string
	entries                                             []consoleEntry
	draft, private                                      []rune
	cursor, scroll, planScroll, revision, step          int
	plan                                                *ConsolePlan
	waiting, secret, finished, stopping, fullPlan, help bool
	started                                             time.Time
	now                                                 time.Time
	err                                                 error
	completed                                           int
}

func NewInstallConsole(events <-chan ConsoleEvent, stopped <-chan struct{}, submit func(string, bool), cancel func(), request string) InstallConsole {
	now := time.Now()
	m := InstallConsole{events: events, stopped: stopped, submit: submit, cancel: cancel,
		width: 100, height: 30, provider: "agent", stage: "Preparing", started: now, now: now, step: -1}
	if request != "" {
		m.add("YOU", request)
	}
	return m
}
func (m InstallConsole) listen() tea.Cmd {
	return func() tea.Msg {
		select {
		case event, ok := <-m.events:
			if ok {
				return event
			}
			return nil
		case <-m.stopped:
			return nil
		}
	}
}
func consoleTimer() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return consoleTick(t) })
}
func (m InstallConsole) Init() tea.Cmd { return tea.Batch(m.listen(), consoleTimer()) }
func consolePlain(value string) string {
	value = ansi.Strip(value)
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' || !unicode.IsControl(r) {
			return r
		}
		return -1
	}, value)
}
func (m *InstallConsole) add(role, text string) {
	text = strings.TrimSpace(consolePlain(text))
	if text == "" {
		return
	}
	if len(text) > 16*1024 {
		text = text[:16*1024] + "\n[message truncated]"
	}
	if role == "SYSTEM" && len(m.entries) > 0 && m.entries[len(m.entries)-1].role == role && len(m.entries[len(m.entries)-1].text)+len(text) < 16*1024 {
		m.entries[len(m.entries)-1].text += "\n" + text
		m.scroll = 0
		return
	}
	m.entries = append(m.entries, consoleEntry{role, text})
	if len(m.entries) > 100 {
		m.entries = m.entries[len(m.entries)-100:]
	}
	m.scroll = 0
}
func (m InstallConsole) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width = max(1, msg.Width)
		m.height = max(1, msg.Height)
	case consoleTick:
		if m.finished {
			return m, nil
		}
		m.now = time.Time(msg)
		return m, consoleTimer()
	case ConsoleEvent:
		switch msg.Kind {
		case "provider":
			m.provider = consolePlain(msg.Text)
		case "status":
			m.stage = consolePlain(msg.Text)
			m.cursor = len(m.draft)
			if m.stage == "Verifying connection" && m.plan != nil {
				m.completed = len(m.plan.Steps)
			}
			m.waiting = false
			m.secret = false
			m.private = nil
		case "output":
			m.add("SYSTEM", msg.Text)
		case "plan":
			m.waiting = false
			m.secret = false
			m.private = nil
			m.plan = msg.Plan
			m.revision++
			m.step = -1
			m.completed = 0
			m.planScroll = 0
			m.stage = "Plan ready"
			if msg.Plan != nil {
				m.add("AGENT · "+m.provider, msg.Plan.Summary)
			}
		case "step":
			m.fullPlan = false
			m.stage = "Installing"
			var current, total int
			if _, err := fmt.Sscanf(msg.Text, "Installation step %d/%d:", &current, &total); err == nil {
				m.step = current - 1
				m.completed = max(m.completed, current-1)
			}
			m.add("STEP", msg.Text)
		case "await":
			if m.stopping {
				break
			}
			m.waiting = true
			m.secret = msg.Secret
			m.prompt = consolePlain(msg.Text)
			m.private = nil
			m.cursor = len(m.draft)
			if msg.Secret {
				m.cursor = 0
				m.fullPlan = false
			}
			if strings.Contains(m.prompt, "Install this plan?") || strings.Contains(m.prompt, "Apply this repair?") {
				m.fullPlan = true
				m.planScroll = 0
				m.stage = "Approval required"
			} else if msg.Secret {
				m.stage = "Private input"
			} else {
				m.stage = "Your turn"
			}
		case "done":
			m.fullPlan = false
			m.finished = true
			m.now = time.Now()
			m.waiting = false
			m.secret = false
			m.private = nil
			m.err = msg.Err
			if msg.Err != nil {
				m.stage = "Stopped"
				m.add("RESULT", msg.Err.Error())
			} else {
				m.stage = "Complete"
				m.add("RESULT", "Setup finished. Start a new agent session to load the changes.")
			}
		}
		return m, m.listen()
	case tea.MouseMsg:
		if msg.Button == tea.MouseButtonWheelUp {
			if m.fullPlan {
				m.planScroll = max(0, m.planScroll-3)
			} else {
				m.scroll += 3
			}
		}
		if msg.Button == tea.MouseButtonWheelDown {
			if m.fullPlan {
				m.planScroll += 3
			} else {
				m.scroll = max(0, m.scroll-3)
			}
		}
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft && msg.Y == 2 {
			bar := m.controls()
			for _, button := range []string{"[ Conversation ]", "[ Plan / F2 ]", "[ Help / F1 ]", "[ Review install ]", "[ Approve ]", "[ Cancel ]"} {
				left := strings.Index(bar, button)
				if left < 0 || msg.X < left || msg.X >= left+len(button) {
					continue
				}
				switch button {
				case "[ Conversation ]":
					m.fullPlan = false
				case "[ Plan / F2 ]":
					m.fullPlan = !m.fullPlan
					m.planScroll = 0
				case "[ Help / F1 ]":
					m.help = !m.help
				case "[ Review install ]":
					return m, m.send("/install", false)
				case "[ Approve ]":
					return m, m.send("y", false)
				case "[ Cancel ]":
					return m, m.send("", false)
				}
			}
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			if m.finished {
				return m, tea.Quit
			}
			if !m.stopping {
				m.stopping = true
				m.waiting = false
				m.private = nil
				m.stage = "Stopping"
				m.cancel()
			}
			return m, nil
		case "esc":
			if m.help {
				m.help = false
			} else if m.fullPlan {
				m.fullPlan = false
			} else if m.finished {
				return m, tea.Quit
			}
			return m, nil
		case "f2":
			m.fullPlan = !m.fullPlan
			m.planScroll = 0
			return m, nil
		case "f1":
			m.help = !m.help
			return m, nil
		case "pgup":
			if m.fullPlan {
				m.planScroll = max(0, m.planScroll-5)
			} else {
				m.scroll += 5
			}
			return m, nil
		case "pgdown":
			if m.fullPlan {
				m.planScroll += 5
			} else {
				m.scroll = max(0, m.scroll-5)
			}
			return m, nil
		case "ctrl+s":
			if m.waiting && !m.secret && strings.Contains(m.prompt, "You") {
				return m, m.send("/install", false)
			}
			return m, nil
		case "enter":
			if m.finished {
				return m, tea.Quit
			}
			if !m.waiting {
				return m, nil
			}
			if m.secret {
				return m, m.send(string(m.private), true)
			}
			return m, m.send(string(m.draft), false)
		}
		if !m.waiting {
			return m, nil
		}
		input := &m.draft
		if m.secret {
			input = &m.private
		}
		m.cursor = min(max(0, m.cursor), len(*input))
		switch msg.Type {
		case tea.KeyLeft:
			m.cursor = max(0, m.cursor-1)
		case tea.KeyRight:
			m.cursor = min(len(*input), m.cursor+1)
		case tea.KeyHome, tea.KeyCtrlA:
			m.cursor = 0
		case tea.KeyEnd, tea.KeyCtrlE:
			m.cursor = len(*input)
		case tea.KeyBackspace, tea.KeyCtrlH:
			if m.cursor > 0 {
				*input = append((*input)[:m.cursor-1], (*input)[m.cursor:]...)
				m.cursor--
			}
		case tea.KeyDelete:
			if m.cursor < len(*input) {
				*input = append((*input)[:m.cursor], (*input)[m.cursor+1:]...)
			}
		case tea.KeyCtrlU:
			*input = nil
			m.cursor = 0
		case tea.KeyRunes:
			if len(*input)+len(msg.Runes) <= 32*1024 {
				clean := []rune(strings.Map(func(r rune) rune {
					if unicode.IsControl(r) {
						return -1
					}
					return r
				}, string(msg.Runes)))
				tail := append([]rune(nil), (*input)[m.cursor:]...)
				*input = append(append((*input)[:m.cursor], clean...), tail...)
				m.cursor += len(clean)
			}
		}
	}
	return m, nil
}
func (m *InstallConsole) send(text string, secret bool) tea.Cmd {
	m.waiting = false
	if secret {
		m.private = nil
	} else {
		m.add("YOU", text)
		m.draft = nil
	}
	m.cursor = 0
	m.stage = "Working"
	submit := m.submit
	return func() tea.Msg { submit(text, secret); return nil }
}
func consoleWrap(text string, width int) []string {
	return strings.Split(ansi.Wrap(consolePlain(text), max(1, width), " "), "\n")
}
func (m InstallConsole) planLines(width int, compact bool) []string {
	if m.plan == nil {
		return []string{muted.Render("No plan yet"), "", "The agent will prepare a recipe", "before asking for approval."}
	}
	p := m.plan
	lines := []string{title.Render(p.Name), muted.Render(fmt.Sprintf("Revision %d · %s", m.revision, p.Transport)), ""}
	if !compact {
		lines = append(lines, consoleWrap(p.Summary, width)...)
		lines = append(lines, "")
	}
	lines = append(lines, title.Render("INSTALLATION STEPS"))
	for index, step := range p.Steps {
		state := "pending"
		if index < m.completed || (m.finished && m.err == nil) {
			state = "done"
		} else if index == m.step {
			if m.finished && m.err != nil {
				state = "stopped"
			} else {
				state = "running"
			}
		}
		lines = append(lines, consoleWrap(fmt.Sprintf("%d  [%s] %s", index+1, state, step.Description), width)...)
		if !compact {
			lines = append(lines, consoleWrap("Command: "+step.Command, width)...)
			lines = append(lines, consoleWrap("Directory: "+step.Directory, width)...)
		}
		lines = append(lines, "")
	}
	for _, section := range []struct {
		label string
		items []string
	}{{"SOFTWARE", p.Requirements}, {"PRIVATE INPUT NAMES", p.Inputs}, {"BEFORE INSTALL", p.Manual}, {"AFTER INSTALL", p.FollowUp}, {"TARGET AGENTS", p.Targets}, {"SOURCES", p.Sources}} {
		if len(section.items) == 0 {
			continue
		}
		lines = append(lines, title.Render(section.label))
		for _, text := range section.items {
			lines = append(lines, consoleWrap(text, width)...)
		}
		lines = append(lines, "")
	}
	if !compact {
		lines = append(lines, muted.Render("Plan: "+p.Digest))
	}
	return lines
}
func (m InstallConsole) transcriptLines(width int) []string {
	var lines []string
	for _, entry := range m.entries {
		lines = append(lines, title.Render(entry.role))
		lines = append(lines, consoleWrap(entry.text, width)...)
		lines = append(lines, "")
	}
	if len(lines) == 0 {
		lines = []string{muted.Render("Connecting your planning agent…")}
	}
	return lines
}
func consoleSlice(lines []string, height, offset int, fromBottom bool) []string {
	start := min(offset, max(0, len(lines)-height))
	if fromBottom {
		start = max(0, len(lines)-height-start)
	}
	out := make([]string, height)
	for i := range out {
		if start+i < len(lines) {
			out[i] = lines[start+i]
		}
	}
	return out
}
func (m InstallConsole) controls() string {
	bar := " [ Conversation ] [ Plan / F2 ] [ Help / F1 ]"
	if m.waiting && !m.secret && !m.finished && !m.stopping {
		if strings.Contains(m.prompt, "You") {
			bar += " [ Review install ]"
		}
		if strings.Contains(m.prompt, "Install this plan?") || strings.Contains(m.prompt, "Apply this repair?") {
			bar += " [ Approve ] [ Cancel ]"
		}
	}
	return bar
}

func (m InstallConsole) View() string {
	width, height := m.width, m.height
	if width < 40 || height < 14 {
		text := "Resize terminal to at least 40 × 14. Ctrl+C stops."
		return ansi.Truncate(text, width, "")
	}
	contentHeight := max(1, height-10)
	lines := make([]string, height)
	elapsed := m.now.Sub(m.started).Round(time.Second)
	lines[0] = title.Render(" MCPDECK / INSTALLATION CHAT")
	lines[1] = fmt.Sprintf(" %s  ·  %s  ·  %s", consolePlain(m.provider), consolePlain(m.stage), elapsed)
	lines[2] = m.controls()
	lines[3] = muted.Render(" " + strings.Repeat("─", max(1, width-2)))
	if m.help {
		help := []string{"INSTALLATION CHAT", "", "Ask the agent to revise the recipe before installing.", "F2 opens exact commands, sources and target agents.", "Ctrl+S continues from chat to the approval prompt.", "Enter sends the current input; an empty approval cancels.", "Passwords use a separate masked field and never enter chat.", "PgUp/PgDown or mouse wheel scroll; Ctrl+C requests cancellation.", "Esc closes help/plan; after completion Enter returns to MCPDeck.", "", "Provider account usage may apply. Keep secrets out of chat."}
		for i, text := range consoleSlice(help, contentHeight, 0, false) {
			lines[4+i] = " " + text
		}
	} else if m.fullPlan {
		for i, text := range consoleSlice(m.planLines(width-4, false), contentHeight, m.planScroll, false) {
			lines[4+i] = " " + text
		}
	} else if width >= 100 {
		right := max(30, width/3)
		left := width - right - 5
		chat := consoleSlice(m.transcriptLines(left-2), contentHeight, m.scroll, true)
		plan := consoleSlice(m.planLines(right-2, true), contentHeight, 0, false)
		for i := range chat {
			lines[4+i] = " " + lipgloss.NewStyle().Width(left).Render(ansi.Truncate(chat[i], left, "")) + muted.Render(" │ ") + plan[i]
		}
	} else {
		for i, text := range consoleSlice(m.transcriptLines(width-4), contentHeight, m.scroll, true) {
			lines[4+i] = " " + text
		}
	}
	prompt := "Working… Input opens when the agent is ready."
	if m.waiting {
		prompt = strings.TrimSpace(m.prompt)
		if strings.Contains(prompt, "You") {
			prompt = "Message your agent · Enter asks · Ctrl+S reviews install"
		}
		if prompt == "" {
			prompt = "Your reply"
		}
	}
	if m.finished {
		prompt = "Finished — Enter or Esc to return"
	}
	lines[height-5] = muted.Render(" " + strings.Repeat("─", max(1, width-2)))
	lines[height-4] = " " + prompt
	valueRunes := m.draft
	if m.secret {
		valueRunes = []rune(strings.Repeat("•", len(m.private)))
	}
	cursor := min(max(0, m.cursor), len(valueRunes))
	inputWidth := max(1, width-7)
	before := consolePlain(string(valueRunes[:cursor]))
	before = ansi.TruncateLeft(before, max(0, ansi.StringWidth(before)-inputWidth+1), "")
	after := ansi.Truncate(consolePlain(string(valueRunes[cursor:])), max(0, inputWidth-ansi.StringWidth(before)-1), "")
	lines[height-3] = " › " + before + selected.Render("▏") + after
	if m.secret {
		lines[height-2] = muted.Render(" Private value · masked · not added to conversation")
	} else {
		lines[height-2] = muted.Render(" Enter send  ·  Ctrl+S review install  ·  F2 plan  ·  PgUp/PgDn scroll")
	}
	lines[height-1] = muted.Render(" Keep secrets out of chat. Commands run only after approval.  Ctrl+C stop")
	for index, line := range lines {
		lines[index] = ansi.Truncate(line, width, "")
	}
	return strings.Join(lines, "\n")
}
