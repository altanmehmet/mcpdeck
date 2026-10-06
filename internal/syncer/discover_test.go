package syncer

import (
	"bytes"
	"encoding/json"
	"github.com/altanmehmet/mcpdeck/internal/testutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
)

func externalFixture(t *testing.T) (store.Store, *model.Deck) {
	t.Helper()
	dir := t.TempDir()
	d := &model.Deck{Version: 1, Servers: map[string]model.ServerConfig{}, Profiles: store.DefaultProfiles()}
	for key, p := range store.AdditionalProfiles() {
		d.Profiles[key] = p
	}
	for key, p := range d.Profiles {
		p.TargetPath = filepath.Join(dir, key+".config")
		d.Profiles[key] = p
		root := map[string]any{"keep": "value"}
		servers, err := profileServers(root, p.Format, true)
		if err != nil {
			t.Fatal(err)
		}
		// Unknown behavior and credentials must survive scanning without importing.
		servers["outside"] = map[string]any{"command": "custom", "unknown_option": true, "env": map[string]any{"TOKEN": "secret-do-not-show"}}
		servers["other"] = map[string]any{"command": "keep"}
		raw, err := encodeConfig(root, p.Format)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p.TargetPath, raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	s := store.Store{Path: filepath.Join(dir, "deck.json")}
	if err := s.Save(d); err != nil {
		t.Fatal(err)
	}
	return s, d
}

func TestDiscoverAndRemoveExternalAcrossAllFormats(t *testing.T) {
	s, d := externalFixture(t)
	before, _ := os.ReadFile(s.Path)
	found, err := Discover(d)
	if err != nil {
		t.Fatal(err)
	}
	item := found["outside"]
	if len(item.Profiles) != len(d.Profiles) {
		t.Fatal("not discovered in every format")
	}
	raw, _ := json.Marshal(item)
	if strings.Contains(string(raw), "secret-do-not-show") || strings.Contains(string(raw), "TOKEN") {
		t.Fatal("credentials in metadata")
	}
	if err := RemoveDiscovered(s, item); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(s.Path)
	if !bytes.Equal(before, after) {
		t.Fatal("external settings adopted into deck")
	}
	for key, p := range d.Profiles {
		_, root, servers, err := readProfile(p)
		if err != nil || servers["outside"] != nil || servers["other"] == nil || root["keep"] != "value" {
			t.Fatalf("incorrect removal in %s: %v", key, err)
		}
		backup, err := os.ReadFile(p.TargetPath + ".mcpdeck-remove-backup")
		if err != nil || !bytes.Contains(backup, []byte("secret-do-not-show")) {
			t.Fatal("missing original backup", key)
		}
		testutil.AssertPrivateFile(t, p.TargetPath+".mcpdeck-remove-backup")
	}
}

func TestExternalRemovalRejectsChangedRecipe(t *testing.T) {
	s, d := externalFixture(t)
	found, _ := Discover(d)
	p := d.Profiles["cursor"]
	_, root, servers, _ := readProfile(p)
	servers["outside"] = map[string]any{"command": "new-command"}
	raw, _ := encodeConfig(root, p.Format)
	if err := os.WriteFile(p.TargetPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := RemoveDiscovered(s, found["outside"]); err == nil {
		t.Fatal("changed recipe removed")
	}
	after, _ := os.ReadFile(p.TargetPath)
	if !bytes.Equal(raw, after) {
		t.Fatal("changed configuration overwritten")
	}
}

func TestDiscoveryReportsMalformedProfilesWithoutLeakingContent(t *testing.T) {
	_, d := externalFixture(t)
	if err := os.WriteFile(d.Profiles["cursor"].TargetPath, []byte("secret-do-not-show"), 0600); err != nil {
		t.Fatal(err)
	}
	found, err := Discover(d)
	if err == nil || strings.Contains(err.Error(), "secret-do-not-show") || len(found["outside"].Profiles) != len(d.Profiles)-1 {
		t.Fatal("incorrect partial discovery", err)
	}
}
