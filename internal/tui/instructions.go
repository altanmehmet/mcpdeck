package tui

import (
	"fmt"
	"strings"

	"github.com/altanmehmet/mcpdeck/internal/instructions"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

type instructionEditor struct {
	text               []rune
	original, message  string
	cursor, offset     int
	selectAll, canUndo bool
	undoText           []rune
	undoCursor         int
	review, standalone bool
	results            []instructions.Result
	documents          []instructions.Document
	document           *instructions.Document
	documentOriginal   string
	sharedDraft        []rune
	picking, adding    bool
	selected           int
}

type instructionsSavedMsg struct {
	results []instructions.Result
	err     error
	text    string
	native  bool
}

// NewInstructions opens the same editor used by the terminal panel.
func NewInstructions(d *model.Deck, s store.Store) Model {
	m := New(d, s)
	m.openInstructions()
	if m.instructionEditor != nil {
		m.instructionEditor.standalone = true
	}
	return m
}

func (m *Model) openInstructions() {
	manager := instructions.New(m.store)
	text, err := manager.Load()
	if err != nil {
		m.status = err.Error()
		return
	}
	results, err := manager.Status(m.deck, false)
	if err != nil {
		m.status = err.Error()
		return
	}
	m.instructionEditor = &instructionEditor{text: []rune(text), original: text, cursor: len([]rune(text)), results: results}
}

func (m Model) updateInstructions(msg tea.Msg) (tea.Model, tea.Cmd) {
	f := m.instructionEditor
	if event, ok := msg.(clipboardResultMsg); ok {
		f.message = "Instructions sent to terminal clipboard. Paste into the agent's personal settings for manual targets."
		if event.err != nil {
			f.message = "Clipboard copy failed; check terminal OSC 52 support."
		}
		return m, nil
	}
	if event, ok := msg.(instructionsSavedMsg); ok {
		m.busy = false
		f.results = event.results
		f.message = "Saved. Start new agent sessions or reload instructions."
		if event.native {
			if event.err == nil {
				f.documentOriginal = event.text
			}
		} else if event.err == nil || event.text == string(f.text) {
			f.original = event.text
		}
		if event.err != nil {
			f.message = "Saved/sync incomplete: " + event.err.Error()
		}
		return m, nil
	}
	if m.busy {
		return m, nil
	}
	if f.picking {
		return m.updateInstructionDocuments(msg)
	}
	key, ok := msg.(tea.KeyMsg)
	if mouse, isMouse := msg.(tea.MouseMsg); isMouse {
		if mouse.Action == tea.MouseActionPress && mouse.Button == tea.MouseButtonLeft && mouse.Y == 4 {
			x := mouse.X
			if f.document != nil {
				x -= 36
			}
			if x >= 0 && x < 44 {
				switch {
				case x < 15:
					key = tea.KeyMsg{Type: tea.KeyCtrlA}
				case x < 30:
					key = tea.KeyMsg{Type: tea.KeyCtrlU}
				default:
					key = tea.KeyMsg{Type: tea.KeyCtrlZ}
				}
				ok = true
			}
		}
		if f.document != nil && mouse.X < 36 && mouse.Action == tea.MouseActionPress && mouse.Button == tea.MouseButtonLeft && mouse.Y == 4 {
			if mouse.X < 18 {
				key = tea.KeyMsg{Type: tea.KeyCtrlF}
			} else {
				key = tea.KeyMsg{Type: tea.KeyCtrlG}
			}
			ok = true
		}
		if mouse.Action == tea.MouseActionPress && mouse.Button == tea.MouseButtonLeft && mouse.Y == 2 {
			if mouse.X < 12 {
				key = tea.KeyMsg{Type: tea.KeyEsc}
				ok = true
			} else if mouse.X < 37 {
				key = tea.KeyMsg{Type: tea.KeyCtrlS}
				ok = true
			} else if mouse.X < 48 {
				key = tea.KeyMsg{Type: tea.KeyCtrlY}
				ok = true
			} else if mouse.X < 63 {
				key = tea.KeyMsg{Type: tea.KeyCtrlE}
				ok = true
			} else if mouse.X < 76 {
				key = tea.KeyMsg{Type: tea.KeyCtrlN}
				ok = true
			}
		}
		if mouse.Button == tea.MouseButtonWheelDown {
			f.offset++
		}
		if mouse.Button == tea.MouseButtonWheelUp {
			f.offset = max(0, f.offset-1)
		}
	}
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "ctrl+a", "ctrl+u", "ctrl+z":
		if f.review {
			f.message = "Press Esc to return to editing before changing text."
			return m, nil
		}
		switch key.String() {
		case "ctrl+a":
			f.selectAll = true
			f.message = "All text selected. Type or paste to replace; Backspace/Delete to clear. Nothing is saved yet."
			if f.document != nil {
				f.message += " The shared MCPDeck block is kept."
			}
		case "ctrl+u":
			m.replaceInstructionText("")
		case "ctrl+z":
			if f.canUndo {
				f.text = append([]rune(nil), f.undoText...)
				f.cursor = f.undoCursor
				f.selectAll, f.canUndo = false, false
				f.message = "Last text edit undone. Ctrl+S to review changes."
			}
		}
		return m, nil
	case "ctrl+e":
		return m.pickInstructionDocuments()
	case "ctrl+g":
		m.restoreSharedInstructions()
		return m, nil
	case "ctrl+f":
		if f.document != nil {
			text, err := instructions.SharedText(string(f.text))
			if err != nil {
				f.message = err.Error()
				return m, nil
			}
			f.resetEditing()
			f.text = []rune(text)
			f.sharedDraft = append([]rune(nil), f.text...)
			f.document = nil
			f.review = false
			f.cursor = len(f.text)
			f.offset = 0
			f.message = "Loaded as shared instructions. Ctrl+S to review and distribute to all supported agents."
		}
		return m, nil
	case "ctrl+n":
		if f.document == nil && !f.adding {
			f.sharedDraft = append([]rune(nil), f.text...)
		}
		f.resetEditing()
		f.document = nil
		f.adding = true
		f.text = nil
		f.cursor = 0
		f.offset = 0
		f.review = false
		f.message = "Write one instruction. Ctrl+S adds it to the shared text and previews all targets."
		return m, nil
	case "ctrl+y":
		return m, copyToTerminalClipboard(string(f.text))
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		if f.selectAll {
			f.selectAll = false
			return m, nil
		}
		if f.review {
			f.review = false
			f.offset = 0
			return m, nil
		}
		if f.document != nil || f.adding {
			m.restoreSharedInstructions()
			return m, nil
		}
		if f.standalone {
			return m, tea.Quit
		}
		m.instructionEditor = nil
		return m, nil
	case "ctrl+s":
		f.selectAll = false
		if f.adding {
			text, err := instructions.AppendText(string(f.sharedDraft), string(f.text))
			if err != nil {
				f.message = err.Error()
				return m, nil
			}
			f.resetEditing()
			f.text = []rune(text)
			f.sharedDraft = append([]rune(nil), f.text...)
			f.adding = false
			f.cursor = len(f.text)
		}
		if f.document != nil {
			return m.saveAgentInstructions()
		}
		if !f.review {
			if err := instructions.Validate(string(f.text)); err != nil {
				f.message = err.Error()
				return m, nil
			}
			results, err := instructions.New(m.store).Preview(m.deck, string(f.text), strings.TrimSpace(string(f.text)) == "")
			if err != nil {
				f.message = err.Error()
				return m, nil
			}
			f.results = results
			f.review = true
			f.offset = 0
			f.message = "Review: REPLACE shared guidance only. Ctrl+S saves; Esc returns to editing."
			if strings.TrimSpace(string(f.text)) == "" {
				f.message = "Review: REMOVE shared MCPDeck blocks from all configured targets; other text stays. Ctrl+S confirms."
			}
			return m, nil
		}
		manager := instructions.New(m.store)
		text, expected := string(f.text), f.original
		m.busy = true
		f.message = "Saving and distributing personal instructions..."
		return m, func() tea.Msg {
			d, err := m.store.Load()
			if err != nil {
				return instructionsSavedMsg{err: err, text: expected}
			}
			results, err := manager.Apply(d, text, strings.TrimSpace(text) == "", &expected)
			// Source can remain unchanged if validation/locking failed.
			saved, loadErr := manager.Load()
			if loadErr != nil {
				saved = expected
			}
			return instructionsSavedMsg{results: results, err: err, text: saved}
		}
	}
	if f.review {
		switch key.String() {
		case "down", "pgdown":
			f.offset++
		case "up", "pgup":
			f.offset = max(0, f.offset-1)
		}
		return m, nil
	}
	switch key.String() {
	case "left":
		f.selectAll = false
		f.cursor = max(0, f.cursor-1)
	case "right":
		f.selectAll = false
		f.cursor = min(len(f.text), f.cursor+1)
	case "home":
		f.selectAll = false
		for f.cursor > 0 && f.text[f.cursor-1] != '\n' {
			f.cursor--
		}
	case "end":
		f.selectAll = false
		for f.cursor < len(f.text) && f.text[f.cursor] != '\n' {
			f.cursor++
		}
	case "up", "down":
		f.selectAll = false
		start := f.cursor
		for start > 0 && f.text[start-1] != '\n' {
			start--
		}
		column := f.cursor - start
		if key.String() == "up" && start > 0 {
			finish := start - 1
			start = finish
			for start > 0 && f.text[start-1] != '\n' {
				start--
			}
			f.cursor = min(finish, start+column)
		}
		if key.String() == "down" {
			finish := f.cursor
			for finish < len(f.text) && f.text[finish] != '\n' {
				finish++
			}
			if finish < len(f.text) {
				start = finish + 1
				finish = start
				for finish < len(f.text) && f.text[finish] != '\n' {
					finish++
				}
				f.cursor = min(finish, start+column)
			}
		}
	case "backspace":
		if f.selectAll {
			m.replaceInstructionText("")
			return m, nil
		}
		if f.cursor > 0 {
			f.rememberEdit()
			f.text = append(f.text[:f.cursor-1], f.text[f.cursor:]...)
			f.cursor--
		}
	case "delete":
		if f.selectAll {
			m.replaceInstructionText("")
			return m, nil
		}
		if f.cursor < len(f.text) {
			f.rememberEdit()
			f.text = append(f.text[:f.cursor], f.text[f.cursor+1:]...)
		}
	default:
		value := ""
		switch {
		case key.Type == tea.KeyRunes:
			value = string(key.Runes)
		case key.Type == tea.KeySpace:
			value = " "
		case key.Type == tea.KeyEnter:
			value = "\n"
		case key.Type == tea.KeyTab:
			value = "    "
		}
		value = strings.ReplaceAll(strings.ReplaceAll(value, "\r\n", "\n"), "\r", "\n")
		if value == "" {
			return m, nil
		}
		if f.selectAll {
			m.replaceInstructionText(value)
			return m, nil
		}
		limit := instructions.MaxBytes
		if f.document != nil {
			limit = instructions.MaxDocumentBytes
		}
		if len(string(f.text))+len(value) <= limit {
			f.rememberEdit()
			runes := []rune(value)
			tail := append([]rune(nil), f.text[f.cursor:]...)
			f.text = append(append(f.text[:f.cursor], runes...), tail...)
			f.cursor += len(runes)
		} else {
			f.message = "Personal instructions are limited to 24 KiB."
		}
	}
	return m, nil
}

func (f *instructionEditor) rememberEdit() {
	f.undoText = append([]rune(nil), f.text...)
	f.undoCursor, f.canUndo = f.cursor, true
}

func (f *instructionEditor) resetEditing() {
	f.selectAll, f.canUndo = false, false
	f.undoText = nil
}

func (m *Model) replaceInstructionText(value string) {
	f := m.instructionEditor
	text := value
	if f.document != nil {
		var err error
		text, err = instructions.ReplacePersonalText(string(f.text), value)
		if err != nil {
			f.message = err.Error()
			return
		}
	}
	limit := instructions.MaxBytes
	if f.document != nil {
		limit = instructions.MaxDocumentBytes
	}
	if len(text) > limit {
		f.message = fmt.Sprintf("Text exceeds the %d byte limit.", limit)
		return
	}
	f.rememberEdit()
	f.text, f.cursor, f.offset = []rune(text), len([]rune(value)), 0
	if f.document != nil {
		f.cursor = len(f.text)
	}
	f.selectAll = false
	f.message = "Draft changed. Ctrl+Z undoes this edit; Ctrl+S reviews before saving."
	if f.document != nil {
		f.message += " Shared guidance was kept."
	}
}

func (m Model) instructionsView() string {
	f := m.instructionEditor
	if f.picking {
		return m.instructionDocumentsView()
	}
	lines := make([]string, m.height)
	lines[0] = title.Render(" MCPDECK / PERSONAL INSTRUCTIONS")
	lines[1] = muted.Render(" One text for your agents. Applies across projects; local project rules keep their scope.")
	back := "Back"
	if f.review {
		back = "Edit"
	}
	button := "Review targets"
	if f.review {
		button = "Save & distribute"
	}
	lines[2] = " [ " + fit(back, 4) + " ]  [ " + fit(button, 17) + " ]  [ Copy ]  [ Existing ]  [ Add one ]"
	lines[3] = muted.Render(" Save replaces shared guidance in every target; other personal text stays. Add one appends to shared guidance.")
	lines[4] = " [ Select all ] [ Clear text ] [ Undo edit ]"
	if f.adding {
		lines[0] = title.Render(" MCPDECK / ADD ONE SHARED INSTRUCTION")
		lines[3] = muted.Render(" This adds to the END of your shared guidance; existing shared and personal instructions are kept.")
	}
	if f.document != nil {
		if f.review {
			button = "Save this file"
		} else {
			button = "Review file"
		}
		lines[2] = " [ " + fit(back, 4) + " ]  [ " + fit(button, 17) + " ]  [ Copy ]  [ Existing ]  [ Add one ]"
		lines[0] = title.Render(" MCPDECK / EXISTING INSTRUCTIONS / " + plain(f.document.Agent))
		lines[1] = muted.Render(" " + plain(f.document.Path))
		lines[3] = muted.Render(" Save updates THIS FILE only. Shared block stays; Use for all replaces shared guidance after review.")
		lines[4] = " [ Use for all ]  [ Shared editor ]  [ Select all ] [ Clear text ] [ Undo edit ]"
	}
	rows := max(1, m.height-9)
	if f.review {
		var entries []string
		for _, r := range f.results {
			entries = append(entries, r.Agent+"  /  "+r.Status)
			if r.Path != "" {
				entries = append(entries, "  "+r.Path)
			}
			if r.Detail != "" {
				entries = append(entries, "  "+r.Detail)
			}
		}
		f.offset = min(f.offset, max(0, len(entries)-rows))
		for i := 0; i < rows && f.offset+i < len(entries); i++ {
			lines[5+i] = " " + plain(entries[f.offset+i])
		}
	} else {
		content := strings.Split(string(f.text), "\n")
		before := string(f.text[:f.cursor])
		row := strings.Count(before, "\n")
		column := len([]rune(before[strings.LastIndex(before, "\n")+1:]))
		if row < f.offset {
			f.offset = row
		}
		if row >= f.offset+rows {
			f.offset = row - rows + 1
		}
		for i := 0; i < rows && f.offset+i < len(content); i++ {
			index := f.offset + i
			runes := []rune(content[index])
			line := plain(content[index])
			if f.selectAll {
				line = selected.Render(line + " ")
			} else if index == row {
				start := max(0, column-max(1, m.width-12))
				line = plain(string(runes[start:column])) + selected.Render("▏") + plain(string(runes[column:]))
			}
			lines[5+i] = fmt.Sprintf(" %3d │ %s", index+1, line)
		}
	}
	lines[m.height-3] = " " + plain(f.message)
	limit := instructions.MaxBytes
	if f.document != nil {
		limit = instructions.MaxDocumentBytes
	}
	lines[m.height-2] = muted.Render(fmt.Sprintf(" %d / %d bytes · Ctrl+A: select all · Ctrl+U: clear · Ctrl+Z: undo", len(string(f.text)), limit))
	lines[m.height-1] = muted.Render(" Ctrl+S: review/save  Ctrl+E: existing  Ctrl+N: add one  Ctrl+Y: copy  Esc: back")
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], m.width, "")
	}
	return strings.Join(lines, "\n")
}
