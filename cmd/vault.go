package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"syscall"

	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/process"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"github.com/altanmehmet/mcpdeck/internal/vault"
	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"
)

const privateInputNotice = "Private input: never sent to the planning agent or added to chat. Use --vault to keep values out of config files. Keep secrets out of normal chat."

func vaultCommand(get func() store.Store) *cobra.Command {
	c := &cobra.Command{Use: "vault", Short: "Keep MCP credentials in your OS credential store"}
	var stdin bool
	bind := &cobra.Command{Use: "bind <server> <variable>", Short: "Privately save a credential and replace its deck value with a vault reference", Args: cobra.ExactArgs(2), RunE: func(c *cobra.Command, args []string) error {
		s := get()
		d, err := s.Load()
		if err != nil {
			return err
		}
		cfg, ok := d.Servers[args[0]]
		if !ok {
			return fmt.Errorf("unknown MCP")
		}
		if err = cfg.CheckVaultTransport(); err != nil {
			return err
		}
		key := args[1]
		if !model.NamePattern.MatchString(key) {
			return fmt.Errorf("invalid variable name")
		}
		if _, ok := cfg.Variables[key]; !ok {
			return fmt.Errorf("variable is not bound in this MCP; use install --vault for a new connection")
		}
		if err = cfg.CheckVaultInput(key); err != nil {
			return err
		}
		fmt.Fprintln(c.ErrOrStderr(), privateInputNotice)
		var value []byte
		if stdin {
			value, err = io.ReadAll(io.LimitReader(c.InOrStdin(), vault.MaxSecret+1))
		} else {
			f, ok := c.InOrStdin().(*os.File)
			if !ok || !term.IsTerminal(f.Fd()) {
				return fmt.Errorf("use a terminal or --stdin; never put a secret in command arguments")
			}
			fmt.Fprint(c.ErrOrStderr(), "Secret (hidden): ")
			value, err = term.ReadPassword(f.Fd())
			fmt.Fprintln(c.ErrOrStderr())
		}
		if err != nil {
			return fmt.Errorf("cannot read private input")
		}
		defer clear(value)
		id := vault.NewID(s.Path, args[0], key)
		if err = vault.Set(id, value); err != nil {
			return err
		}
		if err = s.Update(func(current *model.Deck) error {
			latest, ok := current.Servers[args[0]]
			if !ok {
				return fmt.Errorf("MCP removed during vault save")
			}
			if err := latest.CheckVaultInput(key); err != nil {
				return err
			}
			if currentValue, exists := latest.Variables[key]; !exists || currentValue != cfg.Variables[key] {
				return fmt.Errorf("input binding changed during vault save; retry with the current deck")
			}
			if latest.Variables == nil {
				latest.Variables = map[string]string{}
			}
			latest.Variables[key] = vault.Reference(id)
			current.Servers[args[0]] = latest
			return nil
		}); err != nil {
			return err
		}
		fmt.Fprintln(c.OutOrStdout(), "Saved in OS vault. Run mcpdeck sync to replace agent settings with a local runtime reference. Old settings/backups may still contain the previous plaintext; rotate migrated credentials.")
		return nil
	}}
	bind.Flags().BoolVar(&stdin, "stdin", false, "Read exact secret bytes from stdin (no newline trimming)")
	runtime := &cobra.Command{Use: "exec <server>", Short: "Run one MCP with locally resolved vault credentials", Args: cobra.ExactArgs(1), RunE: func(c *cobra.Command, args []string) error {
		d, err := get().Load()
		if err != nil {
			return err
		}
		cfg, ok := d.Servers[args[0]]
		if !ok {
			return fmt.Errorf("unknown MCP")
		}
		if !cfg.HasVault() {
			return fmt.Errorf("MCP has no vault references")
		}
		for key := range cfg.Variables {
			if vault.IsReference(cfg.Variables[key]) {
				if err = cfg.CheckVaultInput(key); err != nil {
					return err
				}
			}
		}
		cfg, err = model.Resolve(cfg)
		if err != nil {
			return fmt.Errorf("cannot resolve MCP credentials; unlock the OS vault or re-enter the credential")
		}
		cfg = model.Stdio(cfg)
		ctx, cancel := signal.NotifyContext(c.Context(), os.Interrupt, syscall.SIGTERM)
		defer cancel()
		child := exec.CommandContext(ctx, cfg.Command, cfg.Args...)
		child.Env = os.Environ()
		for key, value := range cfg.Env {
			child.Env = append(child.Env, key+"="+value)
		}
		child.Stdin = c.InOrStdin()
		child.Stdout = c.OutOrStdout()
		child.Stderr = io.Discard
		if err = process.RunSetup(child); err != nil {
			return fmt.Errorf("MCP runtime stopped; child diagnostics suppressed to protect credentials")
		}
		return nil
	}}
	forget := &cobra.Command{Use: "forget <server> <variable>", Short: "Delete a vault credential; its MCP will fail closed until rebound", Args: cobra.ExactArgs(2), RunE: func(c *cobra.Command, args []string) error {
		d, err := get().Load()
		if err != nil {
			return err
		}
		cfg, ok := d.Servers[args[0]]
		if !ok {
			return fmt.Errorf("unknown MCP")
		}
		if err = vault.Delete(cfg.Variables[args[1]]); err != nil {
			return fmt.Errorf("vault credential could not be deleted")
		}
		fmt.Fprintln(c.OutOrStdout(), "Vault credential deleted. Rebind it before starting this MCP.")
		return nil
	}}
	c.AddCommand(bind, runtime, forget)
	return c
}

// protectInputs is called only after planning/chat/approval have finished.
func protectInputs(s store.Store, name string, cfg *model.ServerConfig) error {
	if err := cfg.CheckVaultTransport(); err != nil {
		return err
	}
	if cfg.Variables == nil {
		cfg.Variables = map[string]string{}
	}
	// Environment-provided inputs must not accidentally be resolved into plaintext during sync.
	used := regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}|\$([A-Za-z_][A-Za-z0-9_]*)`)
	fields := append([]string{cfg.URL}, cfg.Args...)
	for _, v := range cfg.Env {
		fields = append(fields, v)
	}
	for _, v := range cfg.Headers {
		fields = append(fields, v)
	}
	for _, field := range fields {
		for _, match := range used.FindAllStringSubmatch(field, -1) {
			key := match[1]
			if key == "" {
				key = match[2]
			}
			if key == "INSTALL_DIR" {
				continue
			}
			if _, ok := cfg.Variables[key]; !ok {
				value, exists := os.LookupEnv(key)
				if !exists || value == "" {
					return fmt.Errorf("missing private input %s", key)
				}
				cfg.Variables[key] = value
			}
		}
	}

	for key, value := range cfg.Variables {
		if key == "INSTALL_DIR" || vault.IsReference(value) {
			continue
		}
		if err := cfg.CheckVaultInput(key); err != nil {
			return err
		}
	}
	for key, value := range cfg.Variables {
		if key == "INSTALL_DIR" || vault.IsReference(value) {
			continue
		}
		id := vault.NewID(s.Path, name, key)
		raw := []byte(value)
		err := vault.Set(id, raw)
		clear(raw)
		if err != nil {
			return err
		}
		cfg.Variables[key] = vault.Reference(id)
	}
	return nil
}
