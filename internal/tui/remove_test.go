package tui

import (
	"github.com/altanmehmet/mcpdeck/internal/store"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/altanmehmet/mcpdeck/internal/model"
)

func TestRemoveNeedsConfirmationAndSyncs(t *testing.T) {
	m := mouseFixture(t)
	for name, p := range store.AdditionalProfiles() {
		p.TargetPath = filepath.Join(t.TempDir(), "agent.json")
		m.deck.Profiles[name] = p
	}
	m = New(m.deck, m.store)
	if len(m.servers) == 0 {
		t.Fatal("missing fixture")
	}
	name := m.servers[m.cursor]
	if err := m.store.Save(m.deck); err != nil {
		t.Fatal(err)
	}
	next, _ := m.Update(keyMessage("x"))
	m = next.(Model)
	if m.removing != name {
		t.Fatal("no confirmation screen")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(Model)
	d, _ := m.store.Load()
	if _, ok := d.Servers[name]; !ok {
		t.Fatal("cancel removed record")
	}
	// Ensure an existing direct configuration must actually be cleaned up.
	m.changeProfiles(func(d *model.Deck) error { return d.SetEnabled("", name, true) }, true)
	next, _ = m.Update(keyMessage("x"))
	m = next.(Model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)
	d, err := m.store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := d.Servers[name]; ok {
		t.Fatal("confirmed removal left record")
	}
	for profile := range d.Profiles {
		if d.IsEnabled(profile, name) {
			t.Fatal("selection left behind")
		}
	}
}
