package tui

import (
	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"github.com/altanmehmet/mcpdeck/internal/syncer"
	tea "github.com/charmbracelet/bubbletea"
	"sort"
)

type Model struct {
	deck                  *model.Deck
	store                 store.Store
	profiles, servers     []string
	tab, cursor           int
	status                string
	removing              string
	external              map[string]syncer.DiscoveredServer
	discoveryWarning      string
	width, height         int
	offset, profileOffset int
	form                  *importForm
	instructionEditor     *instructionEditor
	help, busy            bool
	textSelection         bool
	mouseCapture          bool
	copyStart             *mousePoint
	keys                  keyPreferences
	keyEditor             *keyEditor
	helpPage              int
}

func New(d *model.Deck, s store.Store) Model {
	return NewWithMouse(d, s, true)
}

// NewWithMouse mirrors Bubble Tea's mouse mode so text selection can be
// enabled and disabled while the app is running.
func NewWithMouse(d *model.Deck, s store.Store, mouseEnabled bool) Model {
	m := Model{deck: d, store: s, width: 100, height: 30, mouseCapture: mouseEnabled}
	m.textSelection = !mouseEnabled
	var prefErr error
	m.keys, prefErr = readKeyPreferences(s)
	if prefErr != nil {
		m.keys = defaultKeyPreferences()
		m.status = "Could not read key settings; using defaults."
	}
	for _, k := range []string{"cursor", "claude", "claude-code", "codex", "copilot", "copilot-cli", "windsurf", "antigravity"} {
		if _, ok := d.Profiles[k]; ok {
			m.profiles = append(m.profiles, k)
		}
	}
	var extra []string
	for k := range d.Profiles {
		known := false
		for _, existing := range m.profiles {
			if existing == k {
				known = true
				break
			}
		}
		if !known {
			extra = append(extra, k)
		}
	}
	sort.Strings(extra)
	m.profiles = append(m.profiles, extra...)
	for k := range d.Servers {
		m.servers = append(m.servers, k)
	}
	m.external, prefErr = syncer.Discover(d)
	if prefErr != nil {
		m.discoveryWarning = "Some agent settings could not be read; discovery is incomplete."
	}
	for name := range m.external {
		m.servers = append(m.servers, name)
	}
	sort.Strings(m.servers)
	return m
}
func (m Model) Init() tea.Cmd { return nil }
func (m *Model) change(fn func(*model.Deck) error) {
	m.changeProfiles(fn, false)
}
func (m *Model) changeProfiles(fn func(*model.Deck) error, all bool) {
	if err := m.store.Update(fn); err != nil {
		m.status = err.Error()
		return
	}
	d, err := m.store.Load()
	if err != nil {
		m.status = err.Error()
		return
	}
	m.deck = d
	sy := syncer.Syncer{Deck: d, ConfigPath: m.store.Path}
	if all {
		err = sy.SyncAll()
	} else {
		_, err = sy.SyncTargets([]string{m.profiles[m.tab]})
	}
	if err != nil {
		m.status = "Saved; sync failed: " + err.Error() + ". Use Retry to retry failed targets."
	} else if all {
		m.status = "Saved and synced in all profiles. Restart agent MCP connections."
	} else {
		m.status = "Saved and synced."
	}
}
