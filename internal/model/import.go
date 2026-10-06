package model

import (
	"encoding/json"
	"fmt"
	"github.com/pelletier/go-toml/v2"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
)

type ImportedServer struct {
	Name     string       `json:"name"`
	Server   ServerConfig `json:"-"`
	Disabled bool         `json:"disabled"`
	Kind     string       `json:"kind"`
	Required []string     `json:"required"`
}

var reference = regexp.MustCompile(`\$\{([^}]+)\}|\$([A-Za-z_][A-Za-z0-9_]*)`)
var variableName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.-]*$`)

// Import accepts published MCP connection snippets, not arbitrary installation
// scripts. Unsupported fields fail explicitly rather than silently losing rules.
func Import(source, fallbackName string, values map[string]string) ([]ImportedServer, error) {
	source = strings.TrimSpace(source)
	if strings.HasPrefix(source, "```") {
		lines := strings.Split(source, "\n")
		if len(lines) < 3 || strings.TrimSpace(lines[len(lines)-1]) != "```" {
			return nil, fmt.Errorf("incomplete configuration code block")
		}
		source = strings.Join(lines[1:len(lines)-1], "\n")
	}
	var root map[string]json.RawMessage
	if strings.HasPrefix(source, "https://") || strings.HasPrefix(source, "http://") {
		if fallbackName == "" {
			parsed, err := url.Parse(source)
			if err != nil {
				return nil, fmt.Errorf("invalid MCP URL")
			}
			fallbackName = strings.ReplaceAll(parsed.Hostname(), ".", "-")
		}
		b, _ := json.Marshal(map[string]string{"url": source})
		root = map[string]json.RawMessage{fallbackName: b}
	} else {
		if json.Unmarshal([]byte(source), &root) != nil {
			var config map[string]any
			if toml.Unmarshal([]byte(source), &config) != nil {
				return nil, fmt.Errorf("paste an MCP URL, JSON configuration or Codex MCP TOML")
			}
			b, _ := json.Marshal(config)
			if json.Unmarshal(b, &root) != nil {
				return nil, fmt.Errorf("invalid configuration")
			}
		}
		wrappers := 0
		for _, field := range []string{"mcpServers", "servers", "mcp_servers", "context_servers", "mcp"} {
			if raw, ok := root[field]; ok {
				wrappers++
				var servers map[string]json.RawMessage
				if json.Unmarshal(raw, &servers) != nil || servers == nil {
					return nil, fmt.Errorf("%s must be an object", field)
				}
			}
		}
		if wrappers > 1 {
			return nil, fmt.Errorf("paste one MCP configuration format at a time")
		}
		if wrappers == 1 {
			for _, field := range []string{"mcpServers", "servers", "mcp_servers", "context_servers", "mcp"} {
				if raw, ok := root[field]; ok {
					var servers map[string]json.RawMessage
					_ = json.Unmarshal(raw, &servers)
					if field == "mcp" {
						if nested, ok := servers["servers"]; ok {
							_ = json.Unmarshal(nested, &servers)
						}
					}
					root = servers
					break
				}
			}
		} else if root["command"] != nil || root["url"] != nil || root["serverUrl"] != nil || root["httpUrl"] != nil {
			b, _ := json.Marshal(root)
			root = map[string]json.RawMessage{fallbackName: b}
		}
	}
	if len(root) == 0 {
		return nil, fmt.Errorf("no MCP servers found")
	}
	names := make([]string, 0, len(root))
	for name := range root {
		names = append(names, name)
	}
	sort.Strings(names)
	imported := make([]ImportedServer, 0, len(names))
	for _, name := range names {
		if name == "" {
			return nil, fmt.Errorf("enter a name for this connection")
		}
		var fields map[string]json.RawMessage
		if json.Unmarshal(root[name], &fields) != nil || fields == nil {
			return nil, fmt.Errorf("server %s must be an object", name)
		}
		allowed := map[string]bool{"command": true, "args": true, "env": true, "url": true, "serverUrl": true, "httpUrl": true, "headers": true, "http_headers": true, "type": true, "transport": true, "disabled": true, "enabled": true, "tools": true}
		for key := range fields {
			if !allowed[key] {
				return nil, fmt.Errorf("server %s: unsupported field %s; its behavior cannot be safely transferred to all agents", name, key)
			}
		}
		aliases := func(target string, alternatives ...string) error {
			for _, key := range alternatives {
				if raw, ok := fields[key]; ok {
					if _, exists := fields[target]; exists {
						return fmt.Errorf("server %s: conflicting %s fields", name, target)
					}
					fields[target] = raw
					delete(fields, key)
				}
			}
			return nil
		}
		if err := aliases("url", "serverUrl", "httpUrl"); err != nil {
			return nil, err
		}
		if err := aliases("headers", "http_headers"); err != nil {
			return nil, err
		}
		var typ, transport string
		if raw, ok := fields["type"]; ok {
			if json.Unmarshal(raw, &typ) != nil {
				return nil, fmt.Errorf("server %s: invalid type", name)
			}
		}
		if raw, ok := fields["transport"]; ok {
			if json.Unmarshal(raw, &transport) != nil {
				return nil, fmt.Errorf("server %s: invalid transport", name)
			}
		}
		if typ != "" && transport != "" && typ != transport {
			return nil, fmt.Errorf("server %s: conflicting transports", name)
		}
		if typ == "" {
			typ = transport
		}
		delete(fields, "type")
		delete(fields, "transport")
		switch typ {
		case "", "stdio", "local":
		case "http", "sse", "streamableHttp":
			b, _ := json.Marshal(typ)
			if typ == "streamableHttp" {
				b = json.RawMessage(`"http"`)
			}
			fields["transport"] = b
		case "streamable-http":
			fields["transport"] = json.RawMessage(`"http"`)
		default:
			return nil, fmt.Errorf("server %s: unsupported transport", name)
		}
		if (typ == "local" || typ == "stdio") && fields["url"] != nil {
			return nil, fmt.Errorf("server %s: local transport cannot have a URL", name)
		}
		item := ImportedServer{Name: name, Kind: "local", Required: []string{}}
		if raw, ok := fields["disabled"]; ok {
			if json.Unmarshal(raw, &item.Disabled) != nil {
				return nil, fmt.Errorf("server %s: invalid disabled value", name)
			}
			delete(fields, "disabled")
		}
		if raw, ok := fields["enabled"]; ok {
			var enabled bool
			if json.Unmarshal(raw, &enabled) != nil {
				return nil, fmt.Errorf("server %s: invalid enabled value", name)
			}
			item.Disabled = item.Disabled || !enabled
			delete(fields, "enabled")
		}
		if raw, ok := fields["tools"]; ok {
			var tools []string
			if json.Unmarshal(raw, &tools) != nil || len(tools) != 1 || tools[0] != "*" {
				return nil, fmt.Errorf("server %s: restricted tools cannot be transferred to all agents", name)
			}
			delete(fields, "tools")
		}
		b, _ := json.Marshal(fields)
		if json.Unmarshal(b, &item.Server) != nil {
			return nil, fmt.Errorf("server %s: invalid connection field types", name)
		}
		required := map[string]bool{}
		item.Server.Variables = map[string]string{}
		normalize := func(value string) string {
			return reference.ReplaceAllStringFunc(value, func(ref string) string {
				parts := reference.FindStringSubmatch(ref)
				key := parts[1]
				if key == "" {
					key = parts[2]
				}
				key = strings.TrimPrefix(key, "env:")
				if strings.HasPrefix(key, "input:") {
					key = "INPUT_" + strings.TrimPrefix(key, "input:")
				}
				if v, ok := values[key]; ok && v != "" {
					item.Server.Variables[key] = v
				} else if os.Getenv(key) == "" {
					required[key] = true
				}
				return "${" + key + "}"
			})
		}
		item.Server.URL = normalize(item.Server.URL)
		for i, v := range item.Server.Args {
			item.Server.Args[i] = normalize(v)
		}
		for k, v := range item.Server.Env {
			item.Server.Env[k] = normalize(v)
		}
		for k, v := range item.Server.Headers {
			item.Server.Headers[k] = normalize(v)
		}
		for key := range required {
			if !variableName.MatchString(key) {
				return nil, fmt.Errorf("server %s: unsupported variable reference", name)
			}
			item.Required = append(item.Required, key)
		}
		sort.Strings(item.Required)
		if item.Server.URL != "" {
			item.Kind = "remote"
		}
		deck := Deck{Version: 1, Servers: map[string]ServerConfig{name: item.Server}, Profiles: map[string]ProfileConfig{}}
		if err := deck.Validate(); err != nil {
			return nil, err
		}
		imported = append(imported, item)
	}
	return imported, nil
}
