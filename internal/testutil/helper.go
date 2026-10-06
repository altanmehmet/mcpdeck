// Package testutil supplies a hermetic MCP subprocess for integration tests.
package testutil

import (
	"bufio"
	"encoding/json"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"os"
	"time"
)

func Config() model.ServerConfig {
	return model.ServerConfig{Command: os.Args[0], Args: []string{"-test.run=^TestHelper$"}, Env: map[string]string{"MCPDECK_TEST_CHILD": "1"}}
}
func Serve() {
	if os.Getenv("MCPDECK_TEST_CHILD") != "1" {
		return
	}
	defer os.Exit(0)
	scanner := bufio.NewScanner(os.Stdin)
	enc := json.NewEncoder(os.Stdout)
	ready := false
	for scanner.Scan() {
		var r struct {
			ID     json.RawMessage            `json:"id"`
			Method string                     `json:"method"`
			Params map[string]json.RawMessage `json:"params"`
		}
		if json.Unmarshal(scanner.Bytes(), &r) != nil {
			os.Exit(2)
		}
		if r.Method == "notifications/initialized" {
			ready = true
			continue
		}
		var result any
		switch r.Method {
		case "initialize":
			result = map[string]any{"protocolVersion": "2025-11-25", "capabilities": map[string]any{"tools": map[string]any{}}}
		case "tools/list":
			if !ready {
				os.Exit(3)
			}
			name := "query"
			cursor := "page2"
			if string(r.Params["cursor"]) == `"page2"` {
				name = "other"
				cursor = ""
			}
			result = map[string]any{"tools": []any{map[string]any{"name": name, "inputSchema": map[string]any{"type": "object"}}}, "nextCursor": cursor}
		case "tools/call":
			if !ready {
				os.Exit(4)
			}
			if string(r.Params["name"]) == `"slow"` {
				time.Sleep(200 * time.Millisecond)
			}
			if string(r.Params["name"]) == `"hang"` {
				time.Sleep(30 * time.Second)
			}
			_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "method": "notifications/message", "params": map[string]any{}})
			result = map[string]any{"content": []any{map[string]any{"type": "text", "text": string(r.Params["name"])}}, "isError": false}
		default:
			continue
		}
		_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "id": r.ID, "result": result})
	}
}
