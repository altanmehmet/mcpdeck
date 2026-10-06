package model

import (
	"strings"
	"testing"
)

func TestRemoteValidation(t *testing.T) {
	for _, s := range []ServerConfig{
		{URL: "ftp://example.com"}, {URL: "https://user:secret@example.com/mcp"}, {URL: "https://example.com/#fragment"},
		{URL: "https://example.com", Command: "echo"}, {URL: "https://example.com", Args: []string{"x"}},
		{URL: "https://example.com", Env: map[string]string{"KEY": "x"}}, {URL: "https://example.com", Transport: "other"},
		{URL: "https://example.com", Headers: map[string]string{"Bad Header": "x"}}, {URL: "https://example.com", Headers: map[string]string{"Auth": "x\r\nInjected:y"}},
		{Command: "echo", Headers: map[string]string{"Auth": "x"}},
	} {
		if s.ValidateConnection() == nil {
			t.Fatalf("invalid connection accepted: transport=%s", s.Transport)
		}
	}
	for _, transport := range []string{"", "http", "sse"} {
		if err := (ServerConfig{URL: "https://example.com/mcp", Transport: transport}).ValidateConnection(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRemoteResolveAndAdapter(t *testing.T) {
	t.Setenv("MCPDECK_TEST_HEADER", "Bearer test-secret")
	cfg := ServerConfig{URL: "https://example.com/mcp", Headers: map[string]string{"Authorization": "${MCPDECK_TEST_HEADER}"}}
	resolved, err := Resolve(cfg)
	if err != nil {
		t.Fatal(err)
	}
	adapter := Stdio(resolved)
	if strings.Contains(strings.Join(adapter.Args, " "), "test-secret") {
		t.Fatal("secret in arguments")
	}
	if adapter.Env["MCPDECK_HEADER_0"] != "Bearer test-secret" {
		t.Fatal("header missing")
	}
	if cfg.Headers["Authorization"] != "${MCPDECK_TEST_HEADER}" {
		t.Fatal("source mutated")
	}
	t.Setenv("MCPDECK_TEST_HEADER", "bad\nheader")
	if _, err = Resolve(cfg); err == nil {
		t.Fatal("expanded header injection accepted")
	}
	t.Setenv("MCPDECK_TEST_HEADER", "")
	if _, err = Resolve(cfg); err == nil {
		t.Fatal("missing credential accepted")
	}
}
