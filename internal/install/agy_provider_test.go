package install

import (
	"context"
	"encoding/json"
	"github.com/altanmehmet/mcpdeck/internal/testutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestAgyEnvironmentIsolatesRulesAndCopiesOnlyLogin(t *testing.T) {
	original := t.TempDir()
	testutil.SetHome(t, original)
	data := filepath.Join(original, ".gemini", "antigravity-cli")
	if err := os.MkdirAll(data, 0700); err != nil {
		t.Fatal(err)
	}
	for name, text := range map[string]string{"antigravity-oauth-token": "fixture-cached-login", "mcp_config.json": "private MCP", "settings.json": "original-settings"} {
		if err := os.WriteFile(filepath.Join(data, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	env, err := agyEnvironment(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var home string
	for _, value := range env {
		if strings.HasPrefix(value, "HOME=") {
			home = strings.TrimPrefix(value, "HOME=")
		}
	}
	if home == "" || home == original {
		t.Fatal("home not isolated")
	}
	copyData := filepath.Join(home, ".gemini", "antigravity-cli")
	raw, err := os.ReadFile(filepath.Join(copyData, "antigravity-oauth-token"))
	if err != nil || string(raw) != "fixture-cached-login" {
		t.Fatal("login copy absent", err)
	}
	if _, err := os.Stat(filepath.Join(copyData, "mcp_config.json")); !os.IsNotExist(err) {
		t.Fatal("user MCPs copied")
	}
	raw, err = os.ReadFile(filepath.Join(copyData, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Permissions struct {
			Deny []string `json:"deny"`
		} `json:"permissions"`
	}
	if json.Unmarshal(raw, &cfg) != nil {
		t.Fatal("invalid settings")
	}
	for _, blocked := range []string{"command(*)", "write_file(*)", "mcp(*)"} {
		if !strings.Contains(strings.Join(cfg.Permissions.Deny, "\n"), blocked) {
			t.Fatal("permission guard missing", blocked)
		}
	}
	raw, _ = os.ReadFile(filepath.Join(data, "settings.json"))
	if string(raw) != "original-settings" {
		t.Fatal("original settings modified")
	}
}

func TestDecodeAgyPlanningResult(t *testing.T) {
	raw, _ := json.Marshal(recipe())
	for _, field := range []string{"structured_output", "response"} {
		result := map[string]any{"status": "SUCCESS"}
		if field == "response" {
			result[field] = string(raw)
		} else {
			result[field] = json.RawMessage(raw)
		}
		event, _ := json.Marshal(map[string]any{"event": "result", "result": result})
		p, err := decodeAgyPlan(append([]byte("{\"event\":\"init\"}\n"), event...))
		if err != nil || p.Name != "demo" {
			t.Fatal(p, err)
		}
		if _, err := decodeAgyPlan(append(append(event, '\n'), event...)); err == nil {
			t.Fatal("duplicate results accepted")
		}
	}
	stringResult, _ := json.Marshal(map[string]any{
		"event": "result",
		"result": map[string]any{
			"status":            "SUCCESS",
			"structured_output": string(raw),
		},
	})
	if p, err := decodeAgyPlan(stringResult); err != nil || p.Name != "demo" {
		t.Fatal(p, err)
	}
	for _, status := range []string{"ERROR", "WAITING", "CANCELED"} {
		event, _ := json.Marshal(map[string]any{"event": "result", "result": map[string]any{"status": status, "response": string(raw)}})
		if _, err := decodeAgyPlan(event); err == nil {
			t.Fatal("non-success accepted", status)
		}
	}
	if _, err := decodeAgyPlan([]byte(`{"event":"init"}`)); err == nil {
		t.Fatal("missing result accepted")
	}
	if _, err := decodeAgyPlan([]byte("not-json")); err == nil {
		t.Fatal("invalid output accepted")
	}
}

func TestAgyCLIProtocolAndModelCatalog(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	t.Setenv("MCPDECK_INSTALL_PASSWORD", "fixture-private")
	raw, _ := json.Marshal(recipe())
	event, _ := json.Marshal(map[string]any{"event": "result", "result": map[string]any{"status": "SUCCESS", "structured_output": json.RawMessage(raw)}})
	makeFixture := func(payload string) string {
		if runtime.GOOS == "windows" {
			return testutil.PlannerExecutable(t, payload, "", 0, "", false)
		}
		path := filepath.Join(t.TempDir(), "agy-fixture")
		script := "#!/bin/sh\nif [ -n \"$MCPDECK_INSTALL_PASSWORD\" ]; then exit 4; fi\ncat >/dev/null\ncat <<'RESULT'\n" + payload + "\nRESULT\n"
		if err := os.WriteFile(path, []byte(script), 0700); err != nil {
			t.Fatal(err)
		}
		return path
	}
	executable := makeFixture(string(event))
	plan, err := PlanWith(context.Background(), PlannerOptions{Provider: "agy", Executable: executable, Model: "fixture-model"}, "fixture request")
	if err != nil || plan.Name != "demo" {
		t.Fatal(plan, err)
	}
	executable = makeFixture("fixture-model\tFixture model\ninvalid model\tRejected\nfixture-model\tDuplicate\n")
	models, err := AgyModels(context.Background(), executable)
	if err != nil || len(models) != 1 || models[0].ID != "fixture-model" {
		t.Fatal(models, err)
	}
}

func TestLiveAgyModelsOptIn(t *testing.T) {
	if os.Getenv("MCPDECK_LIVE_AGY_MODELS") != "1" {
		t.Skip("explicit local metadata check only")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	models, err := AgyModels(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("model count: %d", len(models))
}
