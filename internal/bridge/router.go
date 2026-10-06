package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"reflect"
	"sort"
	"strings"
)

type toolPage struct {
	Tools      []map[string]json.RawMessage `json:"tools"`
	NextCursor string                       `json:"nextCursor,omitempty"`
}

func validateToolNames(server string, tools []map[string]json.RawMessage) error {
	seen := map[string]bool{}
	for _, tool := range tools {
		var name string
		if json.Unmarshal(tool["name"], &name) != nil || name == "" {
			return errors.New("invalid tool name")
		}
		if len(server+"__"+name) > 128 || seen[name] {
			return fmt.Errorf("invalid or duplicate tool name for %s", server)
		}
		seen[name] = true
	}
	return nil
}

func (b *Bridge) list(ctx context.Context) (json.RawMessage, error) {
	all := []map[string]json.RawMessage{}
	names := append([]string(nil), b.deck.Profiles[b.profile].EnabledServers...)
	sort.Strings(names)
	seen := map[string]bool{}
	for _, name := range names {
		cfg := b.deck.Servers[name]
		var page toolPage
		cached := len(cfg.CachedTools) > 0 && json.Unmarshal(cfg.CachedTools, &page) == nil && page.Tools != nil && page.NextCursor == "" && validateToolNames(name, page.Tools) == nil
		if !cached {
			page = toolPage{Tools: []map[string]json.RawMessage{}}
			cursor := ""
			visited := map[string]bool{}
			for {
				params, _ := json.Marshal(map[string]string{"cursor": cursor})
				if cursor == "" {
					params = json.RawMessage(`{}`)
				}
				raw, err := b.manager.Request(ctx, name, "tools/list", params)
				if err != nil {
					return nil, err
				}
				var part toolPage
				if json.Unmarshal(raw, &part) != nil || part.Tools == nil {
					return nil, errors.New("invalid child tool list")
				}
				page.Tools = append(page.Tools, part.Tools...)
				if part.NextCursor == "" {
					break
				}
				if visited[part.NextCursor] || len(visited) >= 1000 {
					return nil, errors.New("invalid tool pagination")
				}
				visited[part.NextCursor] = true
				cursor = part.NextCursor
			}
		}
		for _, tool := range page.Tools {
			var original string
			if json.Unmarshal(tool["name"], &original) != nil || original == "" {
				return nil, errors.New("invalid cached tool name")
			}
			full := name + "__" + original
			if len(full) > 128 || seen[full] {
				return nil, fmt.Errorf("invalid or duplicate tool name for %s", name)
			}
			seen[full] = true
			copy := map[string]json.RawMessage{}
			for k, v := range tool {
				copy[k] = v
			}
			copy["name"], _ = json.Marshal(full)
			all = append(all, copy)
		}
		if !cached {
			raw, _ := json.Marshal(page)
			if err := b.store.Update(func(d *model.Deck) error {
				current, ok := d.Servers[name]
				if ok && current.Command == cfg.Command && current.URL == cfg.URL && current.Transport == cfg.Transport && reflect.DeepEqual(current.Variables, cfg.Variables) && reflect.DeepEqual(current.Headers, cfg.Headers) && reflect.DeepEqual(current.Args, cfg.Args) && reflect.DeepEqual(current.Env, cfg.Env) {
					current.CachedTools = raw
					d.Servers[name] = current
				}
				return nil
			}); err != nil {
				return nil, err
			}
			cfg.CachedTools = raw
			b.deck.Servers[name] = cfg
		}
	}
	return json.Marshal(map[string]any{"tools": all})
}
func (b *Bridge) call(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
	var p map[string]json.RawMessage
	var name string
	if json.Unmarshal(params, &p) != nil || json.Unmarshal(p["name"], &name) != nil {
		return nil, &Error{Code: -32602, Message: "tool name is required"}
	}
	server, tool, ok := strings.Cut(name, "__")
	if !ok || tool == "" || !b.deck.IsEnabled(b.profile, server) {
		return nil, &Error{Code: -32602, Message: "unknown or disabled tool namespace"}
	}
	if raw, ok := p["arguments"]; ok {
		var args map[string]json.RawMessage
		if json.Unmarshal(raw, &args) != nil || args == nil {
			return nil, &Error{Code: -32602, Message: "arguments must be an object"}
		}
	}
	p["name"], _ = json.Marshal(tool)
	raw, _ := json.Marshal(p)
	return b.manager.Request(ctx, server, "tools/call", raw)
}
