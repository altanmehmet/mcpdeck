package cmd

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/altanmehmet/mcpdeck/internal/install"
	"github.com/spf13/cobra"
)

func TestDetectedPlannerListExcludesMissingAndPreservesConfiguredModel(t *testing.T) {
	dir := t.TempDir()
	name := "claude"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	executable := filepath.Join(dir, name)
	if err := os.WriteFile(executable, []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	saved := install.PlannerOptions{Provider: "claude", Model: "custom-model", Executable: executable}
	options := detectedPlanners(saved)
	found := false
	for _, option := range options {
		if option.Provider == "gemini" || option.IsAPI() {
			t.Fatalf("undetected provider offered: %+v", option)
		}
		if option.Provider == "claude" {
			found = true
			if option.Model != saved.Model || option.Executable != executable {
				t.Fatal("saved CLI settings lost")
			}
		}
	}
	if !found {
		t.Fatal("installed CLI not offered")
	}
	c := &cobra.Command{}
	var out bytes.Buffer
	c.SetOut(&out)
	selected, err := chooseDetectedPlanner(c, bufio.NewReader(strings.NewReader("not-an-agent\n\n")), saved)
	if err != nil || selected != saved {
		t.Fatalf("selection: %+v, %v", selected, err)
	}
	if !strings.Contains(out.String(), "Select a number from the list") {
		t.Fatal("free-text provider accepted")
	}
}

func TestPlannerModelSelection(t *testing.T) {
	for _, tc := range []struct{ input, want string }{{"\n", "saved-model"}, {"chosen-model\n", "chosen-model"}, {"/default\n", ""}, {"invalid model\nvalid-model\n", "valid-model"}} {
		c := &cobra.Command{}
		c.SetOut(&bytes.Buffer{})
		option, err := choosePlannerModel(c, bufio.NewReader(strings.NewReader(tc.input)), install.PlannerOptions{Provider: "copilot", Model: "saved-model"})
		if err != nil || option.Model != tc.want {
			t.Fatal(option, err)
		}
	}
}

func TestAgyModelCatalogChoicePreservesDefaultsAndIDs(t *testing.T) {
	models := []install.PlannerModel{{ID: "fixture-one", Label: "Model one"}, {ID: "fixture-two", Label: "Model two"}}
	for _, tc := range []struct{ input, want string }{{"\n", "saved-model"}, {"2\n", "fixture-one"}, {"3\n", "fixture-two"}, {"4\ncustom-model\n", "custom-model"}} {
		c := &cobra.Command{}
		c.SetOut(&bytes.Buffer{})
		got, err := selectAgyModel(c, bufio.NewReader(strings.NewReader(tc.input)), install.PlannerOptions{Provider: "agy", Model: "saved-model"}, models)
		if err != nil || got.Model != tc.want {
			t.Fatal(got, err)
		}
	}
}
