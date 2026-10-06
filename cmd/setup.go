package cmd

import (
	"encoding/json"
	"github.com/spf13/cobra"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"github.com/altanmehmet/mcpdeck/internal/syncer"
	"os"
	"os/exec"
	"sort"
)

// Setup checks only installed executables and configuration files. It does not
// start servers, install software, test accounts or expose connection secrets.
func setupCommand(get func() store.Store) *cobra.Command {
	return &cobra.Command{Use: "setup", Short: "Report desktop setup requirements as JSON", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		d, err := get().Load()
		if err != nil {
			return err
		}
		type runtime struct {
			Name      string `json:"name"`
			Available bool   `json:"available"`
			Required  bool   `json:"required"`
		}
		type profile struct {
			Name     string `json:"name"`
			Exists   bool   `json:"exists"`
			Detected bool   `json:"detected"`
		}
		result := struct {
			Runtimes []runtime `json:"runtimes"`
			Profiles []profile `json:"profiles"`
		}{Runtimes: []runtime{}, Profiles: []profile{}}
		required := map[string]bool{"npx": false, "uvx": false, "docker": false}
		for name, p := range d.Profiles {
			info, e := os.Stat(syncer.ExpandPath(p.TargetPath))
			result.Profiles = append(result.Profiles, profile{name, e == nil && !info.IsDir(), store.AgentDetected(name, p)})
			for _, server := range p.EnabledServers {
				cfg := d.Servers[server]
				if cfg.URL != "" && p.Mode != "bridge" && !syncer.UsesRemoteAdapter(cfg, p) {
					continue
				}
				cfg = model.Stdio(cfg)
				required[cfg.Command] = true
			}
		}
		for command, needed := range required {
			_, e := exec.LookPath(command)
			result.Runtimes = append(result.Runtimes, runtime{command, e == nil, needed})
		}
		sort.Slice(result.Runtimes, func(i, j int) bool { return result.Runtimes[i].Name < result.Runtimes[j].Name })
		sort.Slice(result.Profiles, func(i, j int) bool { return result.Profiles[i].Name < result.Profiles[j].Name })
		return json.NewEncoder(c.OutOrStdout()).Encode(result)
	}}
}
