// Package vault stores credentials in the current user's OS credential store.
// References are not credentials and can safely be persisted in deck files.
package vault

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
)

const Prefix = "vault://"
const MaxSecret = 2560 // Windows CRED_MAX_CREDENTIAL_BLOB_SIZE.
var identifier = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,160}$`)

func ID(config, server, variable string) string {
	sum := sha256.Sum256([]byte(config + "\x00" + server + "\x00" + variable))
	return fmt.Sprintf("%x", sum[:])
}

// NewID avoids overwriting the previous credential before a repair is verified.
func NewID(config, server, variable string) string { return ID(config, server, variable+rand.Text()) }
func Reference(id string) string                   { return Prefix + id }
func IsReference(value string) bool                { return strings.HasPrefix(value, Prefix) }
func Get(reference string) (string, error) {
	id := strings.TrimPrefix(reference, Prefix)
	if !IsReference(reference) || !identifier.MatchString(id) {
		return "", fmt.Errorf("invalid vault reference")
	}
	value, err := readSecret(id)
	if err != nil || len(value) == 0 || len(value) > MaxSecret {
		return "", fmt.Errorf("vault credential unavailable; unlock the OS credential store or re-enter it with mcpdeck vault bind")
	}
	return string(value), nil
}
func Set(id string, value []byte) error {
	if !identifier.MatchString(id) || len(value) == 0 || len(value) > MaxSecret || strings.ContainsRune(string(value), '\x00') {
		return fmt.Errorf("invalid vault key or secret (maximum 2560 bytes)")
	}
	return writeSecret(id, value)
}

// Delete removes only the referenced OS credential; callers retain a failing reference.
func Delete(reference string) error {
	id := strings.TrimPrefix(reference, Prefix)
	if !IsReference(reference) || !identifier.MatchString(id) {
		return fmt.Errorf("invalid vault reference")
	}
	return deleteSecret(id)
}
