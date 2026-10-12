package tui

import (
	"fmt"
	"github.com/altanmehmet/mcpdeck/internal/testutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/altanmehmet/mcpdeck/internal/instructions"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
	tea "github.com/charmbracelet/bubbletea"
)

func TestInstructionsEditorRequiresReviewAndPreservesUnicode(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	t.Setenv("COPILOT_HOME", filepath.Join(home, ".copilot"))
	s := store.Store{Path: filepath.Join(home, "deck", "deck.json")}
	config := filepath.Join(home, ".cursor", "mcp.json")
	if err := store.AtomicWrite(config, []byte("{}")); err != nil {
		t.Fatal(err)
	}
	d := &model.Deck{Version: 1, Servers: map[string]model.ServerConfig{}, Profiles: map[string]model.ProfileConfig{"cursor": {TargetPath: config}}}
	if err := s.Save(d); err != nil {
		t.Fatal(err)
	}
	m := NewInstructions(d, s)
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Türkçe yanıt ver.\nNever log secrets.")})
	m = next.(Model)
	next, command := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = next.(Model)
	if command != nil || !m.instructionEditor.review {
		t.Fatal("save did not require target review")
	}
	if _, err := os.Stat(instructions.New(s).SourcePath); !os.IsNotExist(err) {
		t.Fatal("review wrote state")
	}
	if !strings.Contains(m.View(), "cursor") {
		t.Fatal("review hides targets")
	}
	next, command = m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = next.(Model)
	if command == nil {
		t.Fatal("save action absent")
	}
	next, _ = m.Update(command())
	m = next.(Model)
	if m.busy || m.instructionEditor != nil || !strings.Contains(m.status, "saved") {
		t.Fatal("save did not finish")
	}
	text, err := instructions.New(s).Load()
	if err != nil || text != "Türkçe yanıt ver.\nNever log secrets." {
		t.Fatal(text, err)
	}
	if _, err := os.Stat(filepath.Join(home, ".cursor", "rules", "mcpdeck-global.mdc")); err != nil {
		t.Fatal(err)
	}
}

func TestInstructionsShortcutAndMouseOpenEditor(t *testing.T) {
	m := mouseFixture(t)
	next, _ := m.Update(keyMessage(m.keys.Bindings["instructions"]))
	if next.(Model).instructionEditor == nil {
		t.Fatal("instructions shortcut absent")
	}
	for _, b := range m.buttons() {
		if b.action == "Instructions" {
			next, _ := m.Update(tea.MouseMsg{X: b.x + 1, Y: b.y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
			if next.(Model).instructionEditor == nil {
				t.Fatal("instructions click absent")
			}
			return
		}
	}
	t.Fatal("instructions button absent")
}

func instructionFixture(t *testing.T) Model {
	t.Helper()
	home := t.TempDir()
	testutil.SetHome(t, home)
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	t.Setenv("COPILOT_HOME", filepath.Join(home, ".copilot"))
	s := store.Store{Path: filepath.Join(home, "deck", "deck.json")}
	d := &model.Deck{Version: 1, Servers: map[string]model.ServerConfig{}, Profiles: map[string]model.ProfileConfig{}}
	for _, agent := range []string{"codex", "cursor"} {
		path := filepath.Join(home, "configs", agent+".json")
		if err := store.AtomicWrite(path, []byte("{}")); err != nil {
			t.Fatal(err)
		}
		d.Profiles[agent] = model.ProfileConfig{TargetPath: path}
	}
	if err := s.Save(d); err != nil {
		t.Fatal(err)
	}
	if _, err := instructions.New(s).Apply(d, "Initial shared guidance.", false, nil); err != nil {
		t.Fatal(err)
	}
	return NewInstructions(d, s)
}

func TestAddOneInstructionKeepsPreviousSharedGuidance(t *testing.T) {
	m := instructionFixture(t)
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	m = next.(Model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Never log credentials.")})
	m = next.(Model)
	next, command := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = next.(Model)
	if command != nil || !m.instructionEditor.review || m.instructionEditor.adding {
		t.Fatal("new instruction was not reviewed as shared guidance")
	}
	next, command = m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = next.(Model)
	if command == nil {
		t.Fatal("missing distribute action")
	}
	next, _ = m.Update(command())
	m = next.(Model)
	text, err := instructions.New(m.store).Load()
	if err != nil || !strings.Contains(text, "Initial shared guidance.") || !strings.Contains(text, "Never log credentials.") {
		t.Fatal(text, err)
	}
}

func TestExistingNativeEditAndUseForAll(t *testing.T) {
	m := instructionFixture(t)
	manager := instructions.New(m.store)
	doc, err := manager.SelectDocument(m.deck, "codex", "")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := manager.ReadDocument(m.deck, doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AtomicWrite(doc.Path, []byte("Personal original.\n"+raw)); err != nil {
		t.Fatal(err)
	}
	m, err = NewAgentInstructions(m.deck, m.store, "codex", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(m.View(), "Personal original.") {
		t.Fatal("native content not visible")
	}
	m.instructionEditor.text = []rune(strings.ReplaceAll(string(m.instructionEditor.text), "Personal original.", "Updated personal guidance."))
	next, command := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = next.(Model)
	if command != nil || !m.instructionEditor.review {
		t.Fatal("agent file change was not reviewed")
	}
	next, command = m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = next.(Model)
	if command == nil {
		t.Fatal("missing native save action")
	}
	next, _ = m.Update(command())
	m = next.(Model)
	text, _ := manager.Load()
	if text != "Initial shared guidance." {
		t.Fatal("native save overwrote shared instructions")
	}
	updated, _ := manager.ReadDocument(m.deck, doc)
	if !strings.Contains(updated, "Updated personal guidance.") {
		t.Fatal("native save failed")
	}
	// Saving closes the editor. Reopening reads the saved file afresh.
	m, err = NewAgentInstructions(m.deck, m.store, "codex", "")
	if err != nil {
		t.Fatal(err)
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlF})
	m = next.(Model)
	if m.instructionEditor.document != nil || !strings.Contains(string(m.instructionEditor.text), "Updated personal guidance.") {
		t.Fatal("use-for-all failed to prepare the shared editor")
	}
	if m.instructionEditor.original != "Initial shared guidance." {
		t.Fatal("native save changed the central concurrency snapshot")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = next.(Model)
	next, command = m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = next.(Model)
	next, _ = m.Update(command())
	m = next.(Model)
	text, _ = manager.Load()
	if !strings.Contains(text, "Updated personal guidance.") {
		t.Fatal("imported instructions were not distributed")
	}
}

func TestInstructionsSaveReturnsHomeAndReopensSavedText(t *testing.T) {
	m := instructionFixture(t)
	m.instructionEditor.standalone = false
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	m = next.(Model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Türkçe yanıt ver.")})
	m = next.(Model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = next.(Model)
	next, save := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = next.(Model)
	next, quit := m.Update(save())
	m = next.(Model)
	if quit != nil || m.busy || m.instructionEditor != nil {
		t.Fatal("successful save did not return home")
	}
	m.openInstructions()
	f := m.instructionEditor
	if !strings.Contains(string(f.text), "Türkçe yanıt ver.") || !strings.Contains(string(f.text), "Initial shared guidance.") {
		t.Fatal("reopen did not load persisted shared text")
	}
	if f.review || f.adding || f.picking || f.document != nil || len(f.sharedDraft) != 0 || f.message != "" || f.offset != 0 {
		t.Fatal("stale editor state survived save")
	}
}

func TestInstructionsBackClosesEveryEditorMode(t *testing.T) {
	for _, mode := range []string{"shared", "add", "review", "existing", "picker"} {
		for _, mouse := range []bool{false, true} {
			t.Run(mode, func(t *testing.T) {
				m := instructionFixture(t)
				f := m.instructionEditor
				f.standalone = false
				f.sharedDraft = append([]rune(nil), f.text...)
				f.text = []rune("Unsaved temporary draft")
				switch mode {
				case "add":
					f.adding = true
				case "review":
					f.review = true
				case "existing":
					f.document = &instructions.Document{Agent: "codex"}
				case "picker":
					f.picking = true
				}
				var msg tea.Msg = tea.KeyMsg{Type: tea.KeyEsc}
				if mouse {
					msg = tea.MouseMsg{X: 4, Y: 2, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
				}
				next, command := m.Update(msg)
				m = next.(Model)
				if command != nil || m.instructionEditor != nil {
					t.Fatal("Back did not return home")
				}
				m.openInstructions()
				if string(m.instructionEditor.text) != "Initial shared guidance." {
					t.Fatal("unsaved draft survived Back")
				}
			})
		}
	}
}

func TestInstructionsFailedSaveKeepsDraftForRetry(t *testing.T) {
	m := instructionFixture(t)
	m.instructionEditor.text = []rune("Unsaved draft")
	m.instructionEditor.review = true
	m.busy = true
	next, command := m.Update(instructionsSavedMsg{err: fmt.Errorf("fixture write failed"), text: "Initial shared guidance."})
	m = next.(Model)
	if command != nil || m.busy || m.instructionEditor == nil || string(m.instructionEditor.text) != "Unsaved draft" {
		t.Fatal("failed save lost the draft")
	}
}
