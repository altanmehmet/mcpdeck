package vault

import (
	"bytes"
	"os"
	"testing"
)

func TestInvalidSecretsAndReferences(t *testing.T) {
	for _, id := range []string{"", "bad\ncommand", "../key", "key with space"} {
		if err := Set(id, []byte("test")); err == nil {
			t.Fatal("invalid key accepted")
		}
		if _, err := Get(Reference(id)); err == nil {
			t.Fatal("invalid reference accepted")
		}
	}
	for _, value := range [][]byte{nil, make([]byte, MaxSecret+1), []byte("bad\x00value")} {
		if err := Set("safe", value); err == nil {
			t.Fatal("invalid secret accepted")
		}
	}
	if _, err := Get("plaintext"); err == nil {
		t.Fatal("plaintext resolved as reference")
	}
}
func TestReferenceScopeAndRotation(t *testing.T) {
	base := ID("config", "server", "TOKEN")
	for _, other := range []string{ID("other", "server", "TOKEN"), ID("config", "other", "TOKEN"), ID("config", "server", "PASSWORD")} {
		if base == other {
			t.Fatal("credential scopes collide")
		}
	}
	if NewID("config", "server", "TOKEN") == NewID("config", "server", "TOKEN") {
		t.Fatal("credential updates would overwrite previous value")
	}
}
func TestNativeVaultRoundTrip(t *testing.T) {
	if os.Getenv("MCPDECK_TEST_OS_VAULT") != "1" {
		t.Skip("opt-in test writes a disposable OS credential")
	}
	id := NewID(t.TempDir(), "fixture", "TOKEN")
	// Parser-sensitive characters never reach shell or process argv.
	value := []byte("fixture-only spaces \" ' $ ` \\ Türkçe\nsecond line\n")
	// Removal is platform-specific in an opt-in test, never touches existing entries.
	t.Cleanup(func() {
		if err := deleteSecret(id); err != nil {
			t.Error(err)
		}
	})
	if err := Set(id, value); err != nil {
		t.Fatal(err)
	}
	actual, err := Get(Reference(id))
	if err != nil || actual != string(value) {
		t.Fatal("OS vault round-trip failed")
	}
}

func TestNativeVaultMaximumSize(t *testing.T) {
	if os.Getenv("MCPDECK_TEST_OS_VAULT") != "1" {
		t.Skip("opt-in OS credential test")
	}
	id := NewID(t.TempDir(), "fixture", "MAX")
	t.Cleanup(func() { _ = deleteSecret(id) })
	value := bytes.Repeat([]byte("x"), MaxSecret)
	if err := Set(id, value); err != nil {
		t.Fatal(err)
	}
	actual, err := Get(Reference(id))
	if err != nil || actual != string(value) {
		t.Fatal("maximum-length secret round-trip failed")
	}
}
