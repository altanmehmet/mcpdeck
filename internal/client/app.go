// Package client is the local desktop facade over MCPDeck's existing managers.
// It never exports server credentials or executes a mutation without a review.
package client

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/altanmehmet/mcpdeck/internal/install"
	"github.com/altanmehmet/mcpdeck/internal/instructions"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"github.com/altanmehmet/mcpdeck/internal/syncer"
)

type Agent struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Detected   bool   `json:"detected"`
	Enabled    int    `json:"enabled"`
	Mode       string `json:"mode"`
	SyncStatus string `json:"syncStatus"`
}
type Server struct {
	Name     string   `json:"name"`
	Kind     string   `json:"kind"`
	Managed  bool     `json:"managed"`
	Profiles []string `json:"profiles"`
	Enabled  int      `json:"enabled"`
}
type State struct {
	Demo             bool                    `json:"demo"`
	Config           string                  `json:"config"`
	Shared           string                  `json:"shared"`
	Agents           []Agent                 `json:"agents"`
	Servers          []Server                `json:"servers"`
	Documents        []instructions.Document `json:"documents"`
	InstructionFiles []InstructionFile       `json:"instructionFiles"`
	Targets          []instructions.Result   `json:"targets"`
	Planner          install.PlannerOptions  `json:"planner"`
	Warnings         []string                `json:"warnings"`
}
type Edit struct {
	Agent    string `json:"agent"`
	Path     string `json:"path"`
	Text     string `json:"text"`
	Expected string `json:"expected"`
	Current  string `json:"current"`
	Personal bool   `json:"personal"`
	Append   bool   `json:"append"`
}
type Review struct {
	Token    string                `json:"token"`
	Title    string                `json:"title"`
	Current  string                `json:"current"`
	Proposed string                `json:"proposed"`
	Results  []instructions.Result `json:"results"`
	Detail   string                `json:"detail"`
}
type Result struct {
	Message string                `json:"message"`
	Results []instructions.Result `json:"results"`
	Partial bool                  `json:"partial"`
}
type pending struct {
	review                       Review
	edit                         *Edit
	doc                          *instructions.Document
	action, name                 string
	profile                      string
	targets                      []string
	restoreTarget, restoreBackup []byte
	enabled                      bool
	external                     *syncer.DiscoveredServer
	deckHash                     string
	expires                      time.Time
}
type App struct {
	demo    bool
	dirty   bool
	mu      sync.Mutex
	s       store.Store
	manager instructions.Manager
	pending *pending
	ctx     context.Context
	cancel  context.CancelFunc
	plan    *planned
	job     *job
}

func New(s store.Store) *App {
	return &App{s: s, manager: instructions.New(s), ctx: context.Background()}
}
func (a *App) Close() { a.Cancel(); a.mu.Lock(); a.pending = nil; a.plan = nil; a.mu.Unlock() }
func token() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("secure randomness unavailable")
	}
	return hex.EncodeToString(b)
}
func deckHash(d *model.Deck) string {
	b, _ := json.Marshal(d)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
func (a *App) loadPlanner() (install.PlannerOptions, error) {
	o := install.PlannerOptions{Provider: "codex"}
	b, err := os.ReadFile(filepath.Join(filepath.Dir(a.s.Path), "planner.json"))
	if os.IsNotExist(err) {
		return o, nil
	}
	if err != nil || len(b) > 64*1024 || json.Unmarshal(b, &o) != nil {
		return o, fmt.Errorf("Cannot read planner settings. Check mcpdeck planner.")
	}
	return o, o.Validate()
}
func (a *App) Snapshot() (State, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	d, err := a.s.Load()
	if err != nil {
		return State{}, fmt.Errorf("Cannot read MCPDeck configuration. Check mcpdeck doctor.")
	}
	text, err := a.manager.Load()
	if err != nil {
		return State{}, fmt.Errorf("Cannot read shared instructions.")
	}
	state := State{Demo: a.demo, Config: a.s.Path, Shared: text, Agents: []Agent{}, Servers: []Server{}, Warnings: []string{}}
	state.Documents, err = a.manager.Documents(d)
	if err != nil {
		state.Warnings = append(state.Warnings, "Some instruction files could not be inspected.")
	}
	state.InstructionFiles = instructionFileStates(state.Documents)
	state.Targets, err = a.manager.Status(d, false)
	if err != nil {
		return State{}, err
	}
	state.Planner, err = a.loadPlanner()
	if err != nil {
		state.Warnings = append(state.Warnings, err.Error())
	}
	for name, p := range d.Profiles {
		mode := p.Mode
		if mode == "" {
			mode = "direct"
		}
		state.Agents = append(state.Agents, Agent{Name: name, Path: p.TargetPath, Detected: store.AgentDetected(name, p), Enabled: len(p.EnabledServers), Mode: mode})
	}
	reports, e := (syncer.Syncer{Deck: d, ConfigPath: a.s.Path}).Results()
	if e != nil {
		state.Warnings = append(state.Warnings, "Cannot read the last synchronization report.")
	} else {
		for i := range state.Agents {
			for _, r := range reports {
				if r.Profile == state.Agents[i].Name && r.TargetPath == syncer.ExpandPath(state.Agents[i].Path) {
					state.Agents[i].SyncStatus = r.Status
				}
			}
		}
	}
	sort.Slice(state.Agents, func(i, j int) bool { return state.Agents[i].Name < state.Agents[j].Name })
	for name, c := range d.Servers {
		kind := "Local stdio"
		if c.URL != "" {
			kind = "Remote MCP"
		}
		item := Server{Name: name, Kind: kind, Managed: true, Profiles: []string{}}
		for key := range d.Profiles {
			if d.IsEnabled(key, name) {
				item.Profiles = append(item.Profiles, key)
				item.Enabled++
			}
		}
		sort.Strings(item.Profiles)
		state.Servers = append(state.Servers, item)
	}
	external, err := syncer.Discover(d)
	if err != nil {
		state.Warnings = append(state.Warnings, "Some agent configurations could not be inspected. Discovered MCP inventory may be incomplete.")
	}
	for name, item := range external {
		state.Servers = append(state.Servers, Server{name, "Agent configuration", false, item.Profiles, len(item.Profiles)})
	}
	sort.Slice(state.Servers, func(i, j int) bool { return state.Servers[i].Name < state.Servers[j].Name })
	return state, nil
}
func (a *App) ReadDocument(agent, path string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	d, err := a.s.Load()
	if err != nil {
		return "", err
	}
	doc, err := a.manager.SelectDocument(d, agent, path)
	if err != nil {
		return "", err
	}
	return a.manager.ReadDocument(d, doc)
}
func (a *App) ReviewInstructions(edit Edit) (Review, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancel != nil {
		return Review{}, fmt.Errorf("Wait for the current operation to finish.")
	}
	d, err := a.s.Load()
	if err != nil {
		return Review{}, err
	}
	r := Review{Token: token(), Title: "Review shared instructions", Detail: "Replace only the MCPDeck section. Keep personal and project rules."}
	var doc *instructions.Document
	if edit.Agent != "" {
		selected, e := a.manager.SelectDocument(d, edit.Agent, edit.Path)
		if e != nil {
			return Review{}, e
		}
		doc = &selected
		r.Current, e = a.manager.ReadDocument(d, selected)
		if e != nil {
			return Review{}, e
		}
		if edit.Append {
			return Review{}, fmt.Errorf("Append is available in shared guidance only.")
		}
		if edit.Personal {
			if r.Current != edit.Expected {
				return Review{}, fmt.Errorf("This file changed. Reload before editing.")
			}
			edit.Text, e = instructions.ReplacePersonalText(r.Current, edit.Text)
			if e != nil {
				return Review{}, e
			}
		}
		if e = a.manager.CheckDocument(d, selected, edit.Text, edit.Expected); e != nil {
			return Review{}, e
		}
		r.Proposed = edit.Text
		r.Title = "Review " + edit.Agent + " instructions"
		r.Detail = "Update this global file only. Agents sharing this exact file also see the change."
		r.Results = []instructions.Result{}
		documents, _ := a.manager.Documents(d)
		seen := map[string]bool{}
		for _, doc := range documents {
			if doc.Path == selected.Path && !seen[doc.Agent] {
				seen[doc.Agent] = true
				r.Results = append(r.Results, instructions.Result{Agent: doc.Agent, Path: selected.Path, Status: "update this file"})
			}
		}
	} else {
		r.Current, err = a.manager.Load()
		if err != nil {
			return Review{}, err
		}
		if r.Current != edit.Expected {
			return Review{}, fmt.Errorf("Shared instructions changed. Reload before editing.")
		}
		r.Proposed = edit.Text
		if edit.Append {
			r.Proposed, err = instructions.AppendText(r.Current, edit.Text)
			if err != nil {
				return Review{}, err
			}
			r.Detail = "Append to shared guidance. Other personal text stays unchanged."
		}
		r.Results, err = a.manager.Preview(d, r.Proposed, strings.TrimSpace(r.Proposed) == "")
		if err != nil {
			return Review{}, err
		}
		if strings.TrimSpace(r.Proposed) == "" {
			r.Detail = "Remove only MCPDeck's shared sections from all configured targets. Other personal text stays unchanged."
		}
	}
	edit.Text = r.Proposed
	if edit.Personal && edit.Agent != "" {
		r.Current, err = instructions.PersonalText(r.Current)
		if err != nil {
			return Review{}, err
		}
		r.Proposed, err = instructions.PersonalText(r.Proposed)
		if err != nil {
			return Review{}, err
		}
	}
	a.pending = &pending{review: r, edit: &edit, doc: doc, deckHash: deckHash(d), expires: time.Now().Add(10 * time.Minute)}
	return r, nil
}
func (a *App) PreparePersonalReplacement(agent, path, text string) (string, error) {
	raw, err := a.ReadDocument(agent, path)
	if err != nil {
		return "", err
	}
	return instructions.ReplacePersonalText(raw, text)
}
func (a *App) ImportShared(text string) (string, error) { return instructions.SharedText(text) }
func (a *App) ReviewServer(name, action string) (Review, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancel != nil {
		return Review{}, fmt.Errorf("Wait for the current operation to finish.")
	}
	d, err := a.s.Load()
	if err != nil {
		return Review{}, err
	}
	if action != "enable" && action != "disable" && action != "remove" {
		return Review{}, fmt.Errorf("Unknown MCP action.")
	}
	r := Review{Token: token(), Title: strings.Title(action) + " " + name, Detail: "Update configured agent settings. Restart their MCP connections afterward."}
	p := &pending{review: r, action: action, name: name, enabled: action == "enable", deckHash: deckHash(d), expires: time.Now().Add(10 * time.Minute)}
	if _, ok := d.Servers[name]; !ok {
		if action != "remove" {
			return Review{}, fmt.Errorf("This MCP belongs to agent settings. Only reviewed removal is available.")
		}
		found, e := syncer.Discover(d)
		if e != nil {
			return Review{}, fmt.Errorf("Cannot fully inspect agents. Resolve configuration errors before removing this MCP.")
		}
		item, ok := found[name]
		if !ok {
			return Review{}, fmt.Errorf("MCP no longer exists.")
		}
		p.external = &item
		for _, key := range item.Profiles {
			r.Results = append(r.Results, instructions.Result{Agent: key, Path: d.Profiles[key].TargetPath, Status: action})
		}
	} else {
		for key, profile := range d.Profiles {
			r.Results = append(r.Results, instructions.Result{Agent: key, Path: profile.TargetPath, Status: action})
		}
	}
	if action == "remove" {
		r.Detail = "Remove this MCP from agent configurations. Installed programs and shared runtimes are kept."
	}
	sort.Slice(r.Results, func(i, j int) bool { return r.Results[i].Agent < r.Results[j].Agent })
	p.review = r
	a.pending = p
	return r, nil
}
func (a *App) ApplyReview(id string) (Result, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	p := a.pending
	if p == nil || p.review.Token != id || time.Now().After(p.expires) {
		return Result{}, fmt.Errorf("Review expired. Review the current changes again.")
	}
	if a.cancel != nil {
		return Result{}, fmt.Errorf("Wait for the current operation to finish.")
	}
	d, err := a.s.Load()
	if err != nil {
		return Result{}, err
	}
	if deckHash(d) != p.deckHash {
		return Result{}, fmt.Errorf("Agent configuration changed. Review the current targets again.")
	}
	a.pending = nil
	if p.edit != nil {
		var results []instructions.Result
		if p.doc != nil {
			err = a.manager.SaveDocument(d, *p.doc, p.edit.Text, p.edit.Expected)
			results = p.review.Results
		} else {
			results, err = a.manager.Apply(d, p.edit.Text, strings.TrimSpace(p.edit.Text) == "", &p.edit.Expected)
		}
		if err != nil {
			if p.doc != nil || len(results) == 0 {
				return Result{}, err
			}
			return Result{Message: "Save or distribution incomplete: " + err.Error(), Results: results, Partial: true}, nil
		}
		return Result{"Saved. Reload agent instructions or start new sessions.", results, false}, nil
	}
	if p.profile != "" || len(p.targets) > 0 {
		return a.applyAgent(d, p)
	}
	if p.action == "remove" {
		if p.external != nil {
			err = syncer.RemoveDiscovered(a.s, *p.external)
		} else {
			err = syncer.Remove(a.s, p.name)
		}
	} else {
		err = a.s.Update(func(current *model.Deck) error {
			if deckHash(current) != p.deckHash {
				return fmt.Errorf("Configuration changed. Review again.")
			}
			return current.SetEnabled("", p.name, p.enabled)
		})
		if err == nil {
			d, err = a.s.Load()
			if err == nil {
				err = (syncer.Syncer{Deck: d, ConfigPath: a.s.Path}).SyncAll()
			}
		}
	}
	if err != nil {
		return Result{Message: "Operation incomplete. Some targets may have changed: " + err.Error(), Partial: true}, nil
	}
	return Result{Message: "Agent settings updated. Restart their MCP connections."}, nil
}
func (a *App) SavePlanner(o install.PlannerOptions) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.demo {
		return fmt.Errorf("Planner settings are read-only in the sample workspace.")
	}
	if a.cancel != nil {
		return fmt.Errorf("Wait for the current operation to finish.")
	}
	if err := o.Validate(); err != nil {
		return err
	}
	o.APIKey = ""
	b, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		return err
	}
	return store.AtomicWrite(filepath.Join(filepath.Dir(a.s.Path), "planner.json"), b)
}

type PersonalDocument struct {
	Text     string `json:"text"`
	Expected string `json:"expected"`
	Current  string `json:"current"`
}

func (a *App) ReadPersonalDocument(agent, path string) (PersonalDocument, error) {
	raw, err := a.ReadDocument(agent, path)
	if err != nil {
		return PersonalDocument{}, err
	}
	text, err := instructions.PersonalText(raw)
	if err != nil {
		return PersonalDocument{}, err
	}
	current, err := instructions.CurrentText(raw)
	return PersonalDocument{Text: text, Expected: raw, Current: current}, err
}

func (a *App) SetDirty(value bool) { a.mu.Lock(); a.dirty = value; a.mu.Unlock() }
func (a *App) NeedsCloseConfirmation() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.dirty || a.cancel != nil
}
