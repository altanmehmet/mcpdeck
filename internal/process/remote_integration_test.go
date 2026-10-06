package process

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Run explicitly with MCPDECK_REMOTE_TEST_DIR pointing at a directory containing
// npm-installed mcp-remote@0.1.38. This exercises the real adapter without downloads
// in the ordinary unit suite or contacting an external account.
func TestRemoteAdapterIntegration(t *testing.T) {
	dir := os.Getenv("MCPDECK_REMOTE_TEST_DIR")
	if dir == "" {
		t.Skip("set MCPDECK_REMOTE_TEST_DIR to test the installed remote adapter")
	}
	t.Chdir(dir)
	t.Setenv("npm_config_offline", "true")
	for _, transport := range []string{"http", "sse"} {
		t.Run(transport, func(t *testing.T) {
			var calls atomic.Int32
			var sessions sync.Map
			var sessionID atomic.Int32
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Logf("fixture request: %s %s authenticated=%t", r.Method, r.URL.Path, r.Header.Get("Authorization") == "Bearer integration-fixture")
				if r.Header.Get("Authorization") != "Bearer integration-fixture" {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				if r.Method == "GET" && transport == "sse" {
					id := fmt.Sprint(sessionID.Add(1))
					messages := make(chan []byte, 16)
					sessions.Store(id, messages)
					defer sessions.Delete(id)
					w.Header().Set("Content-Type", "text/event-stream")
					fmt.Fprintf(w, "event: endpoint\ndata: /messages?session=%s\n\n", id)
					w.(http.Flusher).Flush()
					for {
						select {
						case msg := <-messages:
							fmt.Fprintf(w, "event: message\ndata: %s\n\n", msg)
							w.(http.Flusher).Flush()
						case <-r.Context().Done():
							return
						}
					}
				}
				if r.Method != "POST" {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				var req struct {
					ID     json.RawMessage `json:"id"`
					Method string          `json:"method"`
					Params json.RawMessage `json:"params"`
				}
				if json.NewDecoder(r.Body).Decode(&req) != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if len(req.ID) == 0 {
					w.WriteHeader(http.StatusAccepted)
					return
				}
				var result any
				switch req.Method {
				case "initialize":
					var p struct {
						Version string `json:"protocolVersion"`
					}
					_ = json.Unmarshal(req.Params, &p)
					result = map[string]any{"protocolVersion": p.Version, "capabilities": map[string]any{"tools": map[string]any{}}, "serverInfo": map[string]string{"name": "remote-fixture", "version": "1"}}
				case "tools/list":
					result = map[string]any{"tools": []any{map[string]any{"name": "echo", "description": "test", "inputSchema": map[string]any{"type": "object"}}}}
				case "tools/call":
					calls.Add(1)
					result = map[string]any{"content": []any{map[string]string{"type": "text", "text": "remote adapter call passed"}}}
				default:
					result = map[string]any{}
				}
				raw, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
				if transport == "sse" {
					value, ok := sessions.Load(r.URL.Query().Get("session"))
					if !ok {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					value.(chan []byte) <- raw
					w.WriteHeader(http.StatusAccepted)
				} else {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write(raw)
				}
			})
			server := httptest.NewServer(handler)
			defer server.Close()
			deck := &model.Deck{Servers: map[string]model.ServerConfig{"remote": {URL: server.URL, Transport: transport, Headers: map[string]string{"Authorization": "Bearer integration-fixture"}}}}
			manager := New(deck, time.Minute)
			defer manager.KillAll()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			listed, err := manager.Request(ctx, "remote", "tools/list", json.RawMessage(`{}`))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(listed), `"echo"`) {
				t.Fatal("tool not discovered")
			}
			result, err := manager.Request(ctx, "remote", "tools/call", json.RawMessage(`{"name":"echo","arguments":{}}`))
			if err != nil {
				t.Fatal(err)
			}
			if calls.Load() != 1 || !strings.Contains(string(result), "remote adapter call passed") {
				t.Fatal("remote tool not executed")
			}
		})
	}
}
