package cmd

import (
	"fmt"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"github.com/spf13/cobra"
	"sort"
	"text/tabwriter"
)

func agentsCommand(get func() store.Store) *cobra.Command {
	return &cobra.Command{Use: "agents", Short: "List agents, modes and enabled MCP counts", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		d, err := get().Load()
		if err != nil {
			return err
		}
		names := []string{}
		for name := range d.Profiles {
			names = append(names, name)
		}
		sort.Strings(names)
		if len(names) == 0 {
			fmt.Fprintln(c.OutOrStdout(), "No agents configured. Run mcpdeck profiles add-defaults.")
			return nil
		}
		w := tabwriter.NewWriter(c.OutOrStdout(), 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "AGENT\tMODE\tMCPS\tCONFIGURATION")
		for _, name := range names {
			p := d.Profiles[name]
			mode := p.Mode
			if mode == "" {
				mode = "direct"
			}
			state := "not detected"
			if store.AgentDetected(name, p) {
				state = "detected"
			}
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", safeText(name), mode, len(p.EnabledServers), state)
		}
		if err = w.Flush(); err != nil {
			return err
		}
		fmt.Fprintln(c.OutOrStdout(), "\nUse --profile <agent> to target one agent. Detection does not test connectivity.\nRun mcpdeck profiles to see configuration paths.")
		return nil
	}}
}
