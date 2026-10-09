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
	root := &cobra.Command{Use: "mcpdeck", Short: "Manage MCP servers and instructions across coding agents", SilenceUsage: true, Version: version}
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
	root.AddCommand(statusCommand(getStore), listCommand(getStore), agentsCommand(getStore))
	root.AddCommand(importCommand(getStore))
	root.AddCommand(setupCommand(getStore))
	root.AddCommand(installCommand(getStore))
	root.AddCommand(repairCommand(getStore))
	root.AddCommand(plannerCommand(getStore))
	root.AddCommand(environmentCommand())
	root.AddCommand(removeCommand(getStore))
	root.AddCommand(discoverCommand(getStore))
	root.AddCommand(instructionsCommand(getStore))
	root.Long = "Manage MCP connections and shared instructions from one place.\nRun mcpdeck without a command to open the interactive panel."
	root.Example = "  mcpdeck                               Open the panel\n  mcpdeck list                          See your MCP servers\n  mcpdeck agents                        See configured agents\n  mcpdeck install \"Add the official Git MCP\"\n  mcpdeck enable git --profile cursor    Enable for one agent\n  mcpdeck doctor                        Check your setup"
	root.AddGroup(&cobra.Group{ID: "daily", Title: "Everyday commands:"}, &cobra.Group{ID: "connections", Title: "Add and manage connections:"}, &cobra.Group{ID: "maintenance", Title: "Maintenance:"}, &cobra.Group{ID: "advanced", Title: "Advanced and scripting:"})
	root.SetHelpCommandGroupID("advanced")
	root.SetCompletionCommandGroupID("advanced")
	for _, c := range root.Commands() {
		shorts := map[string]string{"profiles": "Show profile paths and formats", "install": "Connect an MCP with a reviewed installation", "add": "Register a server by command or URL", "import": "Import existing MCP settings", "remove": "Remove an MCP from its agents", "sync": "Apply saved settings to agents", "repair": "Repair or update an installed MCP", "doctor": "Check runtimes and configuration", "enable": "Enable an MCP (all agents unless --profile is set)", "disable": "Disable an MCP (all agents unless --profile is set)"}
		if short, ok := shorts[c.Name()]; ok {
			c.Short = short
		}
		switch c.Name() {
		case "list", "agents", "enable", "disable", "instructions":
			c.GroupID = "daily"
		case "install", "add", "import", "remove", "discover":
			c.GroupID = "connections"
		case "sync", "doctor", "repair":
			c.GroupID = "maintenance"
		default:
			c.GroupID = "advanced"
		}
	}
	return root
}
