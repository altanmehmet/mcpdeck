package tui

import (
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
	if m.busy || !strings.Contains(m.instructionEditor.message, "Saved") {
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

func TestSharedSelectionReplacementUndoAndReviewedClear(t *testing.T) {
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
	if err := store.AtomicWrite(doc.Path, []byte("Keep personal guidance.\n"+raw)); err != nil {
		t.Fatal(err)
	}
	send := func(k tea.KeyMsg) tea.Cmd { next, command := m.Update(k); m = next.(Model); return command }
	send(tea.KeyMsg{Type: tea.KeyCtrlA})
	send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Yeni talimat 日本語 🚀.")})
	if string(m.instructionEditor.text) != "Yeni talimat 日本語 🚀." {
		t.Fatal("selection appended instead of replacing")
	}
	send(tea.KeyMsg{Type: tea.KeyCtrlZ})
	if string(m.instructionEditor.text) != "Initial shared guidance." {
		t.Fatal("replacement could not be undone")
	}
	send(tea.KeyMsg{Type: tea.KeyCtrlA})
	send(tea.KeyMsg{Type: tea.KeyBackspace})
	if len(m.instructionEditor.text) != 0 {
		t.Fatal("select-all delete did not clear")
	}
	if current, _ := manager.Load(); current != "Initial shared guidance." {
		t.Fatal("clear wrote before review")
	}
	if command := send(tea.KeyMsg{Type: tea.KeyCtrlS}); command != nil || !strings.Contains(m.instructionEditor.message, "REMOVE") {
		t.Fatal("removal was not explicitly reviewed")
	}
	send(tea.KeyMsg{Type: tea.KeyCtrlU})
	if !m.instructionEditor.review {
		t.Fatal("clear escaped the review gate")
	}
	command := send(tea.KeyMsg{Type: tea.KeyCtrlS})
	if command == nil {
		t.Fatal("missing reviewed clear")
	}
	next, _ := m.Update(command())
	m = next.(Model)
	if current, _ := manager.Load(); current != "" {
		t.Fatal("shared guidance not cleared")
	}
	current, err := manager.ReadDocument(m.deck, doc)
	if err != nil || !strings.Contains(current, "Keep personal guidance.") || strings.Contains(current, "mcpdeck:global-instructions") {
		t.Fatal("clear damaged personal guidance", err)
	}
}

func TestNativeSelectAllReplacementPreservesSharedAndUndoScope(t *testing.T) {
	m := instructionFixture(t)
	manager := instructions.New(m.store)
	doc, err := manager.SelectDocument(m.deck, "codex", "")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := manager.ReadDocument(m.deck, doc)
	if err := store.AtomicWrite(doc.Path, []byte("Old personal guidance.\n"+raw)); err != nil {
		t.Fatal(err)
	}
	m, err = NewAgentInstructions(m.deck, m.store, "codex", "")
	if err != nil {
		t.Fatal(err)
	}
	send := func(k tea.KeyMsg) tea.Cmd { next, command := m.Update(k); m = next.(Model); return command }
	send(tea.KeyMsg{Type: tea.KeyCtrlA})
	send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("New personal guidance.")})
	if strings.Contains(string(m.instructionEditor.text), "Old personal") || !strings.Contains(string(m.instructionEditor.text), raw) {
		t.Fatal("native replacement lost the protected block")
	}
	send(tea.KeyMsg{Type: tea.KeyCtrlU})
	if strings.Contains(string(m.instructionEditor.text), "New personal") || !strings.Contains(string(m.instructionEditor.text), raw) {
		t.Fatal("native clear damaged shared guidance")
	}
	send(tea.KeyMsg{Type: tea.KeyCtrlZ})
	if !strings.Contains(string(m.instructionEditor.text), "New personal") {
		t.Fatal("native clear could not be undone")
	}
	if command := send(tea.KeyMsg{Type: tea.KeyCtrlS}); command != nil || !m.instructionEditor.review {
		t.Fatal("native replacement did not pass protected-block validation")
	}
	command := send(tea.KeyMsg{Type: tea.KeyCtrlS})
	if command == nil {
		t.Fatal("native replacement could not be saved")
	}
	next, _ := m.Update(command())
	m = next.(Model)
	current, err := manager.ReadDocument(m.deck, doc)
	if err != nil || !strings.Contains(current, "New personal") || !strings.Contains(current, raw) {
		t.Fatal("native save lost content", err)
	}
	if central, _ := manager.Load(); central != "Initial shared guidance." {
		t.Fatal("native edit changed all agents")
	}
	send(tea.KeyMsg{Type: tea.KeyCtrlG})
	send(tea.KeyMsg{Type: tea.KeyCtrlZ})
	if string(m.instructionEditor.text) != "Initial shared guidance." {
		t.Fatal("undo crossed editor scope")
	}
}

func TestInstructionReloadReadsExternalChangesAndProtectsDraft(t *testing.T) {
	m := instructionFixture(t)
	manager := instructions.New(m.store)
	if err := store.AtomicWrite(manager.SourcePath, []byte("Externally updated shared guidance.")); err != nil {
		t.Fatal(err)
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	m = next.(Model)
	if string(m.instructionEditor.text) != "Externally updated shared guidance." || m.instructionEditor.original != string(m.instructionEditor.text) {
		t.Fatal("shared reload did not refresh concurrency snapshot")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Draft.")})
	m = next.(Model)
	draft := string(m.instructionEditor.text)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	m = next.(Model)
	if string(m.instructionEditor.text) != draft || !strings.Contains(m.instructionEditor.message, "Unsaved") {
		t.Fatal("reload destroyed unsaved edits")
	}
	var err error
	m, err = NewAgentInstructions(m.deck, m.store, "codex", "")
	if err != nil {
		t.Fatal(err)
	}
	path := m.instructionEditor.document.Path
	updated := "External personal guidance.\n" + m.instructionEditor.documentOriginal
	if err := store.AtomicWrite(path, []byte(updated)); err != nil {
		t.Fatal(err)
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	m = next.(Model)
	if string(m.instructionEditor.text) != updated || m.instructionEditor.documentOriginal != updated {
		t.Fatal("agent reload did not refresh concurrency snapshot")
	}
}
