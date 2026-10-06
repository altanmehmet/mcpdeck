package syncer

import (
	"fmt"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Syncer struct {
	Deck       *model.Deck
	ConfigPath string
	Executable string
}

func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		h, _ := os.UserHomeDir()
		return filepath.Join(h, path[2:])
	}
	return path
}
func (s Syncer) SyncBridgeMode(key string) error { return s.sync(key, true) }
func (s Syncer) Sync(key string) error           { return s.sync(key, s.Deck.Profiles[key].Mode == "bridge") }
func (s Syncer) SyncAll() error {
	keys := make([]string, 0, len(s.Deck.Profiles))
	for k := range s.Deck.Profiles {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	_, err := s.SyncTargets(keys)
	return err
}
func (s Syncer) sync(key string, bridge bool) error {
	p, ok := s.Deck.Profiles[key]
	if !ok {
		return fmt.Errorf("unknown profile %q", key)
	}
	path := ExpandPath(p.TargetPath)
	return store.UpdateFile(path, ".mcpdeck-backup", func(b []byte, existed bool) ([]byte, error) {
		return s.render(key, bridge, b)
	})
}

func (s Syncer) render(key string, bridge bool, b []byte) ([]byte, error) {
	p := s.Deck.Profiles[key]
	root, err := decodeConfig(b, p.Format)
	if err != nil {
		return nil, err
	}
	servers, err := profileServers(root, p.Format, true)
	if err != nil {
		return nil, err
	}
	// Deck server names and the reserved mcpdeck entry are managed; unrelated entries survive.
	for name := range s.Deck.Servers {
		delete(servers, name)
	}
	delete(servers, "mcpdeck")
	if bridge {
		exe := s.Executable
		if exe == "" {
			exe, err = os.Executable()
			if err != nil {
				return nil, err
			}
		}
		args := []string{"bridge", "--profile", key, "--idle-timeout", "3m"}
		if s.ConfigPath != "" {
			args = append(args, "--config", s.ConfigPath)
		}
		servers["mcpdeck"] = serverEntry(model.ServerConfig{Command: exe, Args: args}, p.Format)
	} else {
		for _, name := range p.EnabledServers {
			cfg, ok := s.Deck.Servers[name]
			if !ok {
				return nil, fmt.Errorf("unknown server %s", name)
			}
			cfg, err = model.Resolve(cfg)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			servers[name] = profileServerEntry(cfg, p)
		}
	}
	if err != nil {
		return nil, err
	}
	if p.Format != "opencode" {
		root[serverField(p.Format)] = servers
	} else if mcp, ok := root["mcp"].(map[string]any); ok {
		if _, nested := mcp["servers"]; nested {
			mcp["servers"] = servers
		}
	}
	b, err = encodeConfig(root, p.Format)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}
