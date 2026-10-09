package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/altanmehmet/mcpdeck/internal/instructions"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"github.com/altanmehmet/mcpdeck/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func instructionsCommand(get func() store.Store) *cobra.Command {
	var agent, documentPath string
	root := &cobra.Command{Use: "instructions", Short: "Manage personal instructions across agents; project rules are excluded", Args: cobra.NoArgs}
	root.PersistentFlags().StringVar(&agent, "agent", "", "View/edit one agent's existing global instruction file")
	root.PersistentFlags().StringVar(&documentPath, "path", "", "Select a global file shown by instructions files")
	root.PersistentPreRunE = func(c *cobra.Command, _ []string) error {
		if documentPath != "" && agent == "" {
			return fmt.Errorf("--path requires --agent")
		}
		if agent != "" && (c.Name() == "sync" || c.Name() == "clear") {
			return fmt.Errorf("%s operates on shared instructions across all agents; omit --agent", c.Name())
		}
		return nil
	}
	root.RunE = func(c *cobra.Command, _ []string) error {
		d, err := get().Load()
		if err != nil {
			return err
		}
		editor := tui.NewInstructions(d, get())
		if agent != "" {
			editor, err = tui.NewAgentInstructions(d, get(), agent, documentPath)
			if err != nil {
				return err
			}
		}
		_, err = tea.NewProgram(editor, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
		return err
	}
	root.AddCommand(&cobra.Command{Use: "show", Short: "Print centrally managed personal instructions", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		manager := instructions.New(get())
		text, err := manager.Load()
		if agent != "" {
			d, e := get().Load()
			if e != nil {
				return e
			}
			doc, e := manager.SelectDocument(d, agent, documentPath)
			if e != nil {
				return e
			}
			text, err = manager.ReadDocument(d, doc)
		}
		if err == nil {
			fmt.Fprint(c.OutOrStdout(), text)
		}
		return err
	}})
	root.AddCommand(&cobra.Command{Use: "files", Short: "List current global instruction files, including rules added outside MCPDeck", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		d, err := get().Load()
		if err != nil {
			return err
		}
		docs, err := instructions.New(get()).Documents(d)
		if err != nil {
			return err
		}
		for _, doc := range docs {
			if agent == "" || doc.Agent == agent {
				fmt.Fprintf(c.OutOrStdout(), "%s\t%s\t%s\n", safeText(doc.Agent), safeText(doc.Path), safeText(doc.Note))
			}
		}
		return nil
	}})
	var all bool
	root.PersistentFlags().BoolVar(&all, "all-profiles", false, "Include configured global adapters even when the agent is not detected")
	root.AddCommand(&cobra.Command{Use: "status", Short: "Show verified global targets and pending changes", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		d, err := get().Load()
		if err != nil {
			return err
		}
		results, err := instructions.New(get()).Status(d, all)
		if err == nil {
			if agent != "" {
				filtered := results[:0]
				for _, result := range results {
					if result.Agent == agent {
						filtered = append(filtered, result)
					}
				}
				if len(filtered) == 0 {
					return fmt.Errorf("unknown agent: %s", safeText(agent))
				}
				results = filtered
			}
			showInstructionResults(c, results)
		}
		return err
	}})
	var file string
	set := &cobra.Command{Use: "set [text]", Short: "Replace personal instructions and distribute them to detected agents", Args: cobra.MaximumNArgs(1)}
	set.Flags().StringVar(&file, "file", "", "Read UTF-8 instructions from a file, or - for stdin")
	set.RunE = func(c *cobra.Command, args []string) error {
		if (len(args) == 0) == (file == "") {
			return fmt.Errorf("supply text or --file, not both")
		}
		text := ""
		if len(args) == 1 {
			text = args[0]
		} else {
			reader := c.InOrStdin()
			if file != "-" {
				f, err := os.Open(file)
				if err != nil {
					return err
				}
				defer f.Close()
				reader = f
			}
			limit := instructions.MaxBytes
			if agent != "" {
				limit = instructions.MaxDocumentBytes
			}
			raw, err := io.ReadAll(io.LimitReader(reader, int64(limit+1)))
			if err != nil {
				return err
			}
			text = string(raw)
		}
		if strings.TrimSpace(text) == "" {
			return fmt.Errorf("instructions are empty; use instructions clear to remove the managed guidance")
		}
		d, err := get().Load()
		if err != nil {
			return err
		}
		if agent != "" {
			manager := instructions.New(get())
			doc, err := manager.SelectDocument(d, agent, documentPath)
			if err != nil {
				return err
			}
			current, err := manager.ReadDocument(d, doc)
			if err != nil {
				return err
			}
			if err := manager.SaveDocument(d, doc, text, current); err != nil {
				return err
			}
			fmt.Fprintf(c.OutOrStdout(), "%s: global file saved with a backup\n", safeText(agent))
			return nil
		}
		results, err := instructions.New(get()).Apply(d, text, all, nil)
		showInstructionResults(c, results)
		return err
	}
	root.AddCommand(set)
	root.AddCommand(&cobra.Command{Use: "append <instruction>", Short: "Add one shared instruction and distribute it to all detected supported agents", Args: cobra.ExactArgs(1), RunE: func(c *cobra.Command, args []string) error {
		if agent != "" {
			return fmt.Errorf("append adds shared instructions to all agents; omit --agent")
		}
		d, err := get().Load()
		if err != nil {
			return err
		}
		manager := instructions.New(get())
		current, err := manager.Load()
		if err != nil {
			return err
		}
		text, err := instructions.AppendText(current, args[0])
		if err != nil {
			return err
		}
		results, err := manager.Apply(d, text, all, &current)
		showInstructionResults(c, results)
		return err
	}})
	root.AddCommand(&cobra.Command{Use: "sync", Short: "Retry distributing the saved personal instructions", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		d, err := get().Load()
		if err != nil {
			return err
		}
		manager := instructions.New(get())
		text, err := manager.Load()
		if err != nil {
			return err
		}
		results, err := manager.Apply(d, text, all, &text)
		showInstructionResults(c, results)
		return err
	}})
	var yes bool
	clear := &cobra.Command{Use: "clear", Short: "Remove only MCPDeck's global instruction blocks", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		if !yes {
			return fmt.Errorf("use --yes to remove the managed personal instructions from all configured global targets")
		}
		d, err := get().Load()
		if err != nil {
			return err
		}
		results, err := instructions.New(get()).Apply(d, "", true, nil)
		showInstructionResults(c, results)
		return err
	}}
	clear.Flags().BoolVar(&yes, "yes", false, "Confirm removal of MCPDeck's managed instructions")
	root.AddCommand(clear)
	return root
}

func showInstructionResults(c *cobra.Command, results []instructions.Result) {
	for _, r := range results {
		fmt.Fprintf(c.OutOrStdout(), "%s: %s\n", safeText(r.Agent), safeText(r.Status))
		if r.Path != "" {
			fmt.Fprintf(c.OutOrStdout(), "  %s\n", safeText(r.Path))
		}
		if r.Detail != "" {
			outputParagraph(c.OutOrStdout(), "  "+r.Detail)
		}
	}
	if len(results) > 0 {
		fmt.Fprintln(c.OutOrStdout(), "Global files only. Start a new agent session or reload instructions to apply. Manual targets need setup in the agent's settings.")
	}
}
