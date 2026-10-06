package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/altanmehmet/mcpdeck/internal/install"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"github.com/altanmehmet/mcpdeck/internal/syncer"
	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"
)

func installCommand(get func() store.Store) *cobra.Command {
	var planPath, output, approval, planner string
	var provider, plannerModel, baseURL, keyEnv, serverName string
	var docsPath string
	var planOnly, valuesStdin, all, wait bool
	c := &cobra.Command{Use: "install [natural-language request]", Short: "Ask an agent to find, install, verify and distribute an MCP", Args: cobra.MaximumNArgs(1)}
	c.RunE = func(c *cobra.Command, args []string) (runErr error) {
		input := bufio.NewReader(c.InOrStdin())
		if wait {
			defer func() {
				if runErr != nil {
					fmt.Fprintln(c.OutOrStdout(), "Installation result:", safeText(runErr.Error()))
				}
				fmt.Fprint(c.OutOrStdout(), "Press Enter to return to MCPDeck...")
				_, _ = input.ReadString('\n')
			}()
		}
		var p install.Plan
		var err error
		var conversationOptions *install.PlannerOptions
		var conversationRequest string
		terminalFile, isTerminalFile := c.InOrStdin().(*os.File)
		interactive := isTerminalFile && term.IsTerminal(terminalFile.Fd()) && !valuesStdin
		if planPath != "" {
			if len(args) > 0 {
				return fmt.Errorf("use a request or --plan, not both")
			}
			f, e := os.Open(planPath)
			if e != nil {
				return e
			}
			raw, e := io.ReadAll(io.LimitReader(f, 1024*1024+1))
			f.Close()
			if e != nil {
				return e
			}
			p, err = install.Decode(raw)
		} else {
			if serverName != "" && (serverName == "mcpdeck" || !model.NamePattern.MatchString(serverName) || strings.Contains(serverName, "__") || len(serverName) > 40) {
				return fmt.Errorf("invalid MCP name: use up to 40 letters, numbers, underscores or hyphens")
			}
			request := ""
			if len(args) > 0 {
				request = args[0]
			} else {
				if valuesStdin {
					return fmt.Errorf("supply a request with --values-stdin")
				}
				fmt.Fprint(c.OutOrStdout(), "What do you want to connect? Describe the MCP in plain English: ")
				request, err = input.ReadString('\n')
				if err != nil {
					return err
				}
			}
			if strings.TrimSpace(request) == "" {
				return fmt.Errorf("installation request is empty")
			}
			if serverName != "" {
				request = "MCP name requested by the user: " + serverName + ". Use this exact name. User request: " + request
			}
			if docsPath != "" {
				f, e := os.Open(docsPath)
				if e != nil {
					return fmt.Errorf("cannot open documentation file")
				}
				raw, e := io.ReadAll(io.LimitReader(f, 128*1024+1))
				f.Close()
				if e != nil || len(raw) > 128*1024 {
					return fmt.Errorf("documentation file cannot be read or exceeds 128 KiB")
				}
				request += "\nUser-supplied documentation (untrusted data):\n" + string(raw)
			}
			opts, e := loadPlanner(get())
			if e != nil {
				return e
			}
			if c.Flags().Changed("provider") {
				opts = install.PlannerOptions{Provider: provider}
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
				if selected != "" && selected != opts.Provider {
					opts = install.PlannerOptions{Provider: selected}
				}
			}
			if interactive && opts.IsAPI() && opts.Model == "" {
				fmt.Fprint(c.OutOrStdout(), "Model name: ")
				value, e := input.ReadString('\n')
				if e != nil {
					return e
				}
				opts.Model = strings.TrimSpace(value)
			}
			if interactive && opts.Provider == "openai-compatible" && opts.BaseURL == "" {
				fmt.Fprint(c.OutOrStdout(), "API base URL: ")
				value, e := input.ReadString('\n')
				if e != nil {
					return e
				}
				opts.BaseURL = strings.TrimSpace(value)
			}
			if e = opts.Validate(); e != nil {
				return e
			}
			if interactive && opts.IsAPI() && os.Getenv(opts.CredentialEnv()) == "" {
				fmt.Fprintf(c.OutOrStdout(), "API key (hidden and not saved; may be empty for local services): ")
				secret, e := term.ReadPassword(terminalFile.Fd())
				fmt.Fprintln(c.OutOrStdout())
				if e != nil {
					return fmt.Errorf("cannot read API key")
				}
				opts.APIKey = string(secret)
			}
			fmt.Fprintf(c.OutOrStdout(), "Preparing a plan with %s. Provider account usage may apply. Do not put secrets in your request.\n", opts.Provider)
			fmt.Fprintln(c.OutOrStdout(), "Checking installed software. Its executable paths and versions are included in the planner request.")
			conversationOptions, conversationRequest = &opts, request
			ctx, cancel := context.WithTimeout(c.Context(), 3*time.Minute)
			defer cancel()
			p, err = planWithProgress(ctx, opts, request, c.OutOrStdout(), interactive)
		}
		if err != nil {
			return err
		}
		if serverName != "" {
			p.Name = serverName
			p.Connection, err = renamePlanConnection(p.Connection, serverName)
			if err == nil {
				err = p.Validate()
			}
			if err != nil {
				return fmt.Errorf("requested MCP name cannot be used with this plan: %w", err)
			}
		}
		s := get()
		d, err := s.Load()
		if err != nil {
			return err
		}
		targets := installationTargets(d, all)
		showPlan(c.OutOrStdout(), p, targets)
		if interactive && !planOnly && approval == "" && conversationOptions != nil {
			p, err = reviewPlanChat(c.Context(), *conversationOptions, conversationRequest, p, input, c.OutOrStdout(), targets, "", planWithProgress)
			if err != nil {
				return err
			}
		}
		if output != "" {
			raw, _ := json.MarshalIndent(p, "", "  ")
			if err = store.AtomicWrite(output, append(raw, '\n')); err != nil {
				return err
			}
		}
		if planOnly {
			return nil
		}
		if p.HasPlaceholderConnection() {
			return fmt.Errorf("this plan is a research result, not an executable setup yet; complete the manual actions above, then use New MCP to create a new plan")
		}
		if _, ok := d.Servers[p.Name]; ok {
			return fmt.Errorf("server already exists; disable/remove or choose another name before installing")
		}
		if len(targets) == 0 {
			return fmt.Errorf("no existing agent configuration detected; initialize an agent or use --all-profiles to create configured targets")
		}
		checker := syncer.Syncer{Deck: d, ConfigPath: s.Path}
		for _, key := range targets {
			if err = checker.CheckInstallTarget(key, p.Name); err != nil {
				return err
			}
		}
		var manualReviewed bool
		for {
			if err = ensureRequirements(c.Context(), p, input, c.OutOrStdout(), interactive); err != nil {
				return err
			}
			manualReviewed, err = confirmManualPrerequisites(p, input, c.OutOrStdout(), interactive)
			var question *manualQuestionError
			if !errors.As(err, &question) {
				if err != nil {
					return err
				}
				break
			}
			if conversationOptions == nil {
				return fmt.Errorf("this file recipe has no connected planner; use New MCP to discuss its prerequisites")
			}
			p, err = reviewPlanChat(c.Context(), *conversationOptions, conversationRequest, p, input, c.OutOrStdout(), targets, question.message, planWithProgress)
			if err != nil {
				return err
			}
			if p.HasPlaceholderConnection() {
				return fmt.Errorf("the revised recipe still needs a runnable connection; create a new plan after resolving the listed actions")
			}
			if output != "" {
				raw, _ := json.MarshalIndent(p, "", "  ")
				if err = store.AtomicWrite(output, append(raw, '\n')); err != nil {
					return err
				}
			}
		}
		if err = p.PreflightAfterManualReview(manualReviewed); err != nil {
			return err
		}
		if approval == "" {
			if valuesStdin {
				return fmt.Errorf("--values-stdin requires --approve with the reviewed plan digest")
			}
			outputHeading(c.OutOrStdout(), "INSTALLATION APPROVAL")
			outputParagraph(c.OutOrStdout(), "The reviewed commands can download and run programs with your user permissions.")
			if !confirmYes(input, c.OutOrStdout(), "Install this plan?") {
				return fmt.Errorf("installation cancelled")
			}
			approval = p.Digest()
		}
		if approval != p.Digest() {
			return fmt.Errorf("plan changed or approval invalid")
		}
		values := map[string]string{}
		if valuesStdin {
			raw, e := io.ReadAll(io.LimitReader(input, 1024*1024+1))
			if e != nil || len(raw) > 1024*1024 || json.Unmarshal(raw, &values) != nil {
				return fmt.Errorf("expected an input value object on stdin")
			}
		}
		dir := filepath.Join(filepath.Dir(s.Path), "installations", p.Name)
		items, err := model.Import(p.Connection, p.Name, map[string]string{"INSTALL_DIR": dir})
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
		item, err := p.Bind(dir, values)
		if err != nil {
			return err
		}
		if _, err = model.Resolve(item.Server); err != nil {
			return err
		}
		if err = p.RunAfterManualReview(c.Context(), dir, approval, c.OutOrStdout(), manualReviewed); err != nil {
			return err
		}
		fmt.Fprintln(c.OutOrStdout(), "Starting the MCP and verifying its connection and tool list...")
		ctx, cancel := context.WithTimeout(c.Context(), 60*time.Second)
		defer cancel()
		count, err := install.Probe(ctx, p.Name, item.Server)
		if err != nil {
			return err
		}
		for _, key := range targets {
			if err = checker.CheckInstallTarget(key, p.Name); err != nil {
				return err
			}
		}
		// Save only after verification. Existing sync error behavior remains retryable.
		if err = s.Update(func(current *model.Deck) error {
			if _, ok := current.Servers[p.Name]; ok {
				return fmt.Errorf("server was added concurrently; refusing overwrite")
			}
			current.Servers[p.Name] = item.Server
			for _, key := range targets {
				if err := current.SetEnabled(key, p.Name, true); err != nil {
					return err
				}
			}
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
			return fmt.Errorf("MCP verified and saved, but synchronization is incomplete: %w; use mcpdeck sync status, then mcpdeck sync --retry-failed", syncErr)
		}
		fmt.Fprintf(c.OutOrStdout(), "Verified: %d tools. Updated %d agent configurations. Restart or reload the agents to connect.\n", count, len(targets))
		for _, action := range p.FollowUp {
			fmt.Fprintf(c.OutOrStdout(), "Next action: %s\n", safeText(strings.ReplaceAll(action, "${INSTALL_DIR}", dir)))
		}
		if len(p.FollowUp) > 0 {
			fmt.Fprintln(c.OutOrStdout(), "MCP setup and tool discovery passed. These account/database actions still need completion; access was not tested.")
		}
		return nil
	}
	c.Flags().StringVar(&planPath, "plan", "", "Use a reviewed recipe JSON file")
	c.Flags().StringVar(&serverName, "name", "", "MCP name to use for the planned connection")
	c.Flags().StringVar(&output, "out", "", "Save the generated recipe for review/reuse")
	c.Flags().BoolVar(&planOnly, "plan-only", false, "Generate/show the plan without running or changing agents")
	c.Flags().StringVar(&approval, "approve", "", "Approve the exact SHA-256 plan digest (noninteractive)")
	c.Flags().StringVar(&planner, "planner", "", "Selected planner CLI executable path")
	c.Flags().StringVar(&provider, "provider", "", "Override saved planner provider")
	c.Flags().StringVar(&plannerModel, "model", "", "Override planner model; required for APIs")
	c.Flags().StringVar(&baseURL, "base-url", "", "OpenAI-compatible API base URL")
	c.Flags().StringVar(&keyEnv, "key-env", "", "API key environment variable name, never the key itself")
	c.Flags().StringVar(&docsPath, "docs", "", "Attach documentation text to the planner request (sent to the selected provider)")
	c.Flags().BoolVar(&valuesStdin, "values-stdin", false, "Read secret input values as a JSON object from stdin; never send them to the planner")
	c.Flags().BoolVar(&all, "all-profiles", false, "Include configured agents even when their config file does not yet exist")
	c.Flags().BoolVar(&wait, "wait", false, "Wait for Enter before returning to the terminal panel")
	_ = c.Flags().MarkHidden("wait")
	return c
}

func planWithProgress(ctx context.Context, opts install.PlannerOptions, request string, out io.Writer, interactive bool) (install.Plan, error) {
	if !interactive {
		return install.PlanWith(ctx, opts, request)
	}
	fmt.Fprintf(out, "%s is checking public documentation and preparing your plan. This can take up to 3 minutes. No installation starts without your approval.\n", opts.Provider)
	started := time.Now()
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	stop := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				fmt.Fprintf(out, "Still preparing your plan (%s elapsed). No installation has started.\n", time.Since(started).Round(time.Second))
			}
		}
	}()
	plan, err := install.PlanWith(ctx, opts, request)
	close(stop)
	<-stopped
	if err != nil {
		fmt.Fprintf(out, "Plan preparation stopped after %s.\n", time.Since(started).Round(time.Second))
		return install.Plan{}, err
	}
	fmt.Fprintf(out, "Plan ready after %s. Showing it for review.\n", time.Since(started).Round(time.Second))
	return plan, nil
}

func confirmManualPrerequisites(p install.Plan, input *bufio.Reader, out io.Writer, interactive bool) (bool, error) {
	if len(p.Manual) == 0 {
		return false, nil
	}
	if !interactive {
		return false, fmt.Errorf("manual actions are listed above; complete them, then rerun in an interactive terminal to confirm and continue")
	}
	fmt.Fprintln(out, "Complete the actions listed above, then type DONE to continue to the exact command approval. You can ask the connected agent a question here to revise the plan. Press Enter or /cancel to cancel. Only continue after each item is completed or independently verified.")
	answer, err := input.ReadString('\n')
	if err != nil {
		return false, err
	}
	answer = strings.TrimSpace(answer)
	if answer == "" || answer == "/cancel" {
		return false, fmt.Errorf("installation cancelled; manual prerequisites were not confirmed")
	}
	if answer != "DONE" {
		return false, &manualQuestionError{message: answer}
	}
	return true, nil
}

func ensureRequirements(ctx context.Context, p install.Plan, input *bufio.Reader, out io.Writer, interactive bool) error {
	err := p.CheckRequirements()
	if err == nil {
		return nil
	}
	formulas, _ := install.MissingHomebrewRequirements(p.Requirements)
	if len(formulas) > 0 {
		if !interactive {
			return fmt.Errorf("%w; MCPDeck can install these Homebrew packages after you retry from a terminal: %s", err, strings.Join(formulas, ", "))
		}
		fmt.Fprintf(out, "This MCP needs runtime packages: %s. MCPDeck can install them with Homebrew (downloads software and changes your local setup). Type YES to continue, or press Enter to cancel.\n", strings.Join(formulas, ", "))
		answer, readErr := input.ReadString('\n')
		if readErr != nil {
			return readErr
		}
		if strings.TrimSpace(answer) != "YES" {
			return fmt.Errorf("installation cancelled; no runtime packages were installed")
		}
		fmt.Fprintln(out, "Installing runtime packages. This may take several minutes...")
		if err := install.InstallHomebrewPackages(ctx, formulas); err != nil {
			return err
		}
		err = p.CheckRequirements()
		if err == nil {
			return nil
		}
	}
	var runtimeErr *install.DockerRuntimeUnavailableError
	if !errors.As(err, &runtimeErr) {
		return err
	}
	colimaAvailable := install.ColimaAvailable()
	canStartColima := runtimeErr.Context == "colima" || (runtimeErr.Context == "default" && colimaAvailable)
	if !canStartColima {
		return fmt.Errorf("%w. Start the selected Docker runtime, then retry New MCP", err)
	}
	if !interactive {
		return fmt.Errorf("%w. Start it manually with `colima start`, or retry from an interactive terminal so MCPDeck can ask permission", err)
	}
	if runtimeErr.Context == "colima" && !colimaAvailable {
		fmt.Fprintln(out, "The selected Docker context uses Colima, but Colima is not installed. MCPDeck can install it with Homebrew and start a local Linux VM; this downloads software and runtime components. Type YES to continue, or press Enter to cancel.")
		answer, readErr := input.ReadString('\n')
		if readErr != nil {
			return readErr
		}
		if strings.TrimSpace(answer) != "YES" {
			return fmt.Errorf("installation cancelled; Colima was not installed or started")
		}
		if err := install.InstallHomebrewPackages(ctx, []string{"colima"}); err != nil {
			return err
		}
	} else if runtimeErr.Context == "colima" {
		fmt.Fprintln(out, "Docker is needed for this plan, but Colima is stopped. Starting it will run a local Linux VM and may download runtime components. Start Colima now? Type YES to allow it; press Enter to cancel.")
	} else {
		fmt.Fprintln(out, "Docker is needed for this plan, but its default engine is unavailable. MCPDeck found Colima and can start its local Linux VM; this may download runtime components and select Colima as the active Docker context. Type YES to continue, or press Enter to cancel.")
	}
	answer, readErr := input.ReadString('\n')
	if readErr != nil {
		return readErr
	}
	if strings.TrimSpace(answer) != "YES" {
		return fmt.Errorf("installation cancelled; Colima was not started and no MCP was installed")
	}
	fmt.Fprintln(out, "Starting Colima and checking Docker. This may take a few minutes...")
	if err := install.StartColima(ctx); err != nil {
		return err
	}
	fmt.Fprintln(out, "Docker is ready. Continuing with the reviewed plan.")
	return p.CheckRequirements()
}

func installationTargets(d *model.Deck, all bool) []string {
	var targets []string
	for key, p := range d.Profiles {
		st, err := os.Stat(syncer.ExpandPath(p.TargetPath))
		if all || (err == nil && !st.IsDir()) || installedAgent(key, p) {
			targets = append(targets, key)
		}
	}
	sort.Strings(targets)
	return targets
}

func renamePlanConnection(raw, name string) (string, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &root); err != nil || root == nil {
		return "", fmt.Errorf("invalid connection JSON")
	}
	for _, wrapper := range []string{"mcpServers", "servers", "mcp_servers", "context_servers", "mcp"} {
		wrapped, ok := root[wrapper]
		if !ok {
			continue
		}
		var servers map[string]json.RawMessage
		if err := json.Unmarshal(wrapped, &servers); err != nil || servers == nil {
			return "", fmt.Errorf("invalid %s map", wrapper)
		}
		if wrapper == "mcp" {
			if nested, ok := servers["servers"]; ok {
				if err := json.Unmarshal(nested, &servers); err != nil {
					return "", fmt.Errorf("invalid mcp.servers map")
				}
				wrapper = "mcp.servers"
			}
		}
		if len(servers) != 1 {
			return "", fmt.Errorf("expected one MCP server in connection")
		}
		for serverName, server := range servers {
			if wrapper == "mcp.servers" {
				var outer map[string]json.RawMessage
				_ = json.Unmarshal(root["mcp"], &outer)
				outer["servers"], _ = json.Marshal(map[string]json.RawMessage{name: server})
				root["mcp"], _ = json.Marshal(outer)
			} else {
				root[wrapper], _ = json.Marshal(map[string]json.RawMessage{name: server})
			}
			_ = serverName
		}
		out, err := json.Marshal(root)
		return string(out), err
	}
	return raw, nil
}

// Installation detection only applies to default targets, never custom profiles.
func installedAgent(key string, p model.ProfileConfig) bool {
	return store.IsInstalledAgent(key, p)
}
func safeText(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)
}
func showPlan(w io.Writer, p install.Plan, targets []string) {
	outputHeading(w, "MCP / "+p.Name)
	outputParagraph(w, p.Summary)
	fmt.Fprintf(w, "\nAgents: %s\n", safeText(strings.Join(targets, ", ")))
	fmt.Fprintf(w, "Plan ID: %s\n", p.Digest())
	if len(p.Steps) > 0 {
		outputHeading(w, "INSTALLATION STEPS")
	}
	for i, step := range p.Steps {
		outputParagraph(w, fmt.Sprintf("%d. %s", i+1, step.Description))
		args, _ := json.Marshal(step.Args)
		outputParagraph(w, "   Command: "+step.Command+" "+string(args))
		outputParagraph(w, "   Directory: "+step.Directory)
		fmt.Fprintln(w)
	}
	if len(p.Requirements) > 0 {
		outputHeading(w, "REQUIRED SOFTWARE")
		for _, req := range p.Requirements {
			outputParagraph(w, "  - "+req)
		}
	}
	items, err := model.Import(p.Connection, p.Name, map[string]string{"INSTALL_DIR": "/installation"})
	if err == nil {
		outputHeading(w, "CONNECTION")
		fmt.Fprintf(w, "Transport: %s\n", items[0].Kind)
		if len(items[0].Required) > 0 {
			outputParagraph(w, "Private inputs: "+strings.Join(items[0].Required, ", "))
		} else {
			fmt.Fprintln(w, "No private inputs required.")
		}
	}
	if len(p.Manual) > 0 {
		outputHeading(w, "BEFORE INSTALLATION")
	}
	for _, manual := range p.Manual {
		outputParagraph(w, "Manual step: "+manual)
		fmt.Fprintln(w)
	}
	if len(p.FollowUp) > 0 {
		outputHeading(w, "NEXT STEPS")
	}
	for _, action := range p.FollowUp {
		outputParagraph(w, "After installation: "+action)
		fmt.Fprintln(w)
	}
	if len(p.Sources) > 0 {
		outputHeading(w, "SOURCES")
	}
	for _, source := range p.Sources {
		outputParagraph(w, "  "+source)
	}
	fmt.Fprintln(w)
}
