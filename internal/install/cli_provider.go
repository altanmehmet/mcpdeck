package install

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/altanmehmet/mcpdeck/internal/process"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type boundedOutput struct{ bytes.Buffer }

func (b *boundedOutput) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 2*1024*1024 {
		return 0, fmt.Errorf("planner output exceeds limit")
	}
	return b.Buffer.Write(p)
}

func generateOtherCLI(ctx context.Context, o PlannerOptions, executable, request string) (Plan, error) {
	dir, err := os.MkdirTemp("", "mcpdeck-planner-*")
	if err != nil {
		return Plan{}, err
	}
	defer os.RemoveAll(dir)
	args := []string{}
	env := plannerEnvironment()
	switch o.Provider {
	case "claude":
		args = []string{"-p", "--output-format", "json", "--json-schema", string(Schema()), "--tools", "WebSearch,WebFetch", "--allowedTools", "WebSearch,WebFetch", "--strict-mcp-config", "--mcp-config", `{"mcpServers":{}}`, "--setting-sources", "", "--no-session-persistence", "--max-turns", "12"}
		env = plannerEnvironment("ANTHROPIC_API_KEY", "CLAUDE_CODE_OAUTH_TOKEN")
	case "gemini":
		// Do not use headless plan mode: it may transition to automatic execution.
		// Project tool allowlist and a policy deny non-web tools, including MCPs.
		policy := filepath.Join(dir, "planning-policy.toml")
		rules := "[[rule]]\ntoolName = \"*\"\ndecision = \"deny\"\npriority = 998\n\n[[rule]]\ntoolName = [\"google_web_search\", \"web_fetch\"]\ndecision = \"allow\"\npriority = 999\n"
		if err = os.WriteFile(policy, []byte(rules), 0600); err != nil {
			return Plan{}, err
		}
		settings := map[string]any{"tools": map[string]any{"core": []string{"google_web_search", "web_fetch"}}, "mcp": map[string]any{"allowed": []string{"mcpdeck-no-mcp-servers"}}, "hooksConfig": map[string]any{"enabled": false}, "general": map[string]any{"enableAutoUpdate": false}, "policyPaths": []string{policy}}
		raw, _ := json.Marshal(settings)
		if err = os.Mkdir(filepath.Join(dir, ".gemini"), 0700); err != nil {
			return Plan{}, err
		}
		if err = os.WriteFile(filepath.Join(dir, ".gemini", "settings.json"), raw, 0600); err != nil {
			return Plan{}, err
		}
		args = []string{"-p", "Return the installation recipe requested on stdin.", "--output-format", "json", "--approval-mode", "default", "--admin-policy", policy, "--extensions", "none", "--allowed-mcp-server-names", "mcpdeck-no-mcp-servers"}
		env = plannerEnvironment("GEMINI_API_KEY", "GOOGLE_API_KEY")
	}
	if o.Model != "" {
		args = append(args, "--model", o.Model)
	}
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdin = strings.NewReader(machinePlannerPrompt(ctx, request) + "\nReturn only JSON matching this schema:\n" + string(Schema()))
	var output boundedOutput
	cmd.Stdout = &output
	cmd.Stderr = io.Discard
	if err = process.RunSetup(cmd); err != nil {
		return Plan{}, fmt.Errorf("%s planner failed; check CLI version, login and model (no installation performed)", o.Provider)
	}
	var result struct {
		Structured json.RawMessage `json:"structured_output"`
		Result     string          `json:"result"`
		Response   string          `json:"response"`
		IsError    bool            `json:"is_error"`
		Error      json.RawMessage `json:"error"`
	}
	if json.Unmarshal(output.Bytes(), &result) != nil || result.IsError || (len(result.Error) > 0 && string(result.Error) != "null") {
		return Plan{}, fmt.Errorf("%s planner returned an error or invalid response", o.Provider)
	}
	if o.Provider == "claude" {
		if len(result.Structured) > 0 && string(result.Structured) != "null" {
			return Decode(result.Structured)
		}
		return decodePlanText(result.Result)
	}
	return decodePlanText(result.Response)
}
