package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/altanmehmet/mcpdeck/internal/instructions"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
)

func NewAgentInstructions(d *model.Deck, s store.Store, agent, path string) (Model, error) {
	m := NewInstructions(d, s)
	manager := instructions.New(s)
	doc, err := manager.SelectDocument(d, agent, path)
	if err != nil {
		return m, err
	}
	text, err := manager.ReadDocument(d, doc)
	if err != nil {
		return m, err
	}
	if m.instructionEditor == nil {
		return m, fmt.Errorf("cannot open shared instruction editor: %s", m.status)
	}
	f := m.instructionEditor
	f.sharedDraft = append([]rune(nil), f.text...)
	f.document = &doc
	f.documentOriginal = text
	f.text = []rune(text)
	f.cursor = 0
	return m, nil
}

func (m *Model) restoreSharedInstructions() {
	f := m.instructionEditor
	if f.document != nil || f.adding || f.picking {
		f.text = append([]rune(nil), f.sharedDraft...)
	}
	f.document = nil
	f.adding = false
	f.picking = false
	f.review = false
	f.cursor = len(f.text)
	f.offset = 0
}

func (m Model) pickInstructionDocuments() (tea.Model, tea.Cmd) {
	f := m.instructionEditor
	documents, err := instructions.New(m.store).Documents(m.deck)
	if err != nil {
		f.message = err.Error()
		return m, nil
	}
	if f.document == nil && !f.adding {
		f.sharedDraft = append([]rune(nil), f.text...)
	}
	f.documents = documents
	f.picking = true
	f.review = false
	f.selected = 0
	f.offset = 0
	f.message = "Select an existing global instruction file. Enter or click to view and edit."
	return m, nil
}

func (m Model) updateInstructionDocuments(msg tea.Msg) (tea.Model, tea.Cmd) {
	f := m.instructionEditor
	open := false
	if mouse, ok := msg.(tea.MouseMsg); ok {
		if mouse.Button == tea.MouseButtonWheelDown {
			f.selected = min(len(f.documents)-1, f.selected+1)
		}
		if mouse.Button == tea.MouseButtonWheelUp {
			f.selected = max(0, f.selected-1)
		}
		if mouse.Button == tea.MouseButtonLeft && mouse.Action == tea.MouseActionPress {
			if mouse.Y == 2 {
				m.restoreSharedInstructions()
				return m, nil
			}
			index := f.offset + mouse.Y - 5
			if mouse.Y >= 5 && mouse.Y < m.height-4 && index >= 0 && index < len(f.documents) {
				f.selected = index
				open = true
			}
		}
	}
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc", "ctrl+g":
			m.restoreSharedInstructions()
			return m, nil
		case "down":
			f.selected = min(len(f.documents)-1, f.selected+1)
		case "up":
			f.selected = max(0, f.selected-1)
		case "enter":
			open = true
		}
	}
	if open && len(f.documents) > 0 {
		doc := f.documents[f.selected]
		text, err := instructions.New(m.store).ReadDocument(m.deck, doc)
		if err != nil {
			f.message = err.Error()
			return m, nil
		}
		f.document = &doc
		f.documentOriginal = text
		f.text = []rune(text)
		f.cursor = 0
		f.offset = 0
		f.picking = false
		f.adding = false
		f.message = "Edit personal text here. Ctrl+S saves this file; Ctrl+F copies its guidance to the shared editor for all agents."
	}
	return m, nil
}

func (m Model) saveAgentInstructions() (tea.Model, tea.Cmd) {
	f := m.instructionEditor
	manager := instructions.New(m.store)
	doc, text, expected := *f.document, string(f.text), f.documentOriginal
	if !f.review {
		if err := manager.CheckDocument(m.deck, doc, text, expected); err != nil {
			f.message = err.Error()
			return m, nil
		}
		f.results = []instructions.Result{{Agent: doc.Agent, Path: doc.Path, Status: "pending", Detail: "Update this global file only. Other agents sharing this exact file will also see the change."}}
		f.review = true
		f.offset = 0
		f.message = "Review this existing file update. Ctrl+S again saves it with a backup."
		return m, nil
	}
	m.busy = true
	return m, func() tea.Msg {
		d, err := m.store.Load()
		if err == nil {
			err = manager.SaveDocument(d, doc, text, expected)
		}
		status := "synced"
		if err != nil {
			status = "failed"
		}
		return instructionsSavedMsg{results: []instructions.Result{{Agent: doc.Agent, Path: doc.Path, Status: status}}, err: err, text: text, native: true}
	}
}

func (m Model) instructionDocumentsView() string {
	f := m.instructionEditor
	lines := make([]string, m.height)
	lines[0] = title.Render(" MCPDECK / EXISTING GLOBAL INSTRUCTIONS")
	lines[1] = muted.Render(" Includes personal files added outside MCPDeck. Project instruction files are excluded.")
	lines[2] = " [ Shared editor ]"
	lines[3] = muted.Render(" Select a file to see its full current text. Ctrl+F in the editor: use for all agents.")
	rows := max(1, m.height-9)
	if f.selected < f.offset {
		f.offset = max(0, f.selected)
	}
	if f.selected >= f.offset+rows {
		f.offset = f.selected - rows + 1
	}
	for i := 0; i < rows && f.offset+i < len(f.documents); i++ {
		index := f.offset + i
		doc := f.documents[index]
		line := " " + plain(doc.Agent) + "  " + plain(doc.Path)
		if doc.Note != "" {
			line += " · " + plain(doc.Note)
		}
		if index == f.selected {
			line = selected.Render(line)
		}
		lines[5+i] = line
	}
	if len(f.documents) == 0 {
		lines[5] = " No verified global instruction locations are configured."
	}
	lines[m.height-3] = " " + plain(f.message)
	lines[m.height-2] = muted.Render(" Managed shared guidance and existing agent-specific guidance are both visible.")
	lines[m.height-1] = muted.Render(" ↑/↓: select   Enter/click: view and edit   Esc: shared editor")
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], m.width, "")
	}
	return strings.Join(lines, "\n")
}
