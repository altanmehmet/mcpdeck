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
