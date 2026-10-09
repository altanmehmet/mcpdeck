package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func fixturePayload(t *testing.T, extra string) []byte {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	binary := []byte("verified application")
	h := sha256.Sum256(binary)
	manifest, _ := json.Marshal(map[string]string{"application": "MCPDeck Desktop", "platform": "windows", "architecture": "amd64", "sha256": hex.EncodeToString(h[:])})
	for name, data := range map[string][]byte{"package/package.json": manifest, "package/MCPDeck.exe": binary} {
		w, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		w.Write(data)
	}
	if extra != "" {
		w, err := z.Create(extra)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte("unexpected"))
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func TestSetupPayloadIntegrityAndArchitecture(t *testing.T) {
	data := fixturePayload(t, "")
	h := sha256.Sum256(data)
	expectedPayloadHash = hex.EncodeToString(h[:])
	defer func() { expectedPayloadHash = "" }()
	root, err := extractPackage(data, t.TempDir(), "amd64")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "MCPDeck.exe"))
	if err != nil || string(raw) != "verified application" {
		t.Fatal("application payload changed", err)
	}
	if _, err = extractPackage(data, t.TempDir(), "arm64"); err == nil {
		t.Fatal("wrong architecture accepted")
	}
	if _, err = extractPackage(append(data, 0), t.TempDir(), "amd64"); err == nil {
		t.Fatal("tampered archive accepted")
	}
}
func TestSetupRejectsEscapingPaths(t *testing.T) {
	for _, path := range []string{"../outside", "/outside", "C:/outside", "package/../../outside", "package/../outside", "package\\outside"} {
		data := fixturePayload(t, path)
		h := sha256.Sum256(data)
		expectedPayloadHash = hex.EncodeToString(h[:])
		if _, err := extractPackage(data, t.TempDir(), "amd64"); err == nil {
			t.Fatal("unsafe path accepted", path)
		}
	}
	expectedPayloadHash = ""
}
