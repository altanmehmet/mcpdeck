package install

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// PlannerOptions contains references to credentials, never credential values.
type PlannerOptions struct {
	Provider   string `json:"provider"`
	Model      string `json:"model,omitempty"`
	Executable string `json:"executable,omitempty"`
	BaseURL    string `json:"base_url,omitempty"`
	KeyEnv     string `json:"key_env,omitempty"`
	APIKey     string `json:"-"`
}

var Providers = []string{"codex", "claude", "gemini", "openai-api", "anthropic-api", "gemini-api", "openai-compatible"}
var envName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func (o PlannerOptions) Validate() error {
	valid := false
	for _, p := range Providers {
		if o.Provider == p {
			valid = true
		}
	}
	if !valid {
		return fmt.Errorf("unknown planner provider; use %s", strings.Join(Providers, ", "))
	}
	if o.KeyEnv != "" && !envName.MatchString(o.KeyEnv) {
		return fmt.Errorf("invalid API key environment variable name")
	}
	if o.IsAPI() {
		if o.Executable != "" {
			return fmt.Errorf("API providers do not use --planner executables")
		}
		if o.Model == "" {
			return fmt.Errorf("API provider requires --model (choose a model available in your account)")
		}
		if o.Provider != "openai-compatible" && o.BaseURL != "" {
			return fmt.Errorf("custom API URLs require openai-compatible provider")
		}
		if o.Provider == "openai-compatible" {
			if _, err := compatibleEndpoint(o.BaseURL); err != nil {
				return err
			}
		}
	} else if o.BaseURL != "" || o.KeyEnv != "" {
		return fmt.Errorf("API settings cannot be used with a CLI provider")
	}
	return nil
}
func (o PlannerOptions) IsAPI() bool {
	return strings.HasSuffix(o.Provider, "-api") || o.Provider == "openai-compatible"
}
func (o PlannerOptions) CredentialEnv() string {
	if o.KeyEnv != "" {
		return o.KeyEnv
	}
	switch o.Provider {
	case "openai-api":
		return "OPENAI_API_KEY"
	case "openai-compatible":
		return "MCPDECK_API_KEY"
	case "anthropic-api":
		return "ANTHROPIC_API_KEY"
	case "gemini-api":
		return "GEMINI_API_KEY"
	}
	return ""
}
func PlanWith(ctx context.Context, o PlannerOptions, request string) (Plan, error) {
	if o.Provider == "" {
		o.Provider = "codex"
	}
	if err := o.Validate(); err != nil {
		return Plan{}, err
	}
	if len(request) > 128*1024 {
		return Plan{}, fmt.Errorf("request exceeds 128 KiB")
	}
	if o.IsAPI() {
		return generateAPI(ctx, o, request)
	}
	executable := o.Executable
	if executable == "" {
		var err error
		if o.Provider == "codex" {
			executable, err = CodexPath()
		} else {
			executable, err = exec.LookPath(o.Provider)
		}
		if err != nil {
			return Plan{}, fmt.Errorf("%s CLI not found; install and sign in, or select an API provider", o.Provider)
		}
	}
	if o.Provider == "codex" {
		return generateCodex(ctx, executable, o.Model, request)
	}
	return generateOtherCLI(ctx, o, executable, request)
}

func plannerEnvironment(extra ...string) []string {
	keys := append([]string{"PATH", "HOME", "TMPDIR", "SYSTEMROOT", "USERPROFILE", "APPDATA", "LOCALAPPDATA", "TEMP", "TMP", "PATHEXT"}, extra...)
	env := []string{}
	for _, k := range keys {
		if v, ok := os.LookupEnv(k); ok {
			env = append(env, k+"="+v)
		}
	}
	return env
}

func decodePlanText(text string) (Plan, error) {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```json\n") {
		text = strings.TrimPrefix(text, "```json\n")
		text = strings.TrimSuffix(text, "```")
	}
	if strings.HasPrefix(text, "```\n") {
		text = strings.TrimPrefix(text, "```\n")
		text = strings.TrimSuffix(text, "```")
	}
	return Decode([]byte(strings.TrimSpace(text)))
}
