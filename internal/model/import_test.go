package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestImportFormats(t *testing.T) {
	for _, source := range []string{
		`{"mcpServers":{"demo":{"url":"https://example.com/mcp"}}}`,
		`{"servers":{"demo":{"type":"http","url":"https://example.com/mcp"}}}`,
		`{"demo":{"serverUrl":"https://example.com/mcp"}}`,
		`{"demo":{"httpUrl":"https://example.com/mcp"}}`,
		"[mcp_servers.demo]\nurl = 'https://example.com/mcp'\n",
		"```json\n{\"demo\":{\"url\":\"https://example.com/mcp\"}}\n```",
		"https://example.com/mcp",
		`{"url":"https://example.com/mcp"}`,
	} {
		result, err := Import(source, "demo", nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 1 || result[0].Name != "demo" || result[0].Server.URL != "https://example.com/mcp" {
			t.Fatal("incorrect detected connection")
		}
	}
}
func TestImportRequiredValuesRemainLiteral(t *testing.T) {
	t.Setenv("IMPORT_TEST_TOKEN", "")
	t.Setenv("INPUT_key", "")
	source := `{"mcpServers":{"one":{"command":"node","args":["server.js"],"env":{"TOKEN":"${env:IMPORT_TEST_TOKEN}"}},"two":{"url":"https://example.com/mcp","headers":{"Authorization":"Bearer ${input:key}"}}}}`
	items, err := Import(source, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || len(items[0].Required) != 1 || len(items[1].Required) != 1 {
		t.Fatal("missing input prompts")
	}
	items, err = Import(source, "", map[string]string{"IMPORT_TEST_TOKEN": "literal$NOT_EXPANDED", "INPUT_key": "token$literal"})
	if err != nil {
		t.Fatal(err)
	}
	one, err := Resolve(items[0].Server)
	if err != nil || one.Env["TOKEN"] != "literal$NOT_EXPANDED" {
		t.Fatalf("literal input altered: %v", err)
	}
	two, err := Resolve(items[1].Server)
	if err != nil || two.Headers["Authorization"] != "Bearer token$literal" {
		t.Fatalf("header altered: %v", err)
	}
	preview, _ := json.Marshal(items)
	if strings.Contains(string(preview), "literal") || strings.Contains(string(preview), "Authorization") {
		t.Fatal("preview leaked secrets")
	}
}
func TestImportRejectsLossyConversions(t *testing.T) {
	for _, source := range []string{
		`{"mcpServers":{},"servers":{}}`,
		`{"demo":{"url":"https://example.com","serverUrl":"https://example.com"}}`,
		`{"demo":{"command":"node","cwd":"/private"}}`,
		`{"demo":{"url":"https://example.com","tools":["read-only"]}}`,
		`{"demo":{"url":"https://example.com","type":"stdio"}}`,
		`{"demo":{"url":"https://example.com","oauth":{"clientSecret":"secret"}}}`,
	} {
		if _, err := Import(source, "", nil); err == nil {
			t.Fatal("unsupported settings silently discarded")
		}
	}
}
