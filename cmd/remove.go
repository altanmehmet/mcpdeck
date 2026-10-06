package cmd

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"github.com/altanmehmet/mcpdeck/internal/syncer"
)

func removeCommand(get func() store.Store) *cobra.Command {
	var yes bool
	c := &cobra.Command{Use: "remove <server>", Aliases: []string{"uninstall"}, Short: "Remove an MCP from MCPDeck and every configured agent", Args: cobra.ExactArgs(1)}
	c.RunE = func(c *cobra.Command, args []string) error {
		s := get()
		d, err := s.Load()
		if err != nil {
			return err
		}
		_, managed := d.Servers[args[0]]
		var external syncer.DiscoveredServer
		if !managed {
			found, scanErr := syncer.Discover(d)
			if scanErr != nil {
				return fmt.Errorf("cannot fully inspect agents: %w", scanErr)
			}
			var exists bool
			external, exists = found[args[0]]
			if !exists {
				return fmt.Errorf("unknown MCP %s", safeText(args[0]))
			}
			fmt.Fprintln(c.OutOrStdout(), "Found directly in agents:", safeText(strings.Join(external.Profiles, ", ")))
		}
		if !yes {
			fmt.Fprintf(c.OutOrStdout(), "Remove %s from MCPDeck and all %d configured agents? Installed files will be kept.\n", safeText(args[0]), len(d.Profiles))
			if !confirmYes(bufio.NewReader(c.InOrStdin()), c.OutOrStdout(), "Remove MCP?") {
				return fmt.Errorf("removal cancelled")
			}
		}
		if managed {
			err = syncer.Remove(s, args[0])
		} else {
			err = syncer.RemoveDiscovered(s, external)
		}
		if err != nil {
			return err
		}
		fmt.Fprintln(c.OutOrStdout(), "MCP removed from MCPDeck and all configured agents. Restart or reload their MCP connections. Installed files were kept.")
		return nil
	}
	c.Flags().BoolVarP(&yes, "yes", "y", false, "Confirm removal without prompting")
	return c
}
