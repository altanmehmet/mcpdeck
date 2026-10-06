package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func consoleUpdate(m InstallConsole, msg tea.Msg) InstallConsole {
	updated, _ := m.Update(msg)
	return updated.(InstallConsole)
}
func TestInstallConsoleApprovalAndPrivateInput(t *testing.T) {
	var submitted string
	var secret bool
	m := NewInstallConsole(nil, make(chan struct{}), func(value string, hidden bool) { submitted = value; secret = hidden }, func() {}, "Install a test MCP")
	m = consoleUpdate(m, ConsoleEvent{Kind: "await", Text: "You ›"})
	m = consoleUpdate(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Question draft")})
	m.waiting = false
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("sent input while workflow was busy")
	}
	m = consoleUpdate(m, ConsoleEvent{Kind: "await", Text: "TOKEN (hidden):", Secret: true})
	m = consoleUpdate(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("do-not-display-token")})
	if strings.Contains(m.View(), "do-not-display-token") || m.cursor != len(m.private) {
		t.Fatal("secret exposed or private cursor incorrect")
	}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(InstallConsole)
	cmd()
	if !secret || submitted != "do-not-display-token" {
		t.Fatal("private value not delivered separately")
	}
	for _, entry := range m.entries {
		if strings.Contains(entry.text, submitted) {
			t.Fatal("secret entered conversation")
		}
	}
	m = consoleUpdate(m, ConsoleEvent{Kind: "status", Text: "Preparing plan"})
	if m.secret || len(m.private) != 0 {
		t.Fatal("private editor remained active during agent work")
	}
	m = consoleUpdate(m, ConsoleEvent{Kind: "plan", Plan: &ConsolePlan{Name: "demo", Steps: []ConsoleStep{{Command: "echo []", Description: "Example step"}}, Targets: []string{"codex"}}})
	m = consoleUpdate(m, ConsoleEvent{Kind: "await", Text: "Install this plan? [y/N]:"})
	if !m.fullPlan || !strings.Contains(m.View(), "Command: echo []") {
		t.Fatal("exact commands not shown for approval")
	}
	m.draft = nil
	m.cursor = 0
	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(InstallConsole)
	cmd()
	if secret || submitted != "" {
		t.Fatal("empty approval must remain a cancel answer")
	}
}
func TestInstallConsoleRevisionFailureAndCancellation(t *testing.T) {
	cancelled := false
	m := NewInstallConsole(nil, make(chan struct{}), func(string, bool) {}, func() { cancelled = true }, "")
	m = consoleUpdate(m, ConsoleEvent{Kind: "plan", Plan: &ConsolePlan{Name: "demo", Digest: "old", Summary: "Previous valid recipe"}})
	m = consoleUpdate(m, ConsoleEvent{Kind: "output", Text: "Agent reply failed; previous recipe retained"})
	if m.plan.Digest != "old" || m.revision != 1 {
		t.Fatal("failed reply replaced recipe")
	}
	m = consoleUpdate(m, ConsoleEvent{Kind: "plan", Plan: &ConsolePlan{Name: "demo", Digest: "new"}})
	if m.revision != 2 {
		t.Fatal("updated recipe revision missing")
	}
	m = consoleUpdate(m, tea.KeyMsg{Type: tea.KeyCtrlC})
	m = consoleUpdate(m, ConsoleEvent{Kind: "await", Text: "You ›"})
	if !cancelled || m.waiting {
		t.Fatal("cancel did not stop input")
	}
	m = consoleUpdate(m, ConsoleEvent{Kind: "done", Err: context.Canceled})
	if !m.finished || m.err == nil {
		t.Fatal("cancel result lost")
	}
	m = consoleUpdate(m, ConsoleEvent{Kind: "done", Err: errors.New("step failed")})
	if strings.Contains(m.View(), "Setup finished") {
		t.Fatal("failure reported as success")
	}
}
func TestInstallConsoleViewportAndEditor(t *testing.T) {
	m := NewInstallConsole(nil, make(chan struct{}), func(string, bool) {}, func() {}, "test")
	for _, size := range [][2]int{{120, 30}, {60, 20}, {40, 14}, {20, 8}} {
		m = consoleUpdate(m, tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		for _, line := range strings.Split(m.View(), "\n") {
			if ansi.StringWidth(line) > size[0] {
				t.Fatal("console exceeded viewport width")
			}
		}
		if len(strings.Split(m.View(), "\n")) > size[1] {
			t.Fatal("console exceeded viewport height")
		}
	}
	m = consoleUpdate(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = consoleUpdate(m, ConsoleEvent{Kind: "await", Text: "You ›"})
	m = consoleUpdate(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Türkçe 🚀")})
	m = consoleUpdate(m, tea.KeyMsg{Type: tea.KeyHome})
	m = consoleUpdate(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("A")})
	if string(m.draft) != "ATürkçe 🚀" || !strings.Contains(ansi.Strip(m.View()), "A▏Türkçe") {
		t.Fatal("Unicode input cursor did not follow edits")
	}
	m.add("SYSTEM", "\x1b[2Jexternal\x00content")
	if strings.Contains(m.entries[len(m.entries)-1].text, "\x1b") || strings.Contains(m.entries[len(m.entries)-1].text, "\x00") {
		t.Fatal("unsafe output controls retained")
	}
}

func TestInstallConsoleMouseApprovalIsExplicitAndPhaseBound(t *testing.T) {
	sent := ""
	m := NewInstallConsole(nil, make(chan struct{}), func(value string, hidden bool) { sent = value }, func() {}, "")
	if strings.Contains(m.controls(), "Approve") {
		t.Fatal("approval displayed before workflow asks")
	}
	m = consoleUpdate(m, ConsoleEvent{Kind: "await", Text: "Install this plan? [y/N]:"})
	left := strings.Index(m.controls(), "[ Approve ]")
	updated, cmd := m.Update(tea.MouseMsg{X: left + 2, Y: 2, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m = updated.(InstallConsole)
	if cmd == nil {
		t.Fatal("visible approval button did not submit")
	}
	cmd()
	if sent != "y" || m.waiting {
		t.Fatal("wrong approval answer")
	}
	if strings.Contains(m.controls(), "Approve") {
		t.Fatal("approval remained actionable after submission")
	}
	m = consoleUpdate(m, ConsoleEvent{Kind: "await", Text: "TOKEN (hidden):", Secret: true})
	if strings.Contains(m.controls(), "Approve") {
		t.Fatal("private field exposed an approval action")
	}
}

func TestInstallConsoleSuccessfulStepsSurviveProbeFailure(t *testing.T) {
	m := NewInstallConsole(nil, make(chan struct{}), func(string, bool) {}, func() {}, "")
	m = consoleUpdate(m, ConsoleEvent{Kind: "plan", Plan: &ConsolePlan{Name: "demo", Steps: []ConsoleStep{{Description: "Downloaded runtime"}}}})
	m = consoleUpdate(m, ConsoleEvent{Kind: "step", Text: "Installation step 1/1: node"})
	m = consoleUpdate(m, ConsoleEvent{Kind: "status", Text: "Verifying connection"})
	m = consoleUpdate(m, ConsoleEvent{Kind: "done", Err: errors.New("MCP probe failed")})
	if !strings.Contains(strings.Join(m.planLines(80, false), "\n"), "[done]") {
		t.Fatal("successful setup step marked failed by later probe error")
	}
	if m.fullPlan || !strings.Contains(m.View(), "MCP probe failed") {
		t.Fatal("result was hidden behind plan view")
	}
}
