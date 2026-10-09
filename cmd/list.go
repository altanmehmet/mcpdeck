package cmd

import (
	"fmt"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"github.com/altanmehmet/mcpdeck/internal/syncer"
	"github.com/spf13/cobra"
	"sort"
	"strings"
	"text/tabwriter"
)

// list is the human-readable inventory. status keeps its existing JSON contract.
func listCommand(get func() store.Store) *cobra.Command {
	var profile string
	var jsonOutput bool
	c := &cobra.Command{Use: "list", Aliases: []string{"ls"}, Short: "List MCP servers and the agents using them", Args: cobra.NoArgs}
	c.RunE = func(c *cobra.Command, args []string) error {
		if jsonOutput {
			if profile != "" {
				return fmt.Errorf("--json cannot be combined with --profile; use status for the complete JSON overview")
			}
			return statusCommand(get).RunE(c, args)
		}
		d, err := get().Load()
		if err != nil {
			return err
		}
		if profile != "" {
			if _, ok := d.Profiles[profile]; !ok {
				return fmt.Errorf("unknown profile %q; run mcpdeck agents", profile)
			}
		}
		type row struct {
			name, kind, source string
			profiles           []string
		}
		rows := []row{}
		for name, cfg := range d.Servers {
			kind := "local"
			if cfg.URL != "" {
				kind = "remote"
			}
			r := row{name: name, kind: kind, source: "managed"}
			for key := range d.Profiles {
				if d.IsEnabled(key, name) {
					r.profiles = append(r.profiles, key)
				}
			}
			sort.Strings(r.profiles)
			rows = append(rows, r)
		}
		external, discoverErr := syncer.Discover(d)
		for name, item := range external {
			rows = append(rows, row{name: name, kind: "agent config", source: "external", profiles: item.Profiles})
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].name < rows[j].name })
		if len(rows) == 0 {
			fmt.Fprintln(c.OutOrStdout(), "No MCP servers. Run mcpdeck install or open mcpdeck to get started.")
		}
		if len(rows) > 0 {
			w := tabwriter.NewWriter(c.OutOrStdout(), 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "SERVER\tTYPE\tSOURCE\tENABLED FOR")
			for _, r := range rows {
				enabled := strings.Join(r.profiles, ", ")
				if enabled == "" {
					enabled = "disabled"
				}
				if profile != "" {
					enabled = "disabled"
					for _, p := range r.profiles {
						if p == profile {
							enabled = profile
						}
					}
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", safeText(r.name), r.kind, r.source, safeText(enabled))
			}
			if err = w.Flush(); err != nil {
				return err
			}
			fmt.Fprintln(c.OutOrStdout(), "\nEnabled means configured. Restart agent connections after changes.")
		}
		if discoverErr != nil {
			fmt.Fprintln(c.ErrOrStderr(), "Some agent files could not be read; external inventory may be incomplete.")
		}
		return nil
	}
	c.Flags().StringVar(&profile, "profile", "", "Show enabled state for one agent")
	c.Flags().BoolVar(&jsonOutput, "json", false, "Print the existing status JSON format (managed servers only)")
	c.Example = "  mcpdeck list\n  mcpdeck list --profile cursor\n  mcpdeck list --json"
	return c
}
