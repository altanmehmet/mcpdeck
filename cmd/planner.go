package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/altanmehmet/mcpdeck/internal/install"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"os"
	"path/filepath"
)

func plannerSettingsPath(s store.Store) string {
	return filepath.Join(filepath.Dir(s.Path), "planner.json")
}
func loadPlanner(s store.Store) (install.PlannerOptions, error) {
	o := install.PlannerOptions{Provider: "codex"}
	raw, err := os.ReadFile(plannerSettingsPath(s))
	if os.IsNotExist(err) {
		return o, nil
	}
	if err != nil {
		return o, err
	}
	if len(raw) > 64*1024 || json.Unmarshal(raw, &o) != nil {
		return o, fmt.Errorf("invalid planner settings")
	}
	return o, o.Validate()
}
func plannerCommand(get func() store.Store) *cobra.Command {
	root := &cobra.Command{Use: "planner", Short: "Show or select the installation planner", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		o, err := loadPlanner(get())
		if err != nil {
			return err
		}
		return json.NewEncoder(c.OutOrStdout()).Encode(o)
	}}
	var o install.PlannerOptions
	set := &cobra.Command{Use: "set", Short: "Save planner selection (API keys are never saved)", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		if err := o.Validate(); err != nil {
			return err
		}
		raw, _ := json.MarshalIndent(o, "", "  ")
		if err := store.AtomicWrite(plannerSettingsPath(get()), append(raw, '\n')); err != nil {
			return err
		}
		fmt.Fprintln(c.OutOrStdout(), "Planlayici secimi kaydedildi. API anahtari kaydedilmedi.")
		return nil
	}}
	set.Flags().StringVar(&o.Provider, "provider", "codex", "codex, claude, gemini, openai-api, anthropic-api, gemini-api, openai-compatible")
	set.Flags().StringVar(&o.Model, "model", "", "Model identifier; required for API providers")
	set.Flags().StringVar(&o.Executable, "executable", "", "Optional CLI executable path")
	set.Flags().StringVar(&o.BaseURL, "base-url", "", "OpenAI-compatible API base URL, including /v1 when required")
	set.Flags().StringVar(&o.KeyEnv, "key-env", "", "Environment variable containing the API key (name only)")
	root.AddCommand(set)
	root.AddCommand(&cobra.Command{Use: "list", Short: "List available planner adapters", Args: cobra.NoArgs, Run: func(c *cobra.Command, _ []string) {
		for _, name := range install.Providers {
			fmt.Fprintln(c.OutOrStdout(), name)
		}
	}})
	return root
}
