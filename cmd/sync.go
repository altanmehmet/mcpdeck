package cmd

import (
	"fmt"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"github.com/altanmehmet/mcpdeck/internal/syncer"
	"github.com/spf13/cobra"
	"sort"
)

func syncCommand(get func() store.Store) *cobra.Command {
	var profile, mode string
	var retryFailed bool
	c := &cobra.Command{Use: "sync", Short: "Sync enabled servers to IDE configuration", Args: cobra.NoArgs, RunE: func(c *cobra.Command, args []string) error {
		s := get()
		d, err := s.Load()
		if err != nil {
			return err
		}
		if retryFailed && (profile != "" || mode != "") {
			return fmt.Errorf("--retry-failed cannot be combined with --profile or --mode")
		}
		if profile != "" {
			if _, ok := d.Profiles[profile]; !ok {
				return fmt.Errorf("unknown profile %q", profile)
			}
		}
		if mode != "" {
			if mode != "direct" && mode != "bridge" {
				return fmt.Errorf("mode must be direct or bridge")
			}
			err = s.Update(func(current *model.Deck) error {
				for k, p := range current.Profiles {
					if profile == "" || k == profile {
						p.Mode = mode
						current.Profiles[k] = p
					}
				}
				return nil
			})
			if err != nil {
				return err
			}
			d, err = s.Load()
			if err != nil {
				return err
			}
		}
		sy := syncer.Syncer{Deck: d, ConfigPath: s.Path}
		var results []syncer.SyncResult
		if retryFailed {
			results, err = sy.RetryFailed()
		} else {
			keys := []string{profile}
			if profile == "" {
				keys = nil
				for key := range d.Profiles {
					keys = append(keys, key)
				}
				sort.Strings(keys)
			}
			results, err = sy.SyncTargets(keys)
		}
		for _, result := range results {
			fmt.Fprintf(c.OutOrStdout(), "%s: %s\n", result.Profile, result.Status)
		}
		if err != nil {
			return fmt.Errorf("%w; inspect mcpdeck sync status, then retry mcpdeck sync --retry-failed", err)
		}
		if retryFailed && len(results) == 0 {
			fmt.Fprintln(c.OutOrStdout(), "No failed targets to retry")
		}
		fmt.Fprintln(c.OutOrStdout(), "Sync complete")
		return nil
	}}
	c.Flags().StringVar(&profile, "profile", "", "Sync only this profile")
	c.Flags().StringVar(&mode, "mode", "", "Persist direct or bridge mode")
	c.Flags().BoolVar(&retryFailed, "retry-failed", false, "Retry only failed targets from the saved sync report")
	c.AddCommand(syncStatusCommand(get), syncRestoreCommand(get))
	return c
}
func cacheCommand(get func() store.Store) *cobra.Command {
	return &cobra.Command{Use: "cache-clear [server]", Short: "Invalidate cached tool definitions", Args: cobra.MaximumNArgs(1), RunE: func(c *cobra.Command, args []string) error {
		return get().Update(func(d *model.Deck) error {
			if len(args) > 0 {
				if _, ok := d.Servers[args[0]]; !ok {
					return fmt.Errorf("unknown server %q", args[0])
				}
			}
			for name, s := range d.Servers {
				if len(args) == 0 || args[0] == name {
					s.CachedTools = nil
					d.Servers[name] = s
				}
			}
			return nil
		})
	}}
}
