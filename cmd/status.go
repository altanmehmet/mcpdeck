package cmd

import (
	"encoding/json"
	"github.com/spf13/cobra"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"sort"
)

// status intentionally does not expose URLs, arguments, environment or headers:
// any of these may contain credentials. It describes selections, not connectivity.
func statusCommand(get func() store.Store) *cobra.Command {
	return &cobra.Command{Use: "status", Short: "Print a secret-free JSON overview", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		d, err := get().Load()
		if err != nil {
			return err
		}
		type server struct {
			Name    string   `json:"name"`
			Kind    string   `json:"kind"`
			Enabled []string `json:"enabled_profiles"`
		}
		type profile struct {
			Name string `json:"name"`
			Mode string `json:"mode"`
			Path string `json:"path"`
		}
		result := struct {
			Servers  []server  `json:"servers"`
			Profiles []profile `json:"profiles"`
		}{Servers: []server{}, Profiles: []profile{}}
		for name, cfg := range d.Servers {
			item := server{Name: name, Kind: "local", Enabled: []string{}}
			if cfg.URL != "" {
				item.Kind = "remote"
			}
			for p := range d.Profiles {
				if d.IsEnabled(p, name) {
					item.Enabled = append(item.Enabled, p)
				}
			}
			sort.Strings(item.Enabled)
			result.Servers = append(result.Servers, item)
		}
		for name, p := range d.Profiles {
			mode := p.Mode
			if mode == "" {
				mode = "direct"
			}
			result.Profiles = append(result.Profiles, profile{name, mode, p.TargetPath})
		}
		sort.Slice(result.Servers, func(i, j int) bool { return result.Servers[i].Name < result.Servers[j].Name })
		sort.Slice(result.Profiles, func(i, j int) bool { return result.Profiles[i].Name < result.Profiles[j].Name })
		return json.NewEncoder(c.OutOrStdout()).Encode(result)
	}}
}
