package syncer

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
)

// DiscoveredServer exposes names and locations only, never connection settings.
// Different definitions with the same name remain in their original profiles.
type DiscoveredServer struct {
	Name         string   `json:"name"`
	Profiles     []string `json:"profiles"`
	fingerprints map[string][32]byte
	paths        map[string]string
}

func readProfile(p model.ProfileConfig) ([]byte, map[string]any, map[string]any, error) {
	f, err := os.Open(ExpandPath(p.TargetPath))
	if os.IsNotExist(err) {
		return nil, nil, nil, nil
	}
	if err != nil {
		return nil, nil, nil, fmt.Errorf("cannot read agent configuration")
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, 4*1024*1024+1))
	if err != nil || len(raw) > 4*1024*1024 {
		return nil, nil, nil, fmt.Errorf("agent configuration unreadable or exceeds 4 MiB")
	}
	root, err := decodeConfig(raw, p.Format)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid agent configuration")
	}
	servers, err := profileServers(root, p.Format, false)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid MCP section")
	}
	return raw, root, servers, nil
}

func entryFingerprint(entry any) [32]byte {
	raw, _ := json.Marshal(entry)
	return sha256.Sum256(raw)
}

// Discover scans configured adapters without importing, resolving or running MCPs.
func Discover(d *model.Deck) (map[string]DiscoveredServer, error) {
	result := map[string]DiscoveredServer{}
	var errs []error
	keys := make([]string, 0, len(d.Profiles))
	for key := range d.Profiles {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		_, _, servers, err := readProfile(d.Profiles[key])
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", key, err))
			continue
		}
		for name, entry := range servers {
			if name == "mcpdeck" {
				continue
			}
			if _, managed := d.Servers[name]; managed {
				continue
			}
			item, ok := result[name]
			if !ok {
				item = DiscoveredServer{Name: name, fingerprints: map[string][32]byte{}, paths: map[string]string{}}
			}
			item.Profiles = append(item.Profiles, key)
			item.fingerprints[key] = entryFingerprint(entry)
			item.paths[key] = d.Profiles[key].TargetPath
			result[name] = item
		}
	}
	return result, errors.Join(errs...)
}

// RemoveDiscovered deletes only the reviewed name from its original agents.
// It preserves unknown MCP fields and settings without adopting their semantics.
func RemoveDiscovered(s store.Store, item DiscoveredServer) error {
	d, err := s.Load()
	if err != nil {
		return err
	}
	if _, managed := d.Servers[item.Name]; managed {
		return fmt.Errorf("MCP is now managed by MCPDeck; refresh before removing")
	}
	if len(item.Profiles) == 0 {
		return fmt.Errorf("no discovered MCP locations")
	}
	current, scanErr := Discover(d)
	if scanErr != nil {
		return fmt.Errorf("discovery incomplete; refresh after checking agent settings")
	}
	fresh, exists := current[item.Name]
	if !exists {
		return nil
	}
	if len(fresh.Profiles) != len(item.Profiles) {
		return fmt.Errorf("MCP locations changed; refresh and review again")
	}
	for _, key := range fresh.Profiles {
		if fresh.paths[key] != item.paths[key] || fresh.fingerprints[key] != item.fingerprints[key] {
			return fmt.Errorf("MCP settings changed; refresh and review again")
		}
	}
	var errs []error
	for _, key := range item.Profiles {
		p, ok := d.Profiles[key]
		if !ok {
			errs = append(errs, fmt.Errorf("%s: profile changed; refresh", key))
			continue
		}
		raw, root, servers, err := readProfile(p)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", key, err))
			continue
		}
		entry, exists := servers[item.Name]
		if !exists {
			continue
		} // A previously completed location needs no rewrite.
		expected, reviewed := item.fingerprints[key]
		if !reviewed || entryFingerprint(entry) != expected {
			errs = append(errs, fmt.Errorf("%s: MCP settings changed; refresh and review again", key))
			continue
		}
		delete(servers, item.Name)
		updated, err := encodeConfig(root, p.Format)
		if err == nil {
			err = store.UpdateFile(ExpandPath(p.TargetPath), ".mcpdeck-remove-backup", func(current []byte, exists bool) ([]byte, error) {
				if !exists || !bytes.Equal(raw, current) {
					return nil, fmt.Errorf("agent settings changed during removal; retry")
				}
				return append(updated, '\n'), nil
			})
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: removal failed; retry after checking this agent", key))
		}
	}
	return errors.Join(errs...)
}
