package syncer

import (
	"fmt"
	"os"
)

// CheckInstallTarget rejects malformed files and unmanaged same-name servers
// before executing installation steps or taking ownership of a connection.
func (s Syncer) CheckInstallTarget(key, name string) error {
	p, ok := s.Deck.Profiles[key]
	if !ok {
		return fmt.Errorf("unknown profile")
	}
	raw, err := os.ReadFile(ExpandPath(p.TargetPath))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("cannot read target %s", key)
	}
	root, err := decodeConfig(raw, p.Format)
	if err != nil {
		return fmt.Errorf("invalid target configuration: %s", key)
	}
	servers, err := profileServers(root, p.Format, false)
	if err != nil {
		return fmt.Errorf("invalid server map in %s", key)
	}
	if servers != nil {
		if _, exists := servers[name]; exists {
			return fmt.Errorf("%s already has a server named %s; refusing overwrite", key, name)
		}
	}
	return nil
}
