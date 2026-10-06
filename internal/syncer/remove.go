package syncer

import (
	"fmt"

	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
)

// Remove keeps the managed name until every profile has been synced. On partial
// failure the disabled record remains, allowing removal to be retried safely.
// Installed programs and shared runtimes are deliberately retained.
func Remove(s store.Store, name string) error {
	if err := s.Update(func(d *model.Deck) error {
		if _, ok := d.Servers[name]; !ok {
			return fmt.Errorf("unknown server %q", name)
		}
		for key, p := range d.Profiles {
			kept := make([]string, 0, len(p.EnabledServers))
			for _, server := range p.EnabledServers {
				if server != name {
					kept = append(kept, server)
				}
			}
			p.EnabledServers = kept
			d.Profiles[key] = p
		}
		return nil
	}); err != nil {
		return err
	}
	d, err := s.Load()
	if err != nil {
		return err
	}
	if err = (Syncer{Deck: d, ConfigPath: s.Path}).SyncAll(); err != nil {
		return fmt.Errorf("MCP disabled; removal incomplete (retry mcpdeck remove %s): %w", name, err)
	}
	return s.Update(func(d *model.Deck) error {
		// A concurrent activation must not be silently discarded.
		for key := range d.Profiles {
			if d.IsEnabled(key, name) {
				return fmt.Errorf("MCP was enabled during removal; retry removal")
			}
		}
		delete(d.Servers, name)
		return nil
	})
}
