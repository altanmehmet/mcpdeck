package syncer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/pelletier/go-toml/v2"
	"github.com/altanmehmet/mcpdeck/internal/model"
)

func serverField(format string) string {
	switch format {
	case "codex":
		return "mcp_servers"
	case "vscode":
		return "servers"
	case "zed":
		return "context_servers"
	case "opencode":
		return "mcp"
	default:
		return "mcpServers"
	}
}
func serverEntry(cfg model.ServerConfig, format string) map[string]any {
	args := cfg.Args
	if args == nil {
		args = []string{}
	}
	entry := map[string]any{"command": cfg.Command, "args": args}
	if len(cfg.Env) > 0 {
		entry["env"] = cfg.Env
	}
	switch format {
	case "opencode":
		command := append([]string{cfg.Command}, args...)
		entry = map[string]any{"type": "local", "command": command, "enabled": true}
		if len(cfg.Env) > 0 {
			entry["environment"] = cfg.Env
		}
	case "vscode":
		entry["type"] = "stdio"
	case "copilot-cli":
		entry["type"] = "local"
		entry["tools"] = []string{"*"}
	}
	return entry
}

func UsesRemoteAdapter(cfg model.ServerConfig, p model.ProfileConfig) bool {
	return cfg.URL != "" && (p.RemoteFormat == "stdio" || (p.Format == "codex" && cfg.Transport == "sse") || (p.RemoteFormat == "" && p.Format != "codex" && p.Format != "vscode" && p.Format != "copilot-cli"))
}

func profileServerEntry(cfg model.ServerConfig, p model.ProfileConfig) map[string]any {
	if cfg.URL == "" {
		return serverEntry(cfg, p.Format)
	}
	// Formats without an explicit remote schema use the stdio adapter. This also
	// keeps custom/legacy profiles safe without guessing their client from a name.
	if UsesRemoteAdapter(cfg, p) {
		return serverEntry(model.Stdio(cfg), p.Format)
	}
	field := "url"
	if p.RemoteFormat == "serverUrl" {
		field = "serverUrl"
	} else if (p.Format == "gemini-cli" || p.Format == "qwen") && cfg.Transport != "sse" {
		field = "httpUrl"
	}
	entry := map[string]any{field: cfg.URL}
	if p.Format == "opencode" {
		entry = map[string]any{"type": "remote", "url": cfg.URL, "enabled": true}
	}
	if p.Format == "vscode" || p.Format == "copilot-cli" || p.RemoteFormat == "http" {
		transport := cfg.Transport
		if transport == "" {
			transport = "http"
		}
		if p.Format == "cline" && transport == "http" {
			transport = "streamableHttp"
		}
		if (p.Format == "continue" || p.Format == "roo-code") && transport == "http" {
			transport = "streamable-http"
		}
		entry["type"] = transport
	}
	if p.Format == "copilot-cli" {
		entry["tools"] = []string{"*"}
	}
	if len(cfg.Headers) > 0 {
		key := "headers"
		if p.Format == "codex" {
			key = "http_headers"
		}
		entry[key] = cfg.Headers
	}
	return entry
}
func decodeConfig(b []byte, format string) (map[string]any, error) {
	root := map[string]any{}
	if b == nil {
		return root, nil
	}
	if format == "codex" {
		if err := toml.Unmarshal(b, &root); err != nil {
			return nil, fmt.Errorf("refusing to overwrite invalid TOML configuration")
		}
	} else {
		if format == "vscode" || format == "zed" || format == "opencode" || format == "qwen" || format == "trae" {
			b = normalizeJSONC(b)
		}
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.UseNumber()
		if !json.Valid(b) || dec.Decode(&root) != nil || root == nil {
			return nil, fmt.Errorf("refusing to overwrite invalid IDE JSON object")
		}
	}
	return root, nil
}

func profileServers(root map[string]any, format string, create bool) (map[string]any, error) {
	if format != "opencode" {
		field := serverField(format)
		if value, ok := root[field]; ok {
			servers, valid := value.(map[string]any)
			if !valid {
				return nil, fmt.Errorf("%s must be an object/table", field)
			}
			return servers, nil
		}
		if create {
			servers := map[string]any{}
			root[field] = servers
			return servers, nil
		}
		return nil, nil
	}
	mcp, ok := root["mcp"].(map[string]any)
	if !ok {
		if _, exists := root["mcp"]; exists {
			return nil, fmt.Errorf("mcp must be an object")
		}
		if !create {
			return nil, nil
		}
		mcp = map[string]any{}
		root["mcp"] = mcp
	}
	// OpenCode v2 nests entries under mcp.servers; the stable schema uses mcp.
	if raw, exists := mcp["servers"]; exists {
		servers, valid := raw.(map[string]any)
		if !valid {
			return nil, fmt.Errorf("mcp.servers must be an object")
		}
		return servers, nil
	}
	return mcp, nil
}
func encodeConfig(root map[string]any, format string) ([]byte, error) {
	if format == "codex" {
		return toml.Marshal(root)
	}
	return json.MarshalIndent(root, "", "  ")
}

// ValidateConfig uses the same parser and root key as sync; errors never include config contents.
func ValidateConfig(b []byte, format string) error {
	root, err := decodeConfig(b, format)
	if err != nil {
		return err
	}
	_, err = profileServers(root, format, false)
	return err
}

// normalizeJSONC removes comments and trailing commas outside strings. The JSON
// decoder still rejects all other invalid syntax. Sync outputs ordinary JSON.
func normalizeJSONC(src []byte) []byte {
	b := append([]byte(nil), src...)
	quoted, escaped := false, false
	for i := 0; i < len(b); i++ {
		if quoted {
			if escaped {
				escaped = false
			} else if b[i] == '\\' {
				escaped = true
			} else if b[i] == '"' {
				quoted = false
			}
			continue
		}
		if b[i] == '"' {
			quoted = true
			continue
		}
		if b[i] == '/' && i+1 < len(b) {
			if b[i+1] == '/' {
				for i < len(b) && b[i] != '\n' {
					b[i] = ' '
					i++
				}
			} else if b[i+1] == '*' {
				b[i] = ' '
				i++
				b[i] = ' '
				closed := false
				for i+1 < len(b) {
					i++
					if b[i] == '*' && i+1 < len(b) && b[i+1] == '/' {
						b[i] = ' '
						i++
						b[i] = ' '
						closed = true
						break
					}
					if b[i] != '\n' && b[i] != '\r' {
						b[i] = ' '
					}
				}
				if !closed {
					return src
				}
			}
		}
	}
	quoted, escaped = false, false
	for i := 0; i < len(b); i++ {
		if quoted {
			if escaped {
				escaped = false
			} else if b[i] == '\\' {
				escaped = true
			} else if b[i] == '"' {
				quoted = false
			}
			continue
		}
		if b[i] == '"' {
			quoted = true
			continue
		}
		if b[i] == ',' {
			j := i + 1
			for j < len(b) && (b[j] == ' ' || b[j] == '\n' || b[j] == '\r' || b[j] == '\t') {
				j++
			}
			if j < len(b) && (b[j] == '}' || b[j] == ']') {
				b[i] = ' '
			}
		}
	}
	return b
}
