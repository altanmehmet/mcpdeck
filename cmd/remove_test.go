package cmd

import (
	"bufio"
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRemoveAllProfilesKeepsExternalSettings(t *testing.T) {
	s := allProfilesFixture(t)
	if err := runCommand(s, "add", "--name", "demo", "--command", "echo"); err != nil {
		t.Fatal(err)
	}
	if err := runCommand(s, "remove", "demo", "--yes"); err != nil {
		t.Fatal(err)
	}
	d, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := d.Servers["demo"]; exists {
		t.Fatal("record remains")
	}
	for name, p := range d.Profiles {
		root, entries := readAgentConfig(t, p)
		if entries["demo"] != nil || d.IsEnabled(name, "demo") {
			t.Fatalf("MCP remains in %s", name)
		}
		if entries["external"] == nil || root["keep"] != "value" {
			t.Fatalf("external settings removed in %s", name)
		}
	}
}

func TestRemovePartialFailureCanBeRetried(t *testing.T) {
	s := allProfilesFixture(t)
	if err := runCommand(s, "add", "--name", "demo", "--command", "echo"); err != nil {
		t.Fatal(err)
	}
	d, _ := s.Load()
	path := d.Profiles["cursor"].TargetPath
	if err := os.WriteFile(path, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := runCommand(s, "remove", "demo", "--yes"); err == nil || !strings.Contains(err.Error(), "removal incomplete") {
		t.Fatalf("expected partial failure: %v", err)
	}
	d, _ = s.Load()
	if _, ok := d.Servers["demo"]; !ok {
		t.Fatal("retry record lost")
	}
	for name := range d.Profiles {
		if d.IsEnabled(name, "demo") {
			t.Fatal("still enabled")
		}
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != "broken" {
		t.Fatal("invalid config overwritten")
	}
	if err := os.WriteFile(path, []byte(`{"mcpServers":{"demo":{"command":"echo"},"external":{"command":"keep"}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := runCommand(s, "remove", "demo", "--yes"); err != nil {
		t.Fatal(err)
	}
	_, entries := readAgentConfig(t, d.Profiles["cursor"])
	if entries["demo"] != nil || entries["external"] == nil {
		t.Fatal("retry failed")
	}
}

func TestRemoveCancellationDoesNotChangeDeck(t *testing.T) {
	s := allProfilesFixture(t)
	if err := runCommand(s, "add", "--name", "demo", "--command", "echo"); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(s.Path)
	c := NewRoot()
	c.SetArgs([]string{"--config", s.Path, "remove", "demo"})
	c.SetIn(strings.NewReader("\n"))
	c.SetOut(&bytes.Buffer{})
	if err := c.Execute(); err == nil {
		t.Fatal("empty confirmation accepted")
	}
	after, _ := os.ReadFile(s.Path)
	if !bytes.Equal(before, after) {
		t.Fatal("cancel changed deck")
	}
}

func TestSimpleApprovalRequiresExplicitYes(t *testing.T) {
	for _, tc := range []struct {
		answer string
		want   bool
	}{{"y\n", true}, {"YES\n", true}, {"\n", false}, {"no\n", false}, {"yes", false}} {
		if got := confirmYes(bufio.NewReader(strings.NewReader(tc.answer)), &bytes.Buffer{}, "Install?"); got != tc.want {
			t.Fatalf("confirmation %q = %v", tc.answer, got)
		}
	}
}

func TestInstallYesApprovalWithoutCopyingDigest(t *testing.T) {
	for _, answer := range []string{"y\n", "\n"} {
		t.Run(strings.TrimSpace(answer)+"confirmation", func(t *testing.T) {
			s, p, path := installFixture(t)
			t.Setenv("INSTALL_SECRET", "private-test-value")
			c := NewRoot()
			var out bytes.Buffer
			c.SetOut(&out)
			c.SetErr(&out)
			c.SetIn(strings.NewReader(answer))
			c.SetArgs([]string{"--config", s.Path, "install", "--plan", path})
			err := c.Execute()
			d, _ := s.Load()
			_, installed := d.Servers[p.Name]
			if answer == "y\n" && (err != nil || !installed) {
				t.Fatalf("yes did not install: %v\n%s", err, out.String())
			}
			if answer == "\n" && (err == nil || installed) {
				t.Fatal("empty answer installed")
			}
			if strings.Contains(out.String(), "first 12") || strings.Contains(out.String(), "private-test-value") {
				t.Fatal("digest copying or secret exposure")
			}
		})
	}
}

func TestRemoveExternalWithoutImport(t *testing.T) {
	s := allProfilesFixture(t)
	before, _ := os.ReadFile(s.Path)
	c := NewRoot()
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&out)
	c.SetIn(strings.NewReader("y\n"))
	c.SetArgs([]string{"--config", s.Path, "remove", "external"})
	if err := c.Execute(); err != nil {
		t.Fatal(err, out.String())
	}
	d, _ := s.Load()
	for key, p := range d.Profiles {
		_, entries := readAgentConfig(t, p)
		if entries["external"] != nil {
			t.Fatal("external MCP remains", key)
		}
	}
	after, _ := os.ReadFile(s.Path)
	if !bytes.Equal(before, after) {
		t.Fatal("external MCP imported during removal")
	}
}
