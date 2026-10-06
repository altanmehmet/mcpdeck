package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"io"
	"strings"
)

func addCommand(get func() store.Store) *cobra.Command {
	var name, command, profile, url, transport string
	var args, env, headers []string
	var disabled, noSync, jsonInput bool
	c := &cobra.Command{Use: "add", Short: "Add a local or remote MCP server to all profiles and sync", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		cfg := model.ServerConfig{}
		if jsonInput {
			for _, flag := range []string{"name", "command", "arg", "env", "url", "transport", "header"} {
				if c.Flags().Changed(flag) {
					return fmt.Errorf("--json-stdin cannot be combined with --%s", flag)
				}
			}
			var input struct {
				Name   string             `json:"name"`
				Server model.ServerConfig `json:"server"`
			}
			dec := json.NewDecoder(io.LimitReader(c.InOrStdin(), 1024*1024+1))
			dec.DisallowUnknownFields()
			if err := dec.Decode(&input); err != nil {
				return fmt.Errorf("invalid server JSON")
			}
			var extra any
			if dec.Decode(&extra) != io.EOF {
				return fmt.Errorf("expected one JSON object")
			}
			name, cfg = input.Name, input.Server
			if name == "" {
				return fmt.Errorf("server name is required")
			}
			cfg.CachedTools = nil
		}
		if name == "" && !jsonInput {
			reader := bufio.NewReader(c.InOrStdin())
			ask := func(label string) (string, error) {
				fmt.Fprint(c.OutOrStdout(), label)
				v, e := reader.ReadString('\n')
				if e != nil && e != io.EOF {
					return "", e
				}
				if e == io.EOF && v == "" {
					return "", io.EOF
				}
				return strings.TrimSpace(v), nil
			}
			var err error
			if name, err = ask("Server name: "); err != nil {
				return err
			}
			if command, err = ask("Executable: "); err != nil {
				return err
			}
			raw, err := ask("Arguments as JSON array ([]): ")
			if err != nil {
				return err
			}
			if raw != "" {
				if err = json.Unmarshal([]byte(raw), &args); err != nil {
					return fmt.Errorf("arguments must be a JSON string array")
				}
			}
			if profile, err = ask("Profile (blank = all, or a profile name): "); err != nil {
				return err
			}
			fmt.Fprintln(c.OutOrStdout(), "Environment variables: KEY=value or KEY=${VARIABLE}. Use references for secrets; blank line finishes.")
			for {
				value, err := ask("Environment variable (blank = done): ")
				if err == io.EOF {
					break
				}
				if err != nil {
					return err
				}
				if value == "" {
					break
				}
				env = append(env, value)
			}
		}
		if disabled && profile != "" {
			return fmt.Errorf("--disabled cannot be combined with a profile selection")
		}
		if !jsonInput {
			cfg = model.ServerConfig{Command: command, Args: args, URL: url, Transport: transport, Headers: map[string]string{}, Env: map[string]string{}}
		}
		for _, e := range env {
			k, v, ok := strings.Cut(e, "=")
			if !ok {
				return fmt.Errorf("env must be KEY=value")
			}
			cfg.Env[k] = v
		}
		for _, h := range headers {
			k, v, ok := strings.Cut(h, "=")
			if !ok {
				return fmt.Errorf("header must be NAME=value")
			}
			cfg.Headers[k] = v
		}
		err := get().Update(func(d *model.Deck) error {
			if _, ok := d.Servers[name]; ok {
				return fmt.Errorf("server %q already exists", name)
			}
			if profile != "" {
				if _, ok := d.Profiles[profile]; !ok {
					return fmt.Errorf("unknown profile %q", profile)
				}
			}
			d.Servers[name] = cfg
			if !disabled {
				return d.SetEnabled(profile, name, true)
			}
			return nil
		})
		if err != nil {
			return err
		}
		if disabled {
			fmt.Fprintln(c.OutOrStdout(), "Server saved, disabled in all profiles. Run mcpdeck enable <server> to apply.")
			return nil
		}
		if noSync {
			fmt.Fprintln(c.OutOrStdout(), "Server saved. Run mcpdeck sync to apply.")
			return nil
		}
		if err := syncSelection(get(), profile); err != nil {
			return fmt.Errorf("server saved; sync incomplete (retry with mcpdeck sync): %w", err)
		}
		fmt.Fprintln(c.OutOrStdout(), "Server saved and synced. Restart the affected agent MCP connections to apply.")
		return nil
	}}
	c.Flags().StringVar(&name, "name", "", "Server name")
	c.Flags().StringVar(&command, "command", "", "Executable (no shell)")
	c.Flags().StringVar(&url, "url", "", "Remote MCP http(s) endpoint")
	c.Flags().StringVar(&transport, "transport", "", "Remote transport: http (default) or sse")
	c.Flags().StringArrayVar(&headers, "header", nil, "NAME=value or NAME=${VARIABLE} (repeatable)")
	c.Flags().BoolVar(&jsonInput, "json-stdin", false, "Read {name,server} JSON from stdin, without placing secrets in arguments")
	c.Flags().StringArrayVar(&args, "arg", nil, "Argument (repeatable)")
	c.Flags().StringArrayVar(&env, "env", nil, "KEY=value or KEY=${VARIABLE} (repeatable)")
	c.Flags().StringVar(&profile, "profile", "", "Enable in one profile only (default: all configured profiles)")
	c.Flags().BoolVar(&disabled, "disabled", false, "Save without enabling or syncing")
	c.Flags().BoolVar(&noSync, "no-sync", false, "Save selections without writing agent settings yet")
	return c
}
