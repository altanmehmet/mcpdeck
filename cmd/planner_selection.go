package cmd

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/altanmehmet/mcpdeck/internal/install"
	"github.com/altanmehmet/mcpdeck/internal/tui"
	"github.com/spf13/cobra"
)

// Detection checks executable availability, not whether an account is signed in.
func detectedPlanners(saved install.PlannerOptions) []install.PlannerOptions {
	var options []install.PlannerOptions
	for _, provider := range []string{"codex", "claude", "gemini", "copilot"} {
		option := install.PlannerOptions{Provider: provider}
		if saved.Provider == provider {
			option = saved
		}
		var path string
		var err error
		if option.Executable != "" {
			path, err = exec.LookPath(option.Executable)
		} else {
			path, err = install.PlannerPath(provider)
		}
		if err == nil {
			option.Executable = path
			options = append(options, option)
		}
	}
	// Explicitly configured APIs remain available without showing unconfigured providers.
	if saved.IsAPI() && saved.Validate() == nil {
		options = append(options, saved)
	}
	return options
}

func chooseDetectedPlanner(c *cobra.Command, input *bufio.Reader, saved install.PlannerOptions) (install.PlannerOptions, error) {
	options := detectedPlanners(saved)
	if len(options) == 0 {
		return saved, fmt.Errorf("no planning agents detected; install Codex, Claude, Gemini or Copilot CLI and sign in, or configure an API with mcpdeck planner set")
	}
	labels := make([]string, len(options))
	initial := 0
	for i, option := range options {
		labels[i] = option.Provider
		if option.Provider == saved.Provider {
			initial = i
		}
	}
	if session := consoleSessionFor(c); session != nil {
		session.send(tui.ConsoleEvent{Kind: "choices", Text: "Choose a detected planning agent", Choices: labels, Selected: initial})
		select {
		case value := <-session.normal:
			index, err := strconv.Atoi(value)
			if err != nil || index < 0 || index >= len(options) {
				return saved, fmt.Errorf("invalid planner selection")
			}
			return options[index], nil
		case <-session.ctx.Done():
			return saved, session.ctx.Err()
		}
	}
	for {
		fmt.Fprintln(c.OutOrStdout(), "Detected planning agents:")
		for i, label := range labels {
			fmt.Fprintf(c.OutOrStdout(), "  %d. %s\n", i+1, label)
		}
		fmt.Fprintf(c.OutOrStdout(), "Choose a number (Enter keeps %s): ", labels[initial])
		value, err := input.ReadString('\n')
		if err != nil {
			return saved, err
		}
		value = strings.TrimSpace(value)
		if value == "" {
			return options[initial], nil
		}
		index, err := strconv.Atoi(value)
		if err == nil && index > 0 && index <= len(options) {
			return options[index-1], nil
		}
		fmt.Fprintln(c.OutOrStdout(), "Select a number from the list.")
	}
}

func choosePlannerModel(c *cobra.Command, input *bufio.Reader, option install.PlannerOptions) (install.PlannerOptions, error) {
	current := option.Model
	if current == "" {
		current = "CLI default"
	}
	for {
		fmt.Fprintf(c.OutOrStdout(), "Model for %s (Enter keeps %s; /default resets; or enter model ID): ", option.Provider, current)
		value, err := input.ReadString('\n')
		if err != nil {
			return option, err
		}
		value = strings.TrimSpace(value)
		if value == "" {
			break
		}
		if value == "/default" {
			option.Model = ""
			break
		}
		if len(value) > 256 || strings.ContainsAny(value, " \t\r\n\x00") {
			fmt.Fprintln(c.OutOrStdout(), "Enter a model ID without spaces, up to 256 bytes.")
			continue
		}
		option.Model = value
		break
	}
	selected := option.Model
	if selected == "" {
		selected = "CLI default"
	}
	fmt.Fprintf(c.OutOrStdout(), "Selected model: %s\n", safeText(selected))
	return option, nil
}
