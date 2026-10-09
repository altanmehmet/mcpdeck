package main

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestArchiveKeepsExecutableAndStaysInsideSource(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "MCPDeck-package")
	if err := os.Mkdir(source, 0755); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(source, "MCPDeck")
	if err := os.WriteFile(binary, []byte("packaged executable"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "private-config"), []byte("not for packaging"), 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "app.zip")
	if err := zipTree(source, path); err != nil {
		t.Fatal(err)
	}
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	found := false
	for _, f := range r.File {
		if f.Name == "private-config" {
			t.Fatal("outside file packaged")
		}
		if f.Name == "MCPDeck-package/MCPDeck" {
			found = true
			if f.Mode().Perm() != 0755 {
				t.Fatal("executable mode lost")
			}
			in, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			raw, _ := io.ReadAll(in)
			in.Close()
			if string(raw) != "packaged executable" {
				t.Fatal("payload changed")
			}
		}
	}
	if !found {
		t.Fatal("executable missing")
	}
}
func TestPackageRejectsSymlinkAssets(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	os.Mkdir(source, 0755)
	if err := os.Symlink(filepath.Join(dir, "private"), filepath.Join(source, "linked")); err != nil {
		t.Skip("host cannot create symlinks", err)
	}
	if err := zipTree(source, filepath.Join(dir, "app.zip")); err == nil {
		t.Fatal("linked asset archived")
	}
}
