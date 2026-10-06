package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/altanmehmet/mcpdeck/internal/install"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"github.com/altanmehmet/mcpdeck/internal/syncer"
	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"
)

// repairCommand updates an MCP already owned by MCPDeck. It keeps the old
// connection active until the revised recipe has been installed and probed.
func repairCommand(get func() store.Store) *cobra.Command {
	var provider, planner, plannerModel, baseURL, keyEnv, approval string
	var valuesStdin, wait bool
	c := &cobra.Command{Use: "repair <mcp-name> [problem description]", Aliases: []string{"update"}, Short: "Diagnose, repair and update an existing MCP with an agent", Args: cobra.RangeArgs(1, 2)}
	c.RunE = func(c *cobra.Command, args []string) (runErr error) {
		input := bufio.NewReader(c.InOrStdin())
		if wait {
			defer func() {
				if runErr != nil {
					fmt.Fprintln(c.OutOrStdout(), "Repair result:", safeText(runErr.Error()))
				}
				fmt.Fprint(c.OutOrStdout(), "Press Enter to return to MCPDeck...")
				_, _ = input.ReadString('\n')
			}()
		}
		s := get()
		d, err := s.Load()
		if err != nil {
			return err
		}
		name := args[0]
		old, ok := d.Servers[name]
		if !ok {
			return fmt.Errorf("MCP %q is not managed by MCPDeck; import it first, then repair it", name)
		}
		targets := enabledTargets(d, name)
		if len(targets) == 0 {
			return fmt.Errorf("MCP %q is not enabled for any agent", name)
		}
		terminalFile, interactiveTerminal := c.InOrStdin().(*os.File)
		interactive := interactiveTerminal && term.IsTerminal(terminalFile.Fd()) && !valuesStdin
		issue := "The MCP is reported as unhealthy. Diagnose its current connection and propose a complete repair recipe."
		if len(args) == 2 && strings.TrimSpace(args[1]) != "" {
			issue = strings.TrimSpace(args[1])
		}
		resolved, probeErr := model.Resolve(old)
		if probeErr == nil {
			ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
			_, probeErr = install.Probe(ctx, name, resolved)
			cancel()
		}
		observed := "healthy probe"
		if probeErr != nil {
			observed = "health check error: " + safeText(probeErr.Error())
		}
		opts, err := loadPlanner(s)
		if err != nil {
			return err
		}
		if c.Flags().Changed("provider") {
			opts.Provider = provider
		}
		if c.Flags().Changed("model") {
			opts.Model = plannerModel
		}
		if c.Flags().Changed("planner") {
			opts.Executable = planner
		}
		if c.Flags().Changed("base-url") {
			opts.BaseURL = baseURL
		}
		if c.Flags().Changed("key-env") {
			opts.KeyEnv = keyEnv
		}
		if interactive && !c.Flags().Changed("provider") {
			fmt.Fprintf(c.OutOrStdout(), "Connected planning agent: %s\nOptions: %s\nChoose an agent (Enter keeps %s): ", opts.Provider, strings.Join(install.Providers, ", "), opts.Provider)
			selected, e := input.ReadString('\n')
			if e != nil {
				return e
			}
			selected = strings.TrimSpace(selected)
			if selected != "" {
				opts.Provider = selected
			}
		}
		if interactive && opts.IsAPI() && opts.Model == "" {
			fmt.Fprint(c.OutOrStdout(), "Model name: ")
			v, e := input.ReadString('\n')
			if e != nil {
				return e
			}
			opts.Model = strings.TrimSpace(v)
		}
		if interactive && opts.Provider == "openai-compatible" && opts.BaseURL == "" {
			fmt.Fprint(c.OutOrStdout(), "API base URL: ")
			v, e := input.ReadString('\n')
			if e != nil {
				return e
			}
			opts.BaseURL = strings.TrimSpace(v)
		}
		if err = opts.Validate(); err != nil {
			return err
		}
		if interactive && opts.IsAPI() && os.Getenv(opts.CredentialEnv()) == "" {
			fmt.Fprint(c.OutOrStdout(), "API key (hidden and not saved; may be empty for local services): ")
			secret, e := term.ReadPassword(terminalFile.Fd())
			fmt.Fprintln(c.OutOrStdout())
			if e != nil {
				return fmt.Errorf("cannot read API key")
			}
			opts.APIKey = string(secret)
		}
		connection, err := redactedConnection(old, name)
		if err != nil {
			return err
		}
		request := fmt.Sprintf("Repair the existing MCP %s. User report: %s\nObserved local status: %s\nCurrent connection shape (redacted): %s\nKeep the exact MCP name and return a complete executable recipe. Reuse compatible installed software. Do not ask for or echo secrets; use named placeholders for private inputs. This is an update, so preserve the connection contract unless the evidence requires a change.", name, issue, observed, connection)
		fmt.Fprintf(c.OutOrStdout(), "Preparing a repair plan with %s. No files change until the revised recipe is approved.\n", opts.Provider)
		generated, err := planWithProgress(c.Context(), opts, request, c.OutOrStdout(), interactive)
		if err != nil {
			return err
		}
		generated.Name = name
		generated.Connection, err = renamePlanConnection(generated.Connection, name)
		if err != nil {
			return err
		}
		if err = generated.Validate(); err != nil {
			return err
		}
		showPlan(c.OutOrStdout(), generated, targets)
		if interactive && approval == "" {
			generated, err = reviewPlanChat(c.Context(), opts, request, generated, input, c.OutOrStdout(), targets, "", planWithProgress)
			if err != nil {
				return err
			}
		}
		if generated.HasPlaceholderConnection() {
			return fmt.Errorf("the repair recipe still has no runnable connection")
		}
		var manualReviewed bool
		for {
			if err = ensureRequirements(c.Context(), generated, input, c.OutOrStdout(), interactive); err != nil {
				return err
			}
			manualReviewed, err = confirmManualPrerequisites(generated, input, c.OutOrStdout(), interactive)
			var question *manualQuestionError
			if !errorsAs(err, &question) {
				if err != nil {
					return err
				}
				break
			}
			generated, err = reviewPlanChat(c.Context(), opts, request, generated, input, c.OutOrStdout(), targets, question.message, planWithProgress)
			if err != nil {
				return err
			}
		}
		if err = generated.PreflightAfterManualReview(manualReviewed); err != nil {
			return err
		}
		if approval == "" {
			outputHeading(c.OutOrStdout(), "UPDATE APPROVAL")
			outputParagraph(c.OutOrStdout(), "The reviewed repair commands can download and run programs with your user permissions.")
			if !confirmYes(input, c.OutOrStdout(), "Apply this repair?") {
				return fmt.Errorf("repair cancelled")
			}
			approval = generated.Digest()
		}
		if approval != generated.Digest() {
			return fmt.Errorf("plan changed or approval invalid")
		}
		values := map[string]string{}
		for k, v := range old.Variables {
			values[k] = v
		}
		if valuesStdin {
			raw, e := io.ReadAll(io.LimitReader(input, 1024*1024+1))
			if e != nil || len(raw) > 1024*1024 || json.Unmarshal(raw, &values) != nil {
				return fmt.Errorf("expected an input value object on stdin")
			}
		}
		dir := filepath.Join(filepath.Dir(s.Path), "installations", name)
		items, err := model.Import(generated.Connection, name, map[string]string{"INSTALL_DIR": dir})
		if err != nil {
			return err
		}
		for _, key := range items[0].Required {
			if values[key] != "" {
				continue
			}
			file, ok := c.InOrStdin().(*os.File)
			if valuesStdin || !ok || !term.IsTerminal(file.Fd()) {
				return fmt.Errorf("missing input %s; set its environment variable or supply --values-stdin", key)
			}
			fmt.Fprintf(c.OutOrStdout(), "%s (hidden): ", safeText(key))
			raw, e := term.ReadPassword(file.Fd())
			fmt.Fprintln(c.OutOrStdout())
			if e != nil {
				return fmt.Errorf("cannot read input")
			}
			values[key] = string(raw)
		}
		item, err := generated.Bind(dir, values)
		if err != nil {
			return err
		}
		resolved, err = model.Resolve(item.Server)
		if err != nil {
			return err
		}
		if err = generated.RunAfterManualReview(c.Context(), dir, approval, c.OutOrStdout(), manualReviewed); err != nil {
			return err
		}
		fmt.Fprintln(c.OutOrStdout(), "Starting the repaired MCP and verifying initialize plus tools/list...")
		ctx, cancel := context.WithTimeout(c.Context(), 60*time.Second)
		count, err := install.Probe(ctx, name, resolved)
		cancel()
		if err != nil {
			return err
		}
		if err = s.Update(func(current *model.Deck) error {
			if _, ok := current.Servers[name]; !ok {
				return fmt.Errorf("MCP was removed while repair was running")
			}
			current.Servers[name] = item.Server
			return nil
		}); err != nil {
			return err
		}
		d, err = s.Load()
		if err != nil {
			return err
		}
		sy := syncer.Syncer{Deck: d, ConfigPath: s.Path}
		results, syncErr := sy.SyncTargets(targets)
		for _, result := range results {
			fmt.Fprintf(c.OutOrStdout(), "%s: %s\n", result.Profile, result.Status)
		}
		if syncErr != nil {
			return fmt.Errorf("MCP repaired and saved, but synchronization is incomplete: %w; use mcpdeck sync status, then mcpdeck sync --retry-failed", syncErr)
		}
		fmt.Fprintf(c.OutOrStdout(), "Repaired and verified: %d tools. Updated %d agent configurations. Restart or reload the agents.\n", count, len(targets))
		return nil
	}
	c.Flags().StringVar(&provider, "provider", "", "Override saved planner provider")
	c.Flags().StringVar(&planner, "planner", "", "Selected planner CLI executable path")
	c.Flags().StringVar(&plannerModel, "model", "", "Override planner model")
	c.Flags().StringVar(&baseURL, "base-url", "", "OpenAI-compatible API base URL")
	c.Flags().StringVar(&keyEnv, "key-env", "", "API key environment variable name")
	c.Flags().StringVar(&approval, "approve", "", "Approve the exact SHA-256 plan digest")
	c.Flags().BoolVar(&valuesStdin, "values-stdin", false, "Read private inputs as a JSON object from stdin")
	c.Flags().BoolVar(&wait, "wait", false, "Wait for Enter before returning")
	_ = c.Flags().MarkHidden("wait")
	return c
}

func enabledTargets(d *model.Deck, name string) []string {
	var out []string
	for key := range d.Profiles {
		if d.IsEnabled(key, name) {
			out = append(out, key)
		}
	}
	sort.Strings(out)
	return out
}

// redactedConnection preserves the recipe shape while replacing values that
// could contain credentials. It is safe to include in an agent prompt.
func redactedConnection(s model.ServerConfig, name string) (string, error) {
	clone := s
	clone.CachedTools = nil
	clone.Args = append([]string(nil), s.Args...)
	clone.Headers = map[string]string{}
	for k, v := range s.Headers {
		clone.Headers[k] = v
	}
	clone.Env = map[string]string{}
	for k, v := range s.Env {
		clone.Env[k] = v
	}
	clone.Variables = map[string]string{}
	for key, value := range s.Variables {
		clone.Variables[key] = "${" + key + "}"
		if value != "" {
			clone.URL = strings.ReplaceAll(clone.URL, value, "${"+key+"}")
			for k, v := range clone.Headers {
				clone.Headers[k] = strings.ReplaceAll(v, value, "${"+key+"}")
			}
			for i, v := range clone.Args {
				clone.Args[i] = strings.ReplaceAll(v, value, "${"+key+"}")
			}
			for k, v := range clone.Env {
				clone.Env[k] = strings.ReplaceAll(v, value, "${"+key+"}")
			}
		}
	}
	for i, value := range clone.Args {
		lower := strings.ToLower(value)
		if strings.Contains(lower, "password") || strings.Contains(lower, "token=") || strings.Contains(lower, "secret") || strings.Contains(lower, "api_key") {
			clone.Args[i] = "<redacted>"
		}
	}
	for key, value := range clone.Headers {
		if !strings.Contains(value, "${") {
			clone.Headers[key] = "<redacted>"
		}
	}
	for key, value := range clone.Env {
		if !strings.Contains(value, "${") && !strings.HasPrefix(value, "$") {
			clone.Env[key] = "<redacted>"
		}
	}
	if parsed, err := url.Parse(clone.URL); err == nil && parsed.Host != "" {
		if parsed.User != nil {
			parsed.User = url.User(parsed.User.Username())
		}
		parsed.RawQuery = ""
		parsed.Fragment = ""
		clone.URL = parsed.String()
	}
	root := map[string]map[string]model.ServerConfig{"mcpServers": {name: clone}}
	raw, err := json.Marshal(root)
	return string(raw), err
}

// Local alias keeps repair.go independent from errors.As in the long command body.
func errorsAs(err error, target any) bool { return errors.As(err, target) }
