package cmd

import (
	"fmt"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"github.com/altanmehmet/mcpdeck/internal/syncer"
	"github.com/altanmehmet/mcpdeck/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"path/filepath"
)

// version is overridden in versioned distribution builds using -ldflags -X.
var version = "0.1.0"

func Execute() error { return NewRoot().Execute() }
func NewRoot() *cobra.Command {
	config := store.Default().Path
	var noMouse bool
	root := &cobra.Command{Use: "mcpdeck", Short: "Manage MCP servers across IDEs", SilenceUsage: true, Version: version}
	root.PersistentFlags().StringVar(&config, "config", config, "Deck configuration file")
	root.Flags().BoolVar(&noMouse, "no-mouse", false, "Start with terminal text selection enabled instead of mouse controls")
	getStore := func() store.Store {
		p, err := filepath.Abs(syncer.ExpandPath(config))
		if err != nil {
			return store.Store{Path: config}
		}
		return store.Store{Path: p}
	}
	root.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("unknown command %q", args[0])
		}
		s := getStore()
		d, err := s.Load()
		if err != nil {
			return err
		}
		options := []tea.ProgramOption{tea.WithAltScreen()}
		if !noMouse {
			options = append(options, tea.WithMouseCellMotion())
		}
		_, err = tea.NewProgram(tui.NewWithMouse(d, s, !noMouse), options...).Run()
		return err
	}
	root.AddCommand(syncCommand(getStore), bridgeCommand(getStore), addCommand(getStore), doctorCommand(getStore), cacheCommand(getStore), profilesCommand(getStore))
	root.AddCommand(activationCommand(getStore, true), activationCommand(getStore, false))
	root.AddCommand(statusCommand(getStore))
	root.AddCommand(importCommand(getStore))
	root.AddCommand(setupCommand(getStore))
	root.AddCommand(installCommand(getStore))
	root.AddCommand(repairCommand(getStore))
	root.AddCommand(plannerCommand(getStore))
	root.AddCommand(environmentCommand())
	root.AddCommand(removeCommand(getStore))
	root.AddCommand(discoverCommand(getStore))
	root.AddCommand(instructionsCommand(getStore))
	return root
}
