package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"io"
	"strings"
)

func importCommand(get func() store.Store) *cobra.Command {
	var name, profile string
	var preview, disabled, jsonInput bool
	c := &cobra.Command{Use: "import", Short: "Detect MCP connections from URL, JSON or TOML on stdin and distribute them", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		raw, err := io.ReadAll(io.LimitReader(c.InOrStdin(), 1024*1024+1))
		if err != nil {
			return fmt.Errorf("cannot read MCP settings")
		}
		if len(raw) > 1024*1024 {
			return fmt.Errorf("MCP settings exceed 1 MiB")
		}
		source := string(raw)
		values := map[string]string{}
		if jsonInput {
			var input struct {
				Source string            `json:"source"`
				Name   string            `json:"name"`
				Values map[string]string `json:"values"`
			}
			decoder := json.NewDecoder(strings.NewReader(source))
			decoder.DisallowUnknownFields()
			if decoder.Decode(&input) != nil {
				return fmt.Errorf("invalid import request")
			}
			var extra any
			if decoder.Decode(&extra) != io.EOF {
				return fmt.Errorf("expected one import request")
			}
			source = input.Source
			values = input.Values
			if name == "" {
				name = input.Name
			}
		}
		imported, err := model.Import(source, name, values)
		if err != nil {
			return err
		}
		if preview {
			return json.NewEncoder(c.OutOrStdout()).Encode(imported)
		}
		if disabled && profile != "" {
			return fmt.Errorf("--disabled cannot be combined with a profile selection")
		}
		for _, item := range imported {
			if !disabled && !item.Disabled {
				if len(item.Required) > 0 {
					return fmt.Errorf("%s requires values for: %s", item.Name, strings.Join(item.Required, ", "))
				}
				if _, err := model.Resolve(item.Server); err != nil {
					return fmt.Errorf("%s: %w", item.Name, err)
				}
			}
		}
		if err = get().Update(func(d *model.Deck) error {
			if profile != "" {
				if _, ok := d.Profiles[profile]; !ok {
					return fmt.Errorf("unknown profile %s", profile)
				}
			}
			for _, item := range imported {
				if _, exists := d.Servers[item.Name]; exists {
					return fmt.Errorf("server %s already exists; no connections imported", item.Name)
				}
			}
			for _, item := range imported {
				d.Servers[item.Name] = item.Server
				if !disabled && !item.Disabled {
					if err := d.SetEnabled(profile, item.Name, true); err != nil {
						return err
					}
				}
			}
			return nil
		}); err != nil {
			return err
		}
		if !disabled {
			if err := syncSelection(get(), profile); err != nil {
				return fmt.Errorf("connections saved; sync incomplete (retry with mcpdeck sync): %w", err)
			}
		}
		fmt.Fprintf(c.OutOrStdout(), "Imported %d MCP connections.\n", len(imported))
		return nil
	}}
	c.Flags().StringVar(&name, "name", "", "Name for a bare connection or URL")
	c.Flags().StringVar(&profile, "profile", "", "Apply to one profile (default: all configured profiles)")
	c.Flags().BoolVar(&preview, "preview", false, "Describe detected connections and required inputs without saving or exposing secrets")
	c.Flags().BoolVar(&disabled, "disabled", false, "Save all connections without enabling or syncing")
	c.Flags().BoolVar(&jsonInput, "json-stdin", false, "Read {source,name,values} from stdin")
	return c
}
