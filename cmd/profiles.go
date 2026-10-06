package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"sort"
)

func profilesCommand(get func() store.Store) *cobra.Command {
	c := &cobra.Command{Use: "profiles", Short: "List configured application profiles", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		d, err := get().Load()
		if err != nil {
			return err
		}
		keys := []string{}
		for k := range d.Profiles {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			p := d.Profiles[k]
			format := p.Format
			if format == "" {
				format = "mcpServers"
			}
			mode := p.Mode
			if mode == "" {
				mode = "direct"
			}
			fmt.Fprintf(c.OutOrStdout(), "%s\t%s\t%s\t%s\n", k, mode, format, p.TargetPath)
		}
		return nil
	}}
	c.AddCommand(&cobra.Command{Use: "add-defaults", Short: "Add missing built-in profiles without modifying existing profiles or IDE files", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		count := 0
		err := get().Update(func(d *model.Deck) error {
			for k, p := range store.DefaultProfiles() {
				if _, exists := d.Profiles[k]; !exists {
					d.Profiles[k] = p
					count++
				}
			}
			return nil
		})
		if err == nil {
			fmt.Fprintf(c.OutOrStdout(), "Added %d profiles. Enable servers in the panel, then sync the desired profile.\n", count)
		}
		return err
	}})
	return c
}
