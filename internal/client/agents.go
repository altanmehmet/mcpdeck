package client

import (
	"fmt"
	"github.com/altanmehmet/mcpdeck/internal/instructions"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/syncer"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// ReviewAgent scopes desktop changes to one explicit agent. It never writes files.
func (a *App) ReviewAgent(profile, action, server string) (Review, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancel != nil {
		return Review{}, fmt.Errorf("Wait for the current operation to finish.")
	}
	d, err := a.s.Load()
	if err != nil {
		return Review{}, err
	}
	p, ok := d.Profiles[profile]
	if !ok {
		return Review{}, fmt.Errorf("Unknown agent profile.")
	}
	title, detail := "", ""
	switch action {
	case "enable", "disable":
		if _, ok := d.Servers[server]; !ok {
			return Review{}, fmt.Errorf("Only MCPDeck-managed servers can be assigned to an agent.")
		}
		title = action + " " + server + " for " + profile
		detail = "Change this agent only, then synchronize its configuration. Other agents keep their selections."
	case "direct", "bridge":
		title = "Use " + action + " mode for " + profile
		detail = "Save the selected mode and synchronize this agent. Existing configuration is backed up. Restart its MCP connection afterward."
	case "sync":
		title = "Synchronize " + profile
		detail = "Apply saved server selections to this agent, with a backup of the previous configuration."
	default:
		return Review{}, fmt.Errorf("Unknown agent action.")
	}
	r := Review{Token: token(), Title: title, Detail: detail, Results: []instructions.Result{{Agent: profile, Path: p.TargetPath, Status: action}}}
	a.pending = &pending{review: r, profile: profile, name: server, action: action, enabled: action == "enable", deckHash: deckHash(d), expires: time.Now().Add(10 * time.Minute)}
	return r, nil
}

// ReviewRecovery binds retry to the actual failed targets, not every profile.
func (a *App) ReviewRecovery(action, profile string) (Review, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancel != nil {
		return Review{}, fmt.Errorf("Wait for the current operation to finish.")
	}
	if action != "retry" && action != "restore" {
		return Review{}, fmt.Errorf("Unknown recovery action.")
	}
	d, err := a.s.Load()
	if err != nil {
		return Review{}, err
	}
	if action == "restore" {
		cfg, ok := d.Profiles[profile]
		if !ok {
			return Review{}, fmt.Errorf("Unknown agent profile.")
		}
		path, err := filepath.EvalSymlinks(syncer.ExpandPath(cfg.TargetPath))
		if err != nil {
			return Review{}, fmt.Errorf("An existing configuration is required for restore.")
		}
		current, err := os.ReadFile(path)
		if err != nil {
			return Review{}, fmt.Errorf("Cannot read current configuration.")
		}
		backup, err := os.ReadFile(path + ".mcpdeck-backup")
		if err != nil {
			return Review{}, fmt.Errorf("No readable backup is available for this agent.")
		}
		if err = syncer.ValidateConfig(backup, cfg.Format); err != nil {
			return Review{}, fmt.Errorf("Backup is invalid; restore refused.")
		}
		r := Review{Token: token(), Title: "Restore " + profile + " configuration", Detail: "Replace the current agent configuration with its last sync backup. Current settings become the new backup. MCPDeck selections remain unchanged; restart the agent afterward.", Results: []instructions.Result{{Agent: profile, Path: cfg.TargetPath, Status: "restore last backup"}}}
		a.pending = &pending{review: r, profile: profile, action: action, restoreTarget: current, restoreBackup: backup, deckHash: deckHash(d), expires: time.Now().Add(10 * time.Minute)}
		return r, nil
	}
	reports, err := (syncer.Syncer{Deck: d, ConfigPath: a.s.Path}).Results()
	if err != nil {
		return Review{}, err
	}
	targets := []string{}
	rows := []instructions.Result{}
	for _, r := range reports {
		if r.Status != "failed" || (profile != "" && profile != r.Profile) {
			continue
		}
		p, ok := d.Profiles[r.Profile]
		if !ok || syncer.ExpandPath(p.TargetPath) != r.TargetPath {
			return Review{}, fmt.Errorf("A failed target changed. Review and sync it individually.")
		}
		targets = append(targets, r.Profile)
		rows = append(rows, instructions.Result{Agent: r.Profile, Path: r.TargetPath, Status: "retry synchronization"})
	}
	if len(targets) == 0 {
		return Review{}, fmt.Errorf("No failed synchronization to retry. Refresh agent status.")
	}
	sort.Strings(targets)
	r := Review{Token: token(), Title: "Retry failed synchronization", Detail: "Retry only the targets listed below, using the current saved selections.", Results: rows}
	a.pending = &pending{review: r, action: action, targets: targets, deckHash: deckHash(d), expires: time.Now().Add(10 * time.Minute)}
	return r, nil
}
func (a *App) applyAgent(d *model.Deck, p *pending) (Result, error) {
	if p.action == "restore" {
		err := (syncer.Syncer{Deck: d, ConfigPath: a.s.Path}).RestoreIfUnchanged(p.profile, p.restoreTarget, p.restoreBackup)
		if err != nil {
			return Result{}, err
		}
		return Result{Message: "Backup restored. Restart this agent. Saved MCPDeck selections were not changed."}, nil
	}

	if p.action != "sync" && p.action != "retry" {
		err := a.s.Update(func(current *model.Deck) error {
			if deckHash(current) != p.deckHash {
				return fmt.Errorf("Configuration changed. Review again.")
			}
			if p.action == "enable" || p.action == "disable" {
				return current.SetEnabled(p.profile, p.name, p.enabled)
			}
			cfg := current.Profiles[p.profile]
			cfg.Mode = p.action
			current.Profiles[p.profile] = cfg
			return nil
		})
		if err != nil {
			return Result{}, err
		}
		d, err = a.s.Load()
		if err != nil {
			return Result{}, err
		}
	}
	targets := p.targets
	if p.profile != "" {
		targets = []string{p.profile}
	}
	reports, err := (syncer.Syncer{Deck: d, ConfigPath: a.s.Path}).SyncTargets(targets)
	rows := []instructions.Result{}
	for _, r := range reports {
		rows = append(rows, instructions.Result{Agent: r.Profile, Path: r.TargetPath, Status: r.Status})
	}
	if err != nil {
		return Result{Message: "Settings saved, but synchronization is incomplete. Review agent status and retry failed targets.", Results: rows, Partial: true}, nil
	}
	return Result{Message: "Agent settings updated. Restart the affected MCP connections.", Results: rows}, nil
}
