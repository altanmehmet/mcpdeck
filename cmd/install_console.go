package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/altanmehmet/mcpdeck/internal/install"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"
)

type consoleSessionKey struct{}
type consoleSession struct {
	ctx             context.Context
	events          chan tui.ConsoleEvent
	stopped         chan struct{}
	normal          chan string
	private         chan string
	mu              sync.Mutex
	pending, prompt string
}

func consoleSessionFor(c *cobra.Command) *consoleSession {
	if c.Context() == nil {
		return nil
	}
	session, _ := c.Context().Value(consoleSessionKey{}).(*consoleSession)
	return session
}
func installInput(c *cobra.Command) io.Reader {
	if session := consoleSessionFor(c); session != nil {
		return &consoleInput{session: session}
	}
	return c.InOrStdin()
}
func installPassword(c *cobra.Command, file *os.File) ([]byte, error) {
	if session := consoleSessionFor(c); session != nil {
		session.mu.Lock()
		prompt := session.prompt
		session.pending = ""
		session.prompt = ""
		session.mu.Unlock()
		session.send(tui.ConsoleEvent{Kind: "await", Text: prompt, Secret: true})
		select {
		case value := <-session.private:
			return []byte(value), nil
		case <-session.ctx.Done():
			return nil, session.ctx.Err()
		}
	}
	return term.ReadPassword(file.Fd())
}
func (s *consoleSession) send(event tui.ConsoleEvent) {
	select {
	case s.events <- event:
	case <-s.stopped:
	}
}

type consoleInput struct {
	session   *consoleSession
	remaining []byte
}

func (r *consoleInput) Read(buffer []byte) (int, error) {
	s := r.session
	if err := s.ctx.Err(); err != nil {
		return 0, err
	}
	if len(r.remaining) == 0 {
		s.mu.Lock()
		prompt := s.prompt
		s.pending = ""
		s.prompt = ""
		s.mu.Unlock()
		s.send(tui.ConsoleEvent{Kind: "await", Text: prompt})
		select {
		case value := <-s.normal:
			r.remaining = []byte(value + "\n")
		case <-s.ctx.Done():
			return 0, s.ctx.Err()
		}
	}
	n := copy(buffer, r.remaining)
	r.remaining = r.remaining[n:]
	return n, nil
}

// Write handles only output already safe for the existing CLI. Private values
// use a separate channel and can never become conversation events.
func (s *consoleSession) Write(raw []byte) (int, error) {
	s.mu.Lock()
	s.pending += string(raw)
	var lines []string
	for {
		index := strings.IndexByte(s.pending, '\n')
		if index < 0 {
			break
		}
		line := strings.TrimSpace(s.pending[:index])
		s.pending = s.pending[index+1:]
		if line != "" {
			lines = append(lines, line)
			s.prompt = line
		}
	}
	if strings.TrimSpace(s.pending) != "" {
		s.prompt = strings.TrimSpace(s.pending)
	}
	s.mu.Unlock()
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "Connected planning agent: "):
			s.send(tui.ConsoleEvent{Kind: "provider", Text: strings.TrimPrefix(line, "Connected planning agent: ")})
		case strings.HasPrefix(line, "Preparing a plan with "), strings.HasPrefix(line, "Preparing a repair plan with "):
			words := strings.Fields(strings.TrimPrefix(strings.TrimPrefix(line, "Preparing a plan with "), "Preparing a repair plan with "))
			if len(words) == 0 {
				continue
			}
			provider := words[0]
			s.send(tui.ConsoleEvent{Kind: "provider", Text: strings.TrimSuffix(provider, ".")})
			s.send(tui.ConsoleEvent{Kind: "status", Text: "Preparing plan"})
		case strings.HasPrefix(line, "INSTALLATION CHAT /"), strings.HasPrefix(line, "Enter / /install"), strings.HasPrefix(line, "/refresh"), strings.HasPrefix(line, "/cancel"), strings.HasPrefix(line, "Still preparing your plan"), strings.Contains(line, "is checking public documentation"), strings.HasPrefix(line, "Plan ready after"):
			// The console clock displays elapsed time without repeated progress spam.
		case strings.HasPrefix(line, "Installation step "):
			s.send(tui.ConsoleEvent{Kind: "step", Text: line})
		case strings.HasPrefix(line, "Starting the MCP"), strings.HasPrefix(line, "Starting the repaired MCP"):
			s.send(tui.ConsoleEvent{Kind: "status", Text: "Verifying connection"})
		default:
			s.send(tui.ConsoleEvent{Kind: "output", Text: line})
		}
	}
	return len(raw), nil
}
func (s *consoleSession) PresentPlan(p install.Plan, targets []string) {
	plan := tui.ConsolePlan{Name: p.Name, Summary: p.Summary, Digest: p.Digest(), Requirements: p.Requirements,
		Manual: p.Manual, FollowUp: p.FollowUp, Sources: p.Sources, Targets: append([]string(nil), targets...)}
	for _, step := range p.Steps {
		args, _ := json.Marshal(step.Args)
		plan.Steps = append(plan.Steps, tui.ConsoleStep{Description: step.Description, Command: step.Command + " " + string(args), Directory: step.Directory})
	}
	if items, err := model.Import(p.Connection, p.Name, map[string]string{"INSTALL_DIR": "/installation"}); err == nil {
		plan.Transport = items[0].Kind
		plan.Inputs = append([]string(nil), items[0].Required...)
	}
	s.send(tui.ConsoleEvent{Kind: "plan", Plan: &plan})
}

// The legacy workflow still owns recipe validation, approvals, credentials and
// mutation ordering. Only interactive presentation and input are adapted here.
func withInstallConsole(c *cobra.Command, args []string, plain bool, run func(*cobra.Command, []string) error) error {
	input, ok := c.InOrStdin().(*os.File)
	output, outputOK := c.OutOrStdout().(*os.File)
	if plain || !ok || !outputOK || !term.IsTerminal(input.Fd()) || !term.IsTerminal(output.Fd()) {
		return run(c, args)
	}
	parent := c.Context()
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	session := &consoleSession{ctx: ctx, events: make(chan tui.ConsoleEvent, 128), stopped: make(chan struct{}), normal: make(chan string, 1), private: make(chan string, 1)}
	out, stderr := c.OutOrStdout(), c.ErrOrStderr()
	c.SetOut(session)
	c.SetErr(session)
	c.SetContext(context.WithValue(ctx, consoleSessionKey{}, session))
	defer func() { c.SetOut(out); c.SetErr(stderr); c.SetContext(parent) }()
	stop := cancel
	submit := func(value string, secret bool) {
		if secret {
			select {
			case session.private <- value:
			case <-ctx.Done():
			}
			return
		}
		select {
		case session.normal <- value:
		case <-ctx.Done():
		}
	}
	request := strings.Join(args, " · ")
	ui := tui.NewInstallConsole(session.events, session.stopped, submit, stop, request)
	result := make(chan error, 1)
	go func() {
		err := run(c, args)
		result <- err
		session.send(tui.ConsoleEvent{Kind: "done", Err: err})
	}()
	_, uiErr := tea.NewProgram(ui, tea.WithInput(input), tea.WithOutput(output), tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	close(session.stopped)
	stop()
	workflowErr := <-result
	if uiErr != nil {
		return fmt.Errorf("installation console stopped: %w", uiErr)
	}
	return workflowErr
}
