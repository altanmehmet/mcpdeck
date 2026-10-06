package install

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

func compatibleEndpoint(base string) (string, error) {
	u, err := url.Parse(base)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("provide a base URL without credentials, query or fragment")
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && loopback(u.Hostname())) {
		return "", fmt.Errorf("API base URL must use HTTPS (HTTP is allowed only on loopback)")
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/chat/completions"
	return u.String(), nil
}
func loopback(host string) bool {
	ip := net.ParseIP(host)
	return host == "localhost" || (ip != nil && ip.IsLoopback())
}

func apiRequest(ctx context.Context, o PlannerOptions, request string) (*http.Request, error) {
	key := o.APIKey
	if key == "" {
		key = os.Getenv(o.CredentialEnv())
	}
	endpoint := ""
	var body map[string]any
	prompt := machinePlannerPrompt(ctx, request) + "\nReturn only JSON matching this schema:\n" + string(Schema())
	switch o.Provider {
	case "openai-api":
		endpoint = "https://api.openai.com/v1/responses"
		body = map[string]any{"model": o.Model, "input": prompt, "store": false, "tools": []any{map[string]any{"type": "web_search"}}, "text": map[string]any{"format": map[string]any{"type": "json_schema", "name": "installation_plan", "strict": true, "schema": json.RawMessage(Schema())}}}
	case "anthropic-api":
		endpoint = "https://api.anthropic.com/v1/messages"
		body = map[string]any{"model": o.Model, "max_tokens": 8192, "messages": []any{map[string]any{"role": "user", "content": prompt}}, "tools": []any{map[string]any{"type": "web_search_20250305", "name": "web_search", "max_uses": 5}}}
	case "gemini-api":
		name := strings.TrimPrefix(o.Model, "models/")
		if !regexp.MustCompile(`^[A-Za-z0-9_.-]+$`).MatchString(name) {
			return nil, fmt.Errorf("invalid Gemini model name")
		}
		endpoint = "https://generativelanguage.googleapis.com/v1beta/models/" + name + ":generateContent"
		body = map[string]any{"contents": []any{map[string]any{"role": "user", "parts": []any{map[string]any{"text": prompt}}}}, "tools": []any{map[string]any{"google_search": map[string]any{}}, map[string]any{"url_context": map[string]any{}}}}
	case "openai-compatible":
		var err error
		endpoint, err = compatibleEndpoint(o.BaseURL)
		if err != nil {
			return nil, err
		}
		// No portable web tool exists in Chat Completions. Require supplied evidence.
		prompt += "\nThis provider has NO web browsing tools. Only use documentation text supplied in the request. If only a URL or name was given, return a blocked plan with a manual instruction to supply documentation text; do not claim to have read the URL."
		body = map[string]any{"model": o.Model, "messages": []any{map[string]any{"role": "user", "content": prompt}}}
	default:
		return nil, fmt.Errorf("unsupported API provider")
	}
	u, _ := url.Parse(endpoint)
	if key == "" && !(o.Provider == "openai-compatible" && loopback(u.Hostname())) {
		return nil, fmt.Errorf("API credential missing; set %s", o.CredentialEnv())
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("cannot encode planner request")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("invalid API request")
	}
	req.Header.Set("Content-Type", "application/json")
	switch o.Provider {
	case "anthropic-api":
		req.Header.Set("x-api-key", key)
		req.Header.Set("anthropic-version", "2023-06-01")
	case "gemini-api":
		req.Header.Set("x-goog-api-key", key)
	default:
		if key != "" {
			req.Header.Set("Authorization", "Bearer "+key)
		}
	}
	return req, nil
}

func generateAPI(ctx context.Context, o PlannerOptions, request string) (Plan, error) {
	client := &http.Client{Timeout: 150 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return fmt.Errorf("API redirects are disabled") }}
	return generateAPIWithClient(ctx, o, request, client)
}
func generateAPIWithClient(ctx context.Context, o PlannerOptions, request string, client *http.Client) (Plan, error) {
	req, err := apiRequest(ctx, o, request)
	if err != nil {
		return Plan{}, err
	}
	response, err := client.Do(req)
	if err != nil {
		return Plan{}, fmt.Errorf("%s request failed; check network, endpoint or timeout", o.Provider)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Plan{}, fmt.Errorf("%s returned HTTP %d; check credentials, model access and quota", o.Provider, response.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, 2*1024*1024+1))
	if err != nil || len(raw) > 2*1024*1024 {
		return Plan{}, fmt.Errorf("invalid or oversized API response")
	}
	text, err := apiText(o.Provider, raw)
	if err != nil {
		return Plan{}, err
	}
	return decodePlanText(text)
}

func apiText(provider string, raw []byte) (string, error) {
	var r struct {
		Status     string          `json:"status"`
		StopReason string          `json:"stop_reason"`
		Error      json.RawMessage `json:"error"`
		Output     []struct {
			Type    string `json:"type"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Candidates []struct {
			FinishReason string `json:"finishReason"`
			Content      struct {
				Parts []struct {
					Text    string `json:"text"`
					Thought bool   `json:"thought"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	bad := func() (string, error) {
		return "", fmt.Errorf("%s returned an incomplete, refused or invalid plan", provider)
	}
	if json.Unmarshal(raw, &r) != nil || (len(r.Error) > 0 && string(r.Error) != "null") {
		return bad()
	}
	var text strings.Builder
	switch provider {
	case "openai-api":
		if r.Status != "completed" {
			return bad()
		}
		for _, item := range r.Output {
			for _, part := range item.Content {
				if part.Type == "output_text" {
					text.WriteString(part.Text)
				}
			}
		}
	case "anthropic-api":
		if r.StopReason != "end_turn" {
			return bad()
		}
		for _, part := range r.Content {
			if part.Type == "text" {
				text.WriteString(part.Text)
			}
		}
	case "gemini-api":
		if len(r.Candidates) != 1 || r.Candidates[0].FinishReason != "STOP" {
			return bad()
		}
		for _, part := range r.Candidates[0].Content.Parts {
			if !part.Thought {
				text.WriteString(part.Text)
			}
		}
	case "openai-compatible":
		if len(r.Choices) != 1 || r.Choices[0].FinishReason != "stop" {
			return bad()
		}
		text.WriteString(r.Choices[0].Message.Content)
	}
	if text.Len() == 0 {
		return bad()
	}
	return text.String(), nil
}
