package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/altanmehmet/mcpdeck/internal/install"
)

func environmentCommand() *cobra.Command {
	var jsonOutput bool
	c := &cobra.Command{Use: "environment", Short: "Inspect installed MCP runtimes without reading credentials", Args: cobra.NoArgs}
	c.RunE = func(c *cobra.Command, _ []string) error {
		inventory := install.InspectEnvironment(c.Context())
		if jsonOutput {
			return json.NewEncoder(c.OutOrStdout()).Encode(inventory)
		}
		fmt.Fprintf(c.OutOrStdout(), "Local environment: %s/%s\nDetected software: %s\n", inventory.OS, inventory.Architecture, inventory.Summary())
		for _, tool := range inventory.Tools {
			if !tool.Available {
				fmt.Fprintf(c.OutOrStdout(), "%s: not found in supported locations\n", tool.Name)
				continue
			}
			fmt.Fprintf(c.OutOrStdout(), "%s: %s (version probe passed: %t)\n  %s\n", tool.Name, safeText(tool.Path), tool.Usable, safeText(tool.Version))
		}
		for _, java := range inventory.Java {
			fmt.Fprintf(c.OutOrStdout(), "JAVA_HOME: %s (version probe passed: %t)\n  %s\n", safeText(java.Home), java.Usable, safeText(java.Version))
		}
		fmt.Fprintf(c.OutOrStdout(), "Oracle connection-store directory exists: %t. Saved connections and database access have not been inspected.\n", inventory.OracleConnectionStore)
		return nil
	}
	c.Flags().BoolVar(&jsonOutput, "json", false, "Print the software inventory as JSON")
	return c
}
