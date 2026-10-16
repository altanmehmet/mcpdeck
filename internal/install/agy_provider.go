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
	"regexp"
	"runtime"
	"strings"

	"github.com/altanmehmet/mcpdeck/internal/process"
)

type PlannerModel struct{ ID, Label string }

var plannerModelID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/+-]{0,255}$`)

// agyEnvironment isolates permissions, MCPs, hooks and instruction files. Only
// the cached login is copied privately; the original configuration is never edited.
func agyEnvironment(dir string) ([]string, error) {
	home := filepath.Join(dir, "home")
	data := filepath.Join(home, ".gemini", "antigravity-cli")
	if err := os.MkdirAll(data, 0700); err != nil {
		return nil, err
	}
	settings := map[string]any{"toolPermission": "request-review", "permissions": map[string]any{
		"deny":  []string{"command(*)", "unsandboxed(*)", "read_file(*)", "write_file(*)", "mcp(*)", "execute_url(*)"},
		"allow": []string{"read_url(*)"},
	}}
	raw, _ := json.Marshal(settings)
	if err := os.WriteFile(filepath.Join(data, "settings.json"), raw, 0600); err != nil {
		return nil, err
	}
	original, _ := os.UserHomeDir()
	token := filepath.Join(original, ".gemini", "antigravity-cli", "antigravity-oauth-token")
	if info, err := os.Lstat(token); err == nil {
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("agy login cache must be a regular file")
		}
		f, err := os.Open(token)
		if err != nil {
			return nil, fmt.Errorf("cannot read agy login cache")
		}
		raw, err := io.ReadAll(io.LimitReader(f, 1024*1024+1))
		f.Close()
		if err != nil || len(raw) > 1024*1024 {
			return nil, fmt.Errorf("cannot read bounded agy login cache")
		}
		if err := os.WriteFile(filepath.Join(data, "antigravity-oauth-token"), raw, 0600); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("cannot inspect agy login cache")
	}
	env := plannerEnvironment()
	filtered := env[:0]
	for _, v := range env {
		if !strings.HasPrefix(v, "HOME=") && !(runtime.GOOS == "windows" && (strings.HasPrefix(v, "USERPROFILE=") || strings.HasPrefix(v, "APPDATA=") || strings.HasPrefix(v, "LOCALAPPDATA="))) {
			filtered = append(filtered, v)
		}
	}
	env = append(filtered, "HOME="+home, "AGY_CLI_DISABLE_AUTO_UPDATE=true")
	if runtime.GOOS == "windows" {
		env = append(env, "USERPROFILE="+home, "APPDATA="+filepath.Join(home, "AppData", "Roaming"), "LOCALAPPDATA="+filepath.Join(home, "AppData", "Local"))
	}
	return env, nil
}

func agyCommand(ctx context.Context, executable, dir string, args ...string) (*exec.Cmd, error) {
	env, err := agyEnvironment(dir)
	if err != nil {
		return nil, err
	}
	args = append([]string{"--log-file", filepath.Join(dir, "agy.log")}, args...)
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stderr = io.Discard
	return cmd, nil
}

// Models come from the installed CLI, not a static list of vendor model names.
func AgyModels(ctx context.Context, executable string) ([]PlannerModel, error) {
	if executable == "" {
		var err error
		executable, err = PlannerPath("agy")
		if err != nil {
			return nil, err
		}
	}
	dir, err := os.MkdirTemp("", "mcpdeck-agy-models-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	cmd, err := agyCommand(ctx, executable, dir, "models")
	if err != nil {
		return nil, err
	}
	var out boundedOutput
	cmd.Stdout = &out
	if err := process.RunSetup(cmd); err != nil {
		return nil, fmt.Errorf("agy model listing failed; check agy login and version")
	}
	var models []PlannerModel
	seen := map[string]bool{}
	for _, line := range strings.Split(out.String(), "\n") {
		columns := strings.SplitN(strings.TrimSpace(line), "\t", 2)
		if len(columns) != 2 || !plannerModelID.MatchString(columns[0]) || strings.TrimSpace(columns[1]) == "" || seen[columns[0]] {
			continue
		}
		seen[columns[0]] = true
		models = append(models, PlannerModel{columns[0], strings.TrimSpace(columns[1])})
		if len(models) > 128 {
			return nil, fmt.Errorf("agy returned too many models")
		}
	}
	if len(models) == 0 {
		return nil, fmt.Errorf("agy returned no model IDs")
	}
	return models, nil
}

func generateAgy(ctx context.Context, o PlannerOptions, executable, request string) (Plan, error) {
	dir, err := os.MkdirTemp("", "mcpdeck-agy-plan-*")
	if err != nil {
		return Plan{}, err
	}
	defer os.RemoveAll(dir)
	args := []string{"--input-format", "stream-json", "--output-format", "stream-json", "--json-schema", string(Schema()), "--mode", "plan", "--disable-slash-commands", "--print-timeout", "2m"}
	if o.Model != "" {
		args = append(args, "--model", o.Model)
	}
	cmd, err := agyCommand(ctx, executable, dir, args...)
	if err != nil {
		return Plan{}, err
	}
	message := map[string]any{"event": "user", "message": map[string]string{"content": machinePlannerPrompt(ctx, request)}}
	raw, _ := json.Marshal(message)
	cmd.Stdin = bytes.NewReader(append(raw, '\n'))
	var out boundedOutput
	cmd.Stdout = &out
	if err := process.RunSetup(cmd); err != nil {
		return Plan{}, fmt.Errorf("agy planner failed; check CLI version, login and model (no installation performed)")
	}
	return decodeAgyPlan(out.Bytes())
}

func decodeAgyPlan(raw []byte) (Plan, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	count := 0
	var plan Plan
	for {
		var event struct {
			Event  string `json:"event"`
			Result struct {
				Status     string          `json:"status"`
				Structured json.RawMessage `json:"structured_output"`
				Response   string          `json:"response"`
			} `json:"result"`
		}
		err := decoder.Decode(&event)
		if err == io.EOF {
			break
		}
		if err != nil {
			return Plan{}, fmt.Errorf("agy returned invalid JSON events")
		}
		if event.Event != "result" {
			continue
		}
		count++
		if count != 1 || event.Result.Status != "SUCCESS" {
			return Plan{}, fmt.Errorf("agy did not return one successful planning result")
		}
		if len(event.Result.Structured) > 0 && string(event.Result.Structured) != "null" {
			plan, err = decodeAgyStructured(event.Result.Structured)
		} else {
			plan, err = decodePlanText(event.Result.Response)
		}
		if err != nil {
			return Plan{}, fmt.Errorf("agy returned invalid installation plan: %w", err)
		}
	}
	if count != 1 {
		return Plan{}, fmt.Errorf("agy returned no planning result")
	}
	return plan, nil
}

// Some agy versions serialize structured_output as a JSON string even when a
// schema was supplied. Accept both that form and the object form used by
// newer versions.
func decodeAgyStructured(raw []byte) (Plan, error) {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return decodePlanText(text)
	}
	return Decode(raw)
}
