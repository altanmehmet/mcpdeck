package model

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
)

// RemoteAdapterVersion is pinned so existing configurations do not silently
// switch executable dependencies. Native HTTP clients do not need this adapter.
const RemoteAdapterVersion = "0.1.38"

var headerName = regexp.MustCompile("^[!#$%&'*+.^_`|~0-9A-Za-z-]+$")

func (s ServerConfig) ValidateConnection() error {
	if s.URL == "" {
		if strings.TrimSpace(s.Command) == "" {
			return fmt.Errorf("provide an executable or an MCP URL")
		}
		if s.Transport != "" || len(s.Headers) > 0 {
			return fmt.Errorf("transport and headers require an MCP URL")
		}
		return nil
	}
	if s.Command != "" || len(s.Args) > 0 || len(s.Env) > 0 {
		return fmt.Errorf("URL connections cannot include command, arguments or environment; use headers")
	}
	if s.Transport != "" && s.Transport != "http" && s.Transport != "sse" {
		return fmt.Errorf("transport must be http or sse")
	}
	u, err := url.Parse(s.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.Fragment != "" {
		return fmt.Errorf("MCP URL must be http(s), without embedded credentials or a fragment")
	}
	for k, v := range s.Headers {
		if !headerName.MatchString(k) || strings.ContainsAny(v, "\r\n\x00") {
			return fmt.Errorf("invalid HTTP header")
		}
	}
	return nil
}

// Stdio adapts remote servers for local-only clients and the on-demand bridge.
// Secret header values live in the child environment, never in process arguments.
func Stdio(s ServerConfig) ServerConfig {
	if s.URL == "" {
		return s
	}
	transport := "http-only"
	if s.Transport == "sse" {
		transport = "sse-only"
	}
	r := ServerConfig{Command: "npx", Args: []string{"-y", "mcp-remote@" + RemoteAdapterVersion, s.URL, "--transport", transport, "--silent"}, Env: map[string]string{}}
	// Finder-launched agents may not inherit the user's Node installation path.
	if executable, err := exec.LookPath("npx"); err == nil {
		r.Command = executable
	}
	if path := os.Getenv("PATH"); path != "" {
		r.Env["PATH"] = path
	}
	if strings.HasPrefix(s.URL, "http://") {
		r.Args = append(r.Args, "--allow-http")
	}
	keys := make([]string, 0, len(s.Headers))
	for k := range s.Headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i, k := range keys {
		key := fmt.Sprintf("MCPDECK_HEADER_%d", i)
		r.Env[key] = s.Headers[k]
		r.Args = append(r.Args, "--header", k+":${"+key+"}")
	}
	return r
}
