package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"github.com/altanmehmet/mcpdeck/internal/syncer"
)

func syncSelection(s store.Store, profile string) error {
	d, err := s.Load()
	if err != nil {
		return err
	}
	sy := syncer.Syncer{Deck: d, ConfigPath: s.Path}
	if profile != "" {
		return sy.Sync(profile)
	}
	return sy.SyncAll()
}

func activationCommand(get func() store.Store, enabled bool) *cobra.Command {
	action := "disable"
	if enabled {
		action = "enable"
	}
	var profile string
	c := &cobra.Command{
		Use:   action + " <server>",
		Short: action + " a server in all configured profiles and sync their settings",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			s := get()
			if err := s.Update(func(d *model.Deck) error {
				return d.SetEnabled(profile, args[0], enabled)
			}); err != nil {
				return err
			}
			if err := syncSelection(s, profile); err != nil {
				return fmt.Errorf("selection saved; sync incomplete (retry with mcpdeck sync): %w", err)
			}
			fmt.Fprintln(c.OutOrStdout(), "Selection saved and synced. Restart the affected agent MCP connections to apply.")
			return nil
		},
	}
	c.Flags().StringVar(&profile, "profile", "", "Apply to one profile only (default: all configured profiles)")
	return c
}
