package install

import (
	"context"
	"encoding/json"
	"github.com/altanmehmet/mcpdeck/internal/testutil"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func apiFixture(provider string, p Plan) []byte {
	raw, _ := json.Marshal(p)
	var out any
	switch provider {
	case "openai-api":
		out = map[string]any{"status": "completed", "output": []any{map[string]any{"type": "message", "content": []any{map[string]any{"type": "output_text", "text": string(raw)}}}}}
	case "anthropic-api":
		out = map[string]any{"stop_reason": "end_turn", "content": []any{map[string]any{"type": "text", "text": string(raw)}}}
	case "gemini-api":
		out = map[string]any{"candidates": []any{map[string]any{"finishReason": "STOP", "content": map[string]any{"parts": []any{map[string]any{"text": "```json\n" + string(raw) + "\n```"}}}}}}
	default:
		out = map[string]any{"choices": []any{map[string]any{"finish_reason": "stop", "message": map[string]any{"content": string(raw)}}}}
	}
	result, _ := json.Marshal(out)
	return result
}
func TestAPIProvidersWireAndSecretIsolation(t *testing.T) {
	for _, provider := range []string{"openai-api", "anthropic-api", "gemini-api", "openai-compatible"} {
		t.Run(provider, func(t *testing.T) {
			o := PlannerOptions{Provider: provider, Model: "fixture-model", APIKey: "fixture-secret"}
			if provider == "openai-compatible" {
				o.BaseURL = "https://example.com/v1"
			}
			t.Setenv("UNRELATED_MCP_PASSWORD", "not-in-prompt")
			client := &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
				raw, _ := io.ReadAll(r.Body)
				if strings.Contains(string(raw), "fixture-secret") || strings.Contains(string(raw), "not-in-prompt") {
					t.Fatal("secret in prompt")
				}
				if strings.Contains(r.URL.String(), "fixture-secret") {
					t.Fatal("key in URL")
				}
				var body map[string]any
				if json.Unmarshal(raw, &body) != nil {
					t.Fatal("invalid body")
				}
				switch provider {
				case "openai-api":
					if r.URL.Path != "/v1/responses" || r.Header.Get("Authorization") != "Bearer fixture-secret" || body["store"] != false || body["tools"] == nil {
						t.Fatal("bad OpenAI request")
					}
				case "anthropic-api":
					if r.URL.Path != "/v1/messages" || r.Header.Get("x-api-key") != "fixture-secret" || r.Header.Get("anthropic-version") == "" || body["tools"] == nil {
						t.Fatal("bad Anthropic request")
					}
				case "gemini-api":
					if !strings.HasSuffix(r.URL.Path, "/fixture-model:generateContent") || r.Header.Get("x-goog-api-key") != "fixture-secret" || body["tools"] == nil {
						t.Fatal("bad Gemini request")
					}
				case "openai-compatible":
					if r.URL.Path != "/v1/chat/completions" || !strings.Contains(string(raw), "NO web browsing tools") {
						t.Fatal("bad compatible request")
					}
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(apiFixture(provider, recipe())))), Header: make(http.Header)}, nil
			})}
			plan, err := generateAPIWithClient(context.Background(), o, "public docs", client)
			if err != nil || plan.Name != "demo" {
				t.Fatal(plan, err)
			}
			saved, _ := json.Marshal(o)
			if strings.Contains(string(saved), "fixture-secret") {
				t.Fatal("API key persisted")
			}
		})
	}
}
func TestAPIRejectsFailuresAndRedirects(t *testing.T) {
	o := PlannerOptions{Provider: "openai-compatible", Model: "fixture", APIKey: "private-test-key"}
	hit := false
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hit = true }))
	defer destination.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, destination.URL, http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	o.BaseURL = server.URL
	if _, err := generateAPI(context.Background(), o, "docs"); err == nil {
		t.Fatal("redirect accepted")
	}
	if hit {
		t.Fatal("redirect leaked credentials")
	}
	for _, status := range []int{401, 429, 500} {
		client := &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader("private-test-key"))}, nil
		})}
		_, err := generateAPIWithClient(context.Background(), o, "docs", client)
		if err == nil || strings.Contains(err.Error(), "private-test-key") {
			t.Fatal("error body leaked", err)
		}
	}
	for _, provider := range []string{"openai-api", "anthropic-api", "gemini-api", "openai-compatible"} {
		if _, err := apiText(provider, []byte(`{"error":{"message":"private-test-key"}}`)); err == nil || strings.Contains(err.Error(), "private-test-key") {
			t.Fatal("bad error handling")
		}
	}
	if _, err := apiText("openai-api", []byte(`{"status":"incomplete"}`)); err == nil {
		t.Fatal("incomplete accepted")
	}
}
func TestPlannerValidation(t *testing.T) {
	for _, o := range []PlannerOptions{{Provider: "unknown"}, {Provider: "openai-api"}, {Provider: "claude", BaseURL: "https://example.com"}, {Provider: "openai-api", Model: "x", BaseURL: "https://example.com"}, {Provider: "gemini-api", Model: "x", KeyEnv: "BAD=KEY"}} {
		if o.Validate() == nil {
			t.Fatal("bad settings accepted", o)
		}
	}
	for _, base := range []string{"http://example.com/v1", "https://user:password@example.com/v1", "https://example.com/v1?key=secret", "https://example.com/v1#secret", ""} {
		if _, err := compatibleEndpoint(base); err == nil {
			t.Fatal("bad endpoint", base)
		}
	}
	for _, base := range []string{"http://127.0.0.1:11434/v1", "http://[::1]:8000/v1", "https://example.com/v1"} {
		if _, err := compatibleEndpoint(base); err != nil {
			t.Fatal(err)
		}
	}
}
func TestClaudeAndGeminiCLIProtocol(t *testing.T) {
	for _, provider := range []string{"claude", "gemini"} {
		t.Run(provider, func(t *testing.T) {
			dir := t.TempDir()
			exe := filepath.Join(dir, "planner-fixture")
			raw, _ := json.Marshal(recipe())
			var envelope []byte
			if provider == "claude" {
				envelope, _ = json.Marshal(map[string]any{"structured_output": json.RawMessage(raw), "is_error": false})
			} else {
				envelope, _ = json.Marshal(map[string]string{"response": string(raw)})
			}
			script := `#!/bin/sh
if [ -n "$MCPDECK_INSTALL_PASSWORD" ]; then exit 4; fi
case " $* " in *" --model fixture-model "*) ;; *) exit 5 ;; esac
cat >/dev/null
cat <<'RESULT'
` + string(envelope) + "\nRESULT\n"
			if err := os.WriteFile(exe, []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			if runtime.GOOS == "windows" {
				exe = testutil.PlannerExecutable(t, string(envelope), "", 0, "", false)
			}
			t.Setenv("MCPDECK_INSTALL_PASSWORD", "private")
			plan, err := PlanWith(context.Background(), PlannerOptions{Provider: provider, Executable: exe, Model: "fixture-model"}, "docs")
			if err != nil || plan.Name != "demo" {
				t.Fatal(plan, err)
			}
		})
	}
}
