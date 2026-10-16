//go:build !darwin && !windows

package vault

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

func secretTool(args []string, input string) ([]byte, error) {
	path, err := exec.LookPath("secret-tool")
	if err != nil {
		return nil, fmt.Errorf("vault requires an unlocked Secret Service and secret-tool; no plaintext fallback")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, path, args...)
	c.Stdin = strings.NewReader(input)
	c.Stderr = io.Discard
	raw, err := c.Output()
	if err != nil {
		return nil, fmt.Errorf("Secret Service unavailable; unlock your keyring")
	}
	return raw, nil
}
func readSecret(id string) ([]byte, error) {
	raw, err := secretTool([]string{"lookup", "application", "mcpdeck", "key", id}, "")
	return []byte(strings.TrimSuffix(string(raw), "\n")), err
}
func writeSecret(id string, value []byte) error {
	_, err := secretTool([]string{"store", "--label=MCPDeck credential", "application", "mcpdeck", "key", id}, string(value))
	if err != nil {
		return err
	}
	saved, err := readSecret(id)
	if err != nil || !bytes.Equal(saved, value) {
		_ = deleteSecret(id)
		return fmt.Errorf("Secret Service did not save the exact credential; no plaintext fallback")
	}
	return nil
}

func deleteSecret(id string) error {
	_, err := secretTool([]string{"clear", "application", "mcpdeck", "key", id}, "")
	return err
}
