package model

import (
	"fmt"
	"github.com/altanmehmet/mcpdeck/internal/vault"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"
)

type ServerConfig struct {
	Variables   map[string]string `json:"variables,omitempty"`
	URL         string            `json:"url,omitempty"`
	Transport   string            `json:"transport,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Command     string            `json:"command"`
	Args        []string          `json:"args"`
	Env         map[string]string `json:"env,omitempty"`
	CachedTools []byte            `json:"cached_tools,omitempty"`
}
type ProfileConfig struct {
	RemoteFormat   string   `json:"remote_format,omitempty"`
	TargetPath     string   `json:"target_path"`
	EnabledServers []string `json:"enabled_servers"`
	Mode           string   `json:"mode,omitempty"`
	Format         string   `json:"format,omitempty"`
}
type Deck struct {
	Version  int                      `json:"version"`
	Servers  map[string]ServerConfig  `json:"servers"`
	Profiles map[string]ProfileConfig `json:"profiles"`
}

var NamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func (d *Deck) IsEnabled(profile, server string) bool {
	for _, s := range d.Profiles[profile].EnabledServers {
		if s == server {
			return true
		}
	}
	return false
}
func (d *Deck) Toggle(profile, server string) {
	p, ok := d.Profiles[profile]
	if !ok {
		return
	}
	if _, ok := d.Servers[server]; !ok {
		return
	}
	for i, s := range p.EnabledServers {
		if s == server {
			p.EnabledServers = append(p.EnabledServers[:i:i], p.EnabledServers[i+1:]...)
			d.Profiles[profile] = p
			return
		}
	}
	p.EnabledServers = append(p.EnabledServers, server)
	d.Profiles[profile] = p
}

// SetEnabled applies an idempotent change to one profile, or all profiles when
// profile is empty. Validation happens before any selection is changed.
func (d *Deck) SetEnabled(profile, server string, enabled bool) error {
	if _, ok := d.Servers[server]; !ok {
		return fmt.Errorf("unknown server %q", server)
	}
	if profile != "" {
		if _, ok := d.Profiles[profile]; !ok {
			return fmt.Errorf("unknown profile %q", profile)
		}
	}
	if len(d.Profiles) == 0 {
		return fmt.Errorf("no profiles configured; run mcpdeck profiles add-defaults")
	}
	for name := range d.Profiles {
		if (profile == "" || profile == name) && d.IsEnabled(name, server) != enabled {
			d.Toggle(name, server)
		}
	}
	return nil
}

func (d *Deck) Validate() error {
	if d.Version != 1 {
		return fmt.Errorf("unsupported deck version %d", d.Version)
	}
	if d.Servers == nil || d.Profiles == nil {
		return fmt.Errorf("servers and profiles must be objects")
	}
	for name, s := range d.Servers {
		if name == "mcpdeck" || !NamePattern.MatchString(name) || strings.Contains(name, "__") || len(name) > 40 {
			return fmt.Errorf("invalid server %q", name)
		}
		if err := s.ValidateConnection(); err != nil {
			return fmt.Errorf("server %s: %w", name, err)
		}
		for key := range s.Variables {
			if !variableName.MatchString(key) {
				return fmt.Errorf("invalid input variable in %s", name)
			}
		}
		for k := range s.Env {
			if k == "" || strings.ContainsAny(k, "=\x00") {
				return fmt.Errorf("invalid environment key in %s", name)
			}
		}
	}
	for name, p := range d.Profiles {
		switch p.RemoteFormat {
		case "", "stdio", "url", "serverUrl", "http":
		default:
			return fmt.Errorf("invalid remote format for %s", name)
		}
		if strings.TrimSpace(p.TargetPath) == "" {
			return fmt.Errorf("profile %s has no target path", name)
		}
		if p.Mode != "" && p.Mode != "direct" && p.Mode != "bridge" {
			return fmt.Errorf("invalid mode for %s", name)
		}
		switch p.Format {
		case "", "mcpServers", "vscode", "copilot-cli", "codex", "zed", "gemini-cli", "opencode", "cline", "continue", "roo-code", "qwen", "trae":
		default:
			return fmt.Errorf("invalid format for %s", name)
		}
		seen := map[string]bool{}
		for _, s := range p.EnabledServers {
			if _, ok := d.Servers[s]; !ok {
				return fmt.Errorf("profile %s references unknown server %s", name, s)
			}
			if seen[s] {
				return fmt.Errorf("duplicate server in %s", name)
			}
			seen[s] = true
		}
	}
	return nil
}

// Expand resolves environment references without invoking a shell. Missing values fail closed.
func Expand(s string) (string, error) {
	return expandWith(s, nil)
}
func expandWith(s string, values map[string]string) (string, error) {
	var missing string
	result := os.Expand(s, func(k string) string {
		v, ok := values[k]
		if ok && vault.IsReference(v) {
			var err error
			v, err = vault.Get(v)
			if err != nil {
				missing = k
				return ""
			}
		}
		if !ok {
			v, ok = os.LookupEnv(k)
		}
		if !ok || v == "" {
			missing = k
		}
		return v
	})
	if missing != "" {
		return "", fmt.Errorf("environment variable %s is missing", missing)
	}
	return result, nil
}
func Resolve(s ServerConfig) (ServerConfig, error) {
	if s.HasVault() {
		if err := s.CheckVaultTransport(); err != nil {
			return ServerConfig{}, err
		}
	}
	for key, value := range s.Variables {
		if vault.IsReference(value) {
			if err := s.CheckVaultInput(key); err != nil {
				return ServerConfig{}, err
			}
		}
	}

	r := ServerConfig{Command: s.Command, URL: s.URL, Transport: s.Transport, Headers: map[string]string{}, Args: make([]string, len(s.Args)), Env: map[string]string{}}
	var err error
	if s.URL != "" {
		r.URL, err = expandWith(s.URL, s.Variables)
		if err != nil {
			return r, err
		}
		for k, v := range s.Headers {
			r.Headers[k], err = expandWith(v, s.Variables)
			if err != nil {
				return r, err
			}
		}
		if err = r.ValidateConnection(); err != nil {
			return r, err
		}
	}
	for i, a := range s.Args {
		r.Args[i], err = expandWith(a, s.Variables)
		if err != nil {
			return r, err
		}
	}
	for k, v := range s.Env {
		r.Env[k], err = expandWith(v, s.Variables)
		if err != nil {
			return r, err
		}
		if r.Env[k] == "" {
			return r, fmt.Errorf("environment variable %s is empty", k)
		}
	}
	return r, nil
}

// HasVault prevents sync from resolving private values into agent config files.
func (s ServerConfig) HasVault() bool {
	for _, value := range s.Variables {
		if vault.IsReference(value) {
			return true
		}
	}
	return false
}

// CheckVaultInput prohibits vault secrets in URLs or process argument lists.
func (s ServerConfig) CheckVaultInput(key string) error {
	for _, value := range append([]string{s.Command, s.URL}, s.Args...) {
		reference := regexp.MustCompile(`\$\{` + regexp.QuoteMeta(key) + `\}|\$` + regexp.QuoteMeta(key) + `(?:[^A-Za-z0-9_]|$)`)
		if reference.MatchString(value) {
			return fmt.Errorf("vault inputs must use environment variables or HTTP headers, not executable, URL or arguments")
		}
	}
	return nil
}

// CheckVaultTransport prevents OS-vault credentials from being sent over public
// plaintext HTTP. Loopback endpoints remain usable for local MCP services.
func (s ServerConfig) CheckVaultTransport() error {
	if s.URL == "" {
		return nil
	}
	u, err := url.Parse(s.URL)
	if err == nil && u.Scheme == "https" {
		return nil
	}
	if err == nil && u.Scheme == "http" {
		host := u.Hostname()
		if host == "localhost" {
			return nil
		}
		if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
			return nil
		}
	}
	return fmt.Errorf("vault credentials require HTTPS for remote MCP endpoints; HTTP is allowed only for localhost or loopback IPs")
}
