package cmd

import (
	"fmt"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"github.com/altanmehmet/mcpdeck/internal/syncer"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
)

func doctorCommand(get func() store.Store) *cobra.Command {
	return &cobra.Command{Use: "doctor", Short: "Check executables, environment and IDE config paths", Args: cobra.NoArgs, RunE: func(c *cobra.Command, args []string) error {
		failed := false
		check := func(label string, err error) {
			if err != nil {
				fmt.Fprintf(c.OutOrStdout(), "[✗] %s: %v\n", label, err)
				failed = true
			} else {
				fmt.Fprintf(c.OutOrStdout(), "[✓] %s\n", label)
			}
		}
		for _, name := range []string{"npx", "uvx", "docker", "go"} {
			_, err := exec.LookPath(name)
			if err != nil {
				fmt.Fprintf(c.OutOrStdout(), "[i] %s not found (optional unless required by an enabled MCP; Go is only needed for source builds)\n", name)
			} else {
				check(name+" available", nil)
			}
		}
		d, err := get().Load()
		check("deck syntax", err)
		if err != nil {
			return fmt.Errorf("doctor found configuration errors")
		}
		enabled := map[string]bool{}
		for _, p := range d.Profiles {
			for _, s := range p.EnabledServers {
				enabled[s] = true
			}
		}
		names := []string{}
		for name := range d.Servers {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			cfg := d.Servers[name]
			_, err := model.Resolve(cfg)
			check(name+" environment", err)
			if enabled[name] {
				if cfg.URL != "" {
					needsAdapter := false
					for key, p := range d.Profiles {
						if d.IsEnabled(key, name) && (p.Mode == "bridge" || syncer.UsesRemoteAdapter(cfg, p)) {
							needsAdapter = true
						}
					}
					if !needsAdapter {
						continue
					}
				}
				cfg = model.Stdio(cfg)
				_, err = exec.LookPath(cfg.Command)
				check(name+" executable", err)
			}
		}
		profiles := []string{}
		for name := range d.Profiles {
			profiles = append(profiles, name)
		}
		sort.Strings(profiles)
		for _, name := range profiles {
			check(name+" config readable/writable", checkProfilePath(syncer.ExpandPath(d.Profiles[name].TargetPath), d.Profiles[name].Format))
		}
		if failed {
			return fmt.Errorf("doctor found issues")
		}
		return nil
	}}
}
func checkPath(path string) error { return checkProfilePath(path, "") }
func checkProfilePath(path, format string) error {
	if b, err := os.ReadFile(path); err == nil {
		if err := syncer.ValidateConfig(b, format); err != nil {
			return err
		}
		f, err := os.OpenFile(path, os.O_WRONLY, 0)
		if err != nil {
			return err
		}
		f.Close()
	} else if !os.IsNotExist(err) {
		return err
	}
	dir := filepath.Dir(path)
	for {
		info, err := os.Stat(dir)
		if err == nil {
			if !info.IsDir() {
				return fmt.Errorf("parent is not a directory")
			}
			break
		}
		if !os.IsNotExist(err) {
			return err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return err
		}
		dir = parent
	}
	f, err := os.CreateTemp(dir, ".mcpdeck-doctor-*")
	if err != nil {
		return err
	}
	name := f.Name()
	if err = f.Close(); err != nil {
		os.Remove(name)
		return err
	}
	return os.Remove(name)
}
