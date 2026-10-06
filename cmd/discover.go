package cmd

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"github.com/altanmehmet/mcpdeck/internal/syncer"
)

func discoverCommand(get func() store.Store) *cobra.Command {
	return &cobra.Command{Use: "discover", Short: "List MCPs added directly to configured agents without exposing secrets", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		d, err := get().Load()
		if err != nil {
			return err
		}
		found, scanErr := syncer.Discover(d)
		items := make([]syncer.DiscoveredServer, 0, len(found))
		for _, item := range found {
			items = append(items, item)
		}
		sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
		if err := json.NewEncoder(c.OutOrStdout()).Encode(items); err != nil {
			return err
		}
		if scanErr != nil {
			return fmt.Errorf("discovery incomplete: %w", scanErr)
		}
		return nil
	}}
}
