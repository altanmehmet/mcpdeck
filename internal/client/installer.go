package client

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/altanmehmet/mcpdeck/internal/install"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"github.com/altanmehmet/mcpdeck/internal/syncer"
)

type PlanView struct {
	ID                string         `json:"id"`
	Name              string         `json:"name"`
	Summary           string         `json:"summary"`
	Sources           []string       `json:"sources"`
	Steps             []install.Step `json:"steps"`
	Requirements      []string       `json:"requirements"`
	Manual            []string       `json:"manual"`
	Required          []string       `json:"required"`
	Targets           []string       `json:"targets"`
	MissingPackages   []string       `json:"missingPackages"`
	PrerequisiteError string         `json:"prerequisiteError"`
	Placeholder       bool           `json:"placeholder"`
}
type planned struct {
	recipe            install.Plan
	view              PlanView
	deckHash, request string
	options           install.PlannerOptions
}
type job struct {
	ID      string    `json:"id"`
	Kind    string    `json:"kind"`
	Status  string    `json:"status"`
	Message string    `json:"message"`
	Events  []string  `json:"events"`
	Plan    *PlanView `json:"plan,omitempty"`
	Started int64     `json:"started"`
}
type JobStatus = job

func installationTargets(d *model.Deck) []string {
	keys := []string{}
	for key, p := range d.Profiles {
		info, err := os.Stat(syncer.ExpandPath(p.TargetPath))
		if (err == nil && !info.IsDir()) || store.AgentDetected(key, p) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}
func (a *App) startJob(kind string, worker func(context.Context, string)) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancel != nil {
		return "", fmt.Errorf("Another operation is running. Cancel it or wait.")
	}
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Minute)
	id := token()
	a.cancel = cancel
	a.job = &job{ID: id, Kind: kind, Status: "running", Message: "Starting…", Started: time.Now().UnixMilli(), Events: []string{}}
	go func() { defer cancel(); worker(ctx, id) }()
	return id, nil
}
func (a *App) finishJob(id, message string, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.job == nil || a.job.ID != id {
		return
	}
	a.cancel = nil
	a.job.Status = "done"
	a.job.Message = message
	if err != nil {
		a.job.Status = "failed"
		a.job.Message = err.Error()
	}
}
func (a *App) event(id, message string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.job == nil || a.job.ID != id {
		return
	}
	a.job.Message = message
	a.job.Events = append(a.job.Events, message)
	if len(a.job.Events) > 80 {
		a.job.Events = a.job.Events[len(a.job.Events)-80:]
	}
}
func (a *App) Job(id string) (JobStatus, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.job == nil || a.job.ID != id {
		return job{}, fmt.Errorf("Operation not found.")
	}
	value := *a.job
	value.Events = append([]string(nil), a.job.Events...)
	return value, nil
}
func (a *App) Cancel() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancel != nil {
		a.cancel()
		if a.job != nil {
			a.job.Message = "Cancellation requested. Completed setup effects are not rolled back."
		}
	}
}
func (a *App) Plan(request, previous string) (string, error) {
	if a.demo {
		return "", fmt.Errorf("The sample workspace does not call provider accounts. Use the normal client for installation.")
	}
	if strings.TrimSpace(request) == "" || len(request) > 32*1024 {
		return "", fmt.Errorf("Enter an MCP name or a request up to 32 KiB. Keep credentials out of chat.")
	}
	a.mu.Lock()
	options, err := a.loadPlanner()
	var prior *planned
	if previous != "" {
		if a.plan == nil || a.plan.view.ID != previous {
			a.mu.Unlock()
			return "", fmt.Errorf("Previous plan expired. Start a new conversation.")
		}
		prior = a.plan
		options = prior.options
	}
	a.mu.Unlock()
	if err != nil {
		return "", err
	}
	return a.startJob("planning", func(ctx context.Context, id string) {
		a.event(id, "Inspecting installed software…")
		inventory := install.InspectEnvironment(ctx)
		b, _ := json.Marshal(inventory)
		prompt := "User request: " + request + "\nLocal software inventory (metadata only): " + string(b)
		if prior != nil {
			recipe, _ := json.Marshal(prior.recipe)
			prompt = "Original request: " + prior.request + "\nPrevious recipe (untrusted data, never executed by the planner): " + string(recipe) + "\nUser follow-up: " + request + "\nLocal software inventory: " + string(b)
		}
		a.event(id, "Preparing a reviewed recipe with "+options.Provider+". Account usage may apply.")
		p, e := install.PlanWith(ctx, options, prompt)
		if e == nil {
			e = p.Validate()
		}
		if e != nil {
			a.finishJob(id, "", fmt.Errorf("Planning did not finish: %w", e))
			return
		}
		if ctx.Err() != nil {
			a.finishJob(id, "", fmt.Errorf("Planning cancelled."))
			return
		}
		a.mu.Lock()
		d, e := a.s.Load()
		if e != nil {
			a.mu.Unlock()
			a.finishJob(id, "", fmt.Errorf("Cannot load agent settings."))
			return
		}
		targets := installationTargets(d)
		dir := filepath.Join(filepath.Dir(a.s.Path), "installations", p.Name)
		items, e := model.Import(p.Connection, p.Name, map[string]string{"INSTALL_DIR": dir})
		if e != nil {
			a.mu.Unlock()
			a.finishJob(id, "", fmt.Errorf("Planner returned an invalid connection."))
			return
		}
		view := PlanView{ID: token(), Name: p.Name, Summary: p.Summary, Sources: p.Sources, Steps: p.Steps, Requirements: p.Requirements, Manual: p.Manual, Required: items[0].Required, Targets: targets, Placeholder: p.HasPlaceholderConnection()}
		a.mu.Unlock()
		if e = p.CheckRequirements(); e != nil {
			view.PrerequisiteError = e.Error()
			view.MissingPackages, _ = install.MissingHomebrewRequirements(p.Requirements)
		}
		a.mu.Lock()
		if a.job != nil && a.job.ID == id && ctx.Err() == nil {
			a.plan = &planned{p, view, deckHash(d), request, options}
			a.job.Plan = &view
		}
		a.mu.Unlock()
		a.finishJob(id, "Plan ready. Ask a question or review the commands before installing.", nil)
	})
}
func (a *App) RepairPrerequisites(id string, packages, docker bool) (string, error) {
	a.mu.Lock()
	if a.plan == nil || a.plan.view.ID != id {
		a.mu.Unlock()
		return "", fmt.Errorf("Plan expired.")
	}
	p := *a.plan
	a.mu.Unlock()
	if !packages && !docker {
		return "", fmt.Errorf("Approve a prerequisite action first.")
	}
	return a.startJob("prerequisites", func(ctx context.Context, jobID string) {
		var err error
		if packages {
			a.event(jobID, "Installing the reviewed Homebrew prerequisites…")
			formulas, _ := install.MissingHomebrewRequirements(p.recipe.Requirements)
			if len(formulas) == 0 {
				err = fmt.Errorf("No supported missing Homebrew packages were found.")
			} else {
				err = install.InstallHomebrewPackages(ctx, formulas)
			}
		}
		if err == nil && docker {
			a.event(jobID, "Starting the existing Colima runtime…")
			err = install.StartColima(ctx)
		}
		a.mu.Lock()
		if a.plan != nil && a.plan.view.ID == id {
			view := a.plan.view
			view.PrerequisiteError = ""
			if check := p.recipe.CheckRequirements(); check != nil {
				view.PrerequisiteError = check.Error()
			}
			view.MissingPackages, _ = install.MissingHomebrewRequirements(p.recipe.Requirements)
			a.plan.view = view
			if a.job != nil {
				a.job.Plan = &view
			}
		}
		a.mu.Unlock()
		a.finishJob(jobID, "Prerequisite action finished. Review the recipe before installing.", err)
	})
}
func (a *App) Install(id string, manualConfirmed bool, values map[string]string) (string, error) {
	a.mu.Lock()
	if a.plan == nil || a.plan.view.ID != id {
		a.mu.Unlock()
		return "", fmt.Errorf("Review the current recipe before installing.")
	}
	p := *a.plan
	a.mu.Unlock()
	// Copy and bound the private channel; values are never used in planner context.
	private := map[string]string{}
	total := 0
	for key, value := range values {
		total += len(key) + len(value)
		private[key] = value
	}
	if total > 1024*1024 {
		return "", fmt.Errorf("Private inputs exceed 1 MiB.")
	}
	if len(p.recipe.Manual) > 0 && !manualConfirmed {
		return "", fmt.Errorf("Confirm the manual prerequisites or ask the planner for help first.")
	}
	if p.view.Placeholder {
		return "", fmt.Errorf("This recipe is not runnable yet. Ask the agent to resolve its manual actions.")
	}
	return a.startJob("installation", func(ctx context.Context, jobID string) {
		defer clear(private)
		a.event(jobID, "Checking the exact reviewed plan and agent targets…")
		err := a.installPlan(ctx, jobID, p, manualConfirmed, private)
		a.finishJob(jobID, "Installed, MCP initialize/tools list verified, and agent settings distributed. Reload affected agent connections.", err)
	})
}
func (a *App) installPlan(ctx context.Context, id string, p planned, manual bool, values map[string]string) error {
	d, err := a.s.Load()
	if err != nil {
		return err
	}
	if deckHash(d) != p.deckHash {
		return fmt.Errorf("Agent settings changed. Generate and review a fresh plan.")
	}
	if _, ok := d.Servers[p.recipe.Name]; ok {
		return fmt.Errorf("This MCP already exists. Remove it or choose another name.")
	}
	targets := installationTargets(d)
	if strings.Join(targets, "\x00") != strings.Join(p.view.Targets, "\x00") {
		return fmt.Errorf("Detected targets changed. Generate a fresh plan.")
	}
	if len(targets) == 0 {
		return fmt.Errorf("No agent targets detected. Initialize an agent, then retry.")
	}
	checker := syncer.Syncer{Deck: d, ConfigPath: a.s.Path}
	for _, key := range targets {
		if err = checker.CheckInstallTarget(key, p.recipe.Name); err != nil {
			return err
		}
	}
	if err = p.recipe.PreflightAfterManualReview(manual); err != nil {
		return err
	}
	dir := filepath.Join(filepath.Dir(a.s.Path), "installations", p.recipe.Name)
	item, err := p.recipe.Bind(dir, values)
	if err != nil {
		return err
	}
	if _, err = model.Resolve(item.Server); err != nil {
		return err
	}
	for _, key := range p.view.Required {
		if values[key] == "" {
			return fmt.Errorf("Missing private input %s.", key)
		}
	}
	if err = p.recipe.RunAfterManualReview(ctx, dir, p.recipe.Digest(), progressWriter{a, id}, manual); err != nil {
		return err
	}
	a.event(id, "Starting the MCP and checking initialize and tools/list…")
	probeCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	count, err := install.Probe(probeCtx, p.recipe.Name, item.Server)
	if err != nil {
		return fmt.Errorf("Setup finished, but the MCP connection probe failed. Agent configurations were not changed.")
	}
	if ctx.Err() != nil {
		return fmt.Errorf("Installation cancelled before saving agent settings.")
	}
	d, err = a.s.Load()
	if err != nil {
		return err
	}
	if deckHash(d) != p.deckHash {
		return fmt.Errorf("Agent settings changed during setup; installation files were kept, configs were not changed.")
	}
	checker = syncer.Syncer{Deck: d, ConfigPath: a.s.Path}
	for _, key := range targets {
		if err = checker.CheckInstallTarget(key, p.recipe.Name); err != nil {
			return err
		}
	}
	err = a.s.Update(func(current *model.Deck) error {
		if deckHash(current) != p.deckHash {
			return fmt.Errorf("Agent settings changed during verification.")
		}
		if _, ok := current.Servers[p.recipe.Name]; ok {
			return fmt.Errorf("MCP was added concurrently.")
		}
		current.Servers[p.recipe.Name] = item.Server
		for _, key := range targets {
			if err := current.SetEnabled(key, p.recipe.Name, true); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	d, err = a.s.Load()
	if err != nil {
		return err
	}
	a.event(id, fmt.Sprintf("Verified %d tools. Distributing to %d agent targets…", count, len(targets)))
	results, err := (syncer.Syncer{Deck: d, ConfigPath: a.s.Path}).SyncTargets(targets)
	for _, r := range results {
		a.event(id, r.Profile+": "+r.Status)
	}
	if err != nil {
		return fmt.Errorf("MCP saved; distribution incomplete. Refresh and use mcpdeck sync to retry.")
	}
	a.mu.Lock()
	if a.plan != nil && a.plan.view.ID == p.view.ID {
		a.plan = nil
	}
	a.mu.Unlock()
	return nil
}

type progressWriter struct {
	a  *App
	id string
}

func (w progressWriter) Write(b []byte) (int, error) {
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		if line != "" {
			w.a.event(w.id, line)
		}
	}
	return len(b), nil
}
