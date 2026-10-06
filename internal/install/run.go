package install

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/process"
)

func (p Plan) Preflight() error {
	return p.PreflightAfterManualReview(false)
}

// PreflightAfterManualReview keeps manual prerequisites blocking by default.
// Interactive callers may proceed only after explicitly confirming that each
// listed item was completed or independently verified.
func (p Plan) PreflightAfterManualReview(manualReviewed bool) error {
	if err := p.Validate(); err != nil {
		return err
	}
	if p.HasPlaceholderConnection() {
		return fmt.Errorf("this plan uses a placeholder MCP connection and cannot be installed yet; complete the manual actions and create a new plan")
	}
	if err := p.CheckRequirements(); err != nil {
		return err
	}
	if len(p.Manual) > 0 && !manualReviewed {
		return fmt.Errorf("manual prerequisites need review before installation")
	}
	return nil
}

// HasPlaceholderConnection reports plans that intentionally use `false` when
// documentation or prerequisites were insufficient to describe a runnable MCP.
func (p Plan) HasPlaceholderConnection() bool {
	items, err := model.Import(p.Connection, p.Name, map[string]string{"INSTALL_DIR": "/installation"})
	return err == nil && len(items) == 1 && items[0].Server.Command == "false"
}

// CheckRequirements checks only documented executable prerequisites. It does
// not approve or execute any installation step.
func (p Plan) CheckRequirements() error {
	for _, name := range p.Requirements {
		executable, err := lookPathExecutable(name)
		if err != nil {
			message := fmt.Sprintf("required executable %q is not available on PATH; install it, then reopen MCPDeck", name)
			if runtime.GOOS == "darwin" {
				if formula, ok := homebrewFormula(name); ok {
					message += fmt.Sprintf(" (Homebrew: brew install %s)", formula)
				}
			}
			return fmt.Errorf("%s", message)
		}
		if name == "docker" {
			if err := checkDockerRuntime(executable); err != nil {
				return err
			}
		}
	}
	return nil
}

// MissingHomebrewRequirements returns supported macOS prerequisites that can
// be installed without administrator privileges through Homebrew.
func MissingHomebrewRequirements(requirements []string) (formulas, unsupported []string) {
	if runtime.GOOS != "darwin" {
		return nil, append([]string(nil), requirements...)
	}
	formulaSet := map[string]bool{}
	unsupportedSet := map[string]bool{}
	for _, requirement := range requirements {
		if _, err := lookPathExecutable(requirement); err == nil {
			continue
		}
		formula, ok := homebrewFormula(requirement)
		if !ok {
			unsupportedSet[requirement] = true
			continue
		}
		formulaSet[formula] = true
		// Docker CLI alone is not a working local engine. Colima provides a
		// user-level Docker runtime which MCPDeck can start after consent.
		if requirement == "docker" {
			formulaSet["colima"] = true
		}
	}
	for formula := range formulaSet {
		formulas = append(formulas, formula)
	}
	for requirement := range unsupportedSet {
		unsupported = append(unsupported, requirement)
	}
	sort.Strings(formulas)
	sort.Strings(unsupported)
	return formulas, unsupported
}

// InstallHomebrewPackages installs only named, allowlisted formulae. The caller
// must obtain explicit user approval before invoking it.
func InstallHomebrewPackages(ctx context.Context, formulas []string) error {
	allowed := map[string]bool{"node": true, "python": true, "uv": true, "docker": true, "colima": true, "bun": true, "pnpm": true, "yarn": true, "go": true, "rust": true, "openjdk": true, "maven": true, "git": true}
	for _, formula := range formulas {
		if !allowed[formula] {
			return fmt.Errorf("unsupported Homebrew package %q", safeDockerContext(formula))
		}
	}
	brew, err := homebrewExecutable()
	if err != nil {
		return fmt.Errorf("Homebrew is not installed; install Homebrew, then retry MCPDeck")
	}
	if len(formulas) == 0 {
		return nil
	}
	args := append([]string{"install"}, formulas...)
	childCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()
	command := exec.CommandContext(childCtx, brew, args...)
	command.Env = plannerEnvironment("HOMEBREW_PREFIX", "HOMEBREW_CELLAR", "HOMEBREW_REPOSITORY")
	var stdout, stderr limitedBuffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := process.RunSetup(command); err != nil {
		detail := redactPlannerDiagnostics(stderr.String() + " " + stdout.String())
		if detail == "" {
			return fmt.Errorf("Homebrew could not install the required runtime packages (%v)", err)
		}
		return fmt.Errorf("Homebrew could not install the required runtime packages: %s", detail)
	}
	return nil
}

func homebrewExecutable() (string, error) {
	if path, err := lookPathExecutable("brew"); err == nil {
		return path, nil
	}
	for _, path := range []string{"/opt/homebrew/bin/brew", "/usr/local/bin/brew"} {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, nil
		}
	}
	return "", fmt.Errorf("brew not found")
}

type DockerRuntimeUnavailableError struct{ Context string }

func (e *DockerRuntimeUnavailableError) Error() string {
	if e.Context == "colima" {
		return "Docker is installed, but the selected Colima runtime is stopped or unreachable"
	}
	if e.Context != "" {
		return fmt.Sprintf("Docker is installed, but its engine is unavailable for context %q", safeDockerContext(e.Context))
	}
	return "Docker is installed, but MCPDeck cannot reach a running Docker engine"
}

func checkDockerRuntime(executable string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	check := exec.CommandContext(ctx, executable, "info", "--format", "{{.ServerVersion}}")
	check.Stdout = io.Discard
	check.Stderr = io.Discard
	if err := check.Run(); err == nil {
		return nil
	}
	contextCmd := exec.CommandContext(ctx, executable, "context", "show")
	var output bytes.Buffer
	contextCmd.Stdout = &output
	contextCmd.Stderr = io.Discard
	_ = contextCmd.Run()
	selected := strings.TrimSpace(output.String())
	return &DockerRuntimeUnavailableError{Context: selected}
}

// StartColima starts the selected local Colima runtime after the CLI has asked
// the user for permission. It never pulls MCP images or edits agent configs.
func StartColima(ctx context.Context) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("automatic Colima startup is supported only on macOS; start the runtime manually")
	}
	executable, err := lookPathExecutable("colima")
	if err != nil {
		return fmt.Errorf("Colima CLI is unavailable; install Colima, then retry New MCP")
	}
	childCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	command := exec.CommandContext(childCtx, executable, "start")
	command.Env = plannerEnvironment("DOCKER_CONFIG", "DOCKER_CONTEXT", "DOCKER_HOST", "COLIMA_HOME", "COLIMA_PROFILE")
	command.Stdin = os.Stdin
	var stdout, stderr limitedBuffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := process.RunSetup(command); err != nil {
		detail := redactPlannerDiagnostics(stderr.String() + " " + stdout.String())
		if detail == "" {
			return fmt.Errorf("Colima could not be started (%v); no MCP was installed", err)
		}
		return fmt.Errorf("Colima could not be started: %s; no MCP was installed", detail)
	}
	docker, err := lookPathExecutable("docker")
	if err != nil {
		return fmt.Errorf("Colima started, but the Docker CLI is no longer available on PATH")
	}
	if err := checkDockerRuntime(docker); err != nil {
		return fmt.Errorf("Colima started, but Docker is not ready yet: %w", err)
	}
	return nil
}

func ColimaAvailable() bool {
	_, err := lookPathExecutable("colima")
	return err == nil
}

func lookPathExecutable(name string) (string, error) {
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}
	if runtime.GOOS == "darwin" && filepath.Base(name) == name {
		for _, path := range []string{filepath.Join("/opt/homebrew/bin", name), filepath.Join("/usr/local/bin", name)} {
			if info, err := os.Stat(path); err == nil && !info.IsDir() && info.Mode()&0111 != 0 {
				return path, nil
			}
		}
	}
	return "", fmt.Errorf("executable not found")
}

func safeDockerContext(value string) string {
	for _, r := range value {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("._-", r)) {
			return "configured"
		}
	}
	return value
}

func homebrewFormula(executable string) (string, bool) {
	switch executable {
	case "node", "npm", "npx":
		return "node", true
	case "python", "python3", "pip", "pip3":
		return "python", true
	case "uv", "uvx":
		return "uv", true
	case "docker":
		return "docker", true
	case "colima":
		return "colima", true
	case "bun":
		return "bun", true
	case "pnpm":
		return "pnpm", true
	case "yarn":
		return "yarn", true
	case "go":
		return "go", true
	case "cargo", "rustc":
		return "rust", true
	case "java":
		return "openjdk", true
	case "mvn":
		return "maven", true
	case "git":
		return "git", true
	default:
		return "", false
	}
}

// Run executes the exact approved recipe. It is not an OS sandbox: downloaded
// programs have the user's permissions. Bounded diagnostics are redacted before display.
func (p Plan) Run(ctx context.Context, dir, approved string, progress io.Writer) error {
	return p.RunAfterManualReview(ctx, dir, approved, progress, false)
}

// RunAfterManualReview executes only the approved plan and allows manual steps
// solely when the caller has collected an explicit confirmation.
func (p Plan) RunAfterManualReview(ctx context.Context, dir, approved string, progress io.Writer, manualReviewed bool) error {
	if approved != p.Digest() {
		return fmt.Errorf("plan approval does not match; review the plan again")
	}
	if err := p.PreflightAfterManualReview(manualReviewed); err != nil {
		return err
	}
	if !filepath.IsAbs(dir) {
		return fmt.Errorf("installation directory must be absolute")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	root, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return err
	}
	for i, s := range p.Steps {
		cwd := filepath.Join(root, s.Directory)
		real, err := filepath.EvalSymlinks(cwd)
		if err != nil {
			return fmt.Errorf("step %d working directory does not exist", i+1)
		}
		rel, err := filepath.Rel(root, real)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return fmt.Errorf("step %d escapes installation directory", i+1)
		}
		args := make([]string, len(s.Args))
		for j, a := range s.Args {
			args[j] = strings.ReplaceAll(a, "${INSTALL_DIR}", root)
		}
		executable := strings.ReplaceAll(s.Command, "${INSTALL_DIR}", root)
		fmt.Fprintf(progress, "Installation step %d/%d: %s\n", i+1, len(p.Steps), filepath.Base(executable))
		childCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		command := exec.CommandContext(childCtx, executable, args...)
		command.Dir = real
		// Setup only receives ordinary runtime environment, not MCP input values.
		for _, k := range []string{"PATH", "HOME", "TMPDIR", "SYSTEMROOT"} {
			if v, ok := os.LookupEnv(k); ok {
				command.Env = append(command.Env, k+"="+v)
			}
		}
		var stdout, stderr limitedBuffer
		command.Stdout = &stdout
		command.Stderr = &stderr
		err = process.RunSetup(command)
		cancel()
		if err != nil {
			detail := redactPlannerDiagnostics(stderr.String() + " " + stdout.String())
			if detail != "" {
				return fmt.Errorf("installation step %d failed: %s; agent configurations were not changed", i+1, detail)
			}
			return fmt.Errorf("installation step %d failed (%v); agent configurations were not changed", i+1, err)
		}
	}
	return nil
}

// Probe initializes a real MCP session and checks all tools/list pages. It never calls a tool.
func Probe(ctx context.Context, name string, cfg model.ServerConfig) (int, error) {
	m := process.New(&model.Deck{Servers: map[string]model.ServerConfig{name: cfg}}, time.Minute)
	defer m.KillAll()
	count := 0
	cursor := ""
	seen := map[string]bool{}
	names := map[string]bool{}
	for page := 0; page < 100; page++ {
		params, _ := json.Marshal(map[string]string{"cursor": cursor})
		if cursor == "" {
			params = []byte(`{}`)
		}
		raw, err := m.Request(ctx, name, "tools/list", params)
		if err != nil {
			return 0, fmt.Errorf("MCP connection verification failed; check runtime, credentials or OAuth (agent configurations unchanged)")
		}
		var result struct {
			Tools *[]struct {
				Name        string         `json:"name"`
				InputSchema map[string]any `json:"inputSchema"`
			} `json:"tools"`
			NextCursor string `json:"nextCursor"`
		}
		if json.Unmarshal(raw, &result) != nil || result.Tools == nil {
			return 0, fmt.Errorf("invalid MCP tools/list response")
		}
		for _, tool := range *result.Tools {
			if tool.Name == "" || names[tool.Name] || tool.InputSchema["type"] != "object" {
				return 0, fmt.Errorf("invalid or duplicate MCP tool")
			}
			names[tool.Name] = true
			count++
		}
		cursor = result.NextCursor
		if cursor == "" {
			return count, nil
		}
		if seen[cursor] {
			return 0, fmt.Errorf("MCP returned a repeated tools cursor")
		}
		seen[cursor] = true
	}
	return 0, fmt.Errorf("MCP tool pagination exceeded limit")
}
