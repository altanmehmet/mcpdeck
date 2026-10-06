package cmd

import (
	"fmt"

	"github.com/altanmehmet/mcpdeck/internal/store"
	"github.com/altanmehmet/mcpdeck/internal/syncer"
	"github.com/spf13/cobra"
)

func syncStatusCommand(get func() store.Store) *cobra.Command {
	return &cobra.Command{Use: "status", Short: "Show the last synchronization outcome for each target", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		s := get()
		results, err := (syncer.Syncer{ConfigPath: s.Path}).Results()
		if err != nil {
			return err
		}
		if len(results) == 0 {
			fmt.Fprintln(c.OutOrStdout(), "No recorded synchronization attempts")
		}
		for _, result := range results {
			fmt.Fprintf(c.OutOrStdout(), "%s: %s (%s)\n", safeText(result.Profile), safeText(result.Status), safeText(result.TargetPath))
		}
		return nil
	}}
}

func syncRestoreCommand(get func() store.Store) *cobra.Command {
	var profile string
	var yes bool
	c := &cobra.Command{Use: "restore", Short: "Restore an agent's last sync backup (deck stays unchanged)", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		s := get()
		d, err := s.Load()
		if err != nil {
			return err
		}
		p, ok := d.Profiles[profile]
		if !ok {
			return fmt.Errorf("choose an existing --profile")
		}
		fmt.Fprintf(c.OutOrStdout(), "Restore %s from its last sync backup: %s\n", safeText(profile), safeText(syncer.ExpandPath(p.TargetPath)))
		if !yes {
			fmt.Fprintln(c.OutOrStdout(), "This replaces the complete agent configuration. Review the backup locally, then repeat with --yes to confirm.")
			return nil
		}
		if err = (syncer.Syncer{Deck: d, ConfigPath: s.Path}).Restore(profile); err != nil {
			return err
		}
		fmt.Fprintln(c.OutOrStdout(), "Backup restored; the previous file is now the backup. Deck selections are unchanged; a later sync reapplies them. Reload the agent to reconnect.")
		return nil
	}}
	c.Flags().StringVar(&profile, "profile", "", "Agent profile to restore")
	c.Flags().BoolVar(&yes, "yes", false, "Confirm replacing the complete target configuration")
	return c
}
