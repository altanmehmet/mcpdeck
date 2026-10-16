package vault

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

func readSecret(id string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, "/usr/bin/security", "find-generic-password", "-s", "com.mcpdeck.vault", "-a", id, "-w")
	c.Stderr = io.Discard
	raw, err := c.Output()
	if err != nil {
		return nil, fmt.Errorf("Keychain credential unavailable")
	}
	raw = bytes.TrimSuffix(raw, []byte("\n"))
	if !bytes.HasPrefix(raw, []byte("mcpdeck-v1:")) {
		return nil, fmt.Errorf("invalid Keychain credential format")
	}
	return base64.RawStdEncoding.DecodeString(string(bytes.TrimPrefix(raw, []byte("mcpdeck-v1:"))))
}
func writeSecret(id string, value []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// Interactive input avoids putting even encoded secret data in process argv.
	// Base64 data and validated IDs cannot inject commands into security's parser.
	// The password stays in stdin, never argv; encoding also fits security's line limit.
	c := exec.CommandContext(ctx, "/usr/bin/security", "-i")
	c.Stdin = strings.NewReader("add-generic-password -U -s com.mcpdeck.vault -a " + id + " -w mcpdeck-v1:" + base64.RawStdEncoding.EncodeToString(value) + "\n")
	c.Stdout = io.Discard
	c.Stderr = io.Discard
	if c.Run() != nil {
		return fmt.Errorf("cannot save credential in Keychain")
	}
	// security interactive mode can exit successfully after a failed command.
	saved, err := readSecret(id)
	if err != nil || !bytes.Equal(saved, value) {
		_ = deleteSecret(id)
		return fmt.Errorf("Keychain did not save the credential; no plaintext fallback")
	}
	return nil
}

func deleteSecret(id string) error {
	c := exec.Command("/usr/bin/security", "delete-generic-password", "-s", "com.mcpdeck.vault", "-a", id)
	c.Stdout = io.Discard
	c.Stderr = io.Discard
	return c.Run()
}
