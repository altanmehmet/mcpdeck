package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/altanmehmet/mcpdeck/internal/install"
	"github.com/altanmehmet/mcpdeck/internal/tui"
	"github.com/spf13/cobra"
)

func testConsoleSession(ctx context.Context) *consoleSession {
	return &consoleSession{ctx: ctx, events: make(chan tui.ConsoleEvent, 32), stopped: make(chan struct{}), normal: make(chan string, 1), private: make(chan string, 1)}
}
func TestConsoleLargeReplyRequestsInputOnlyOnce(t *testing.T) {
	session := testConsoleSession(context.Background())
	_, _ = session.Write([]byte("You › "))
	want := strings.Repeat("long reply ", 700)
	session.normal <- want
	input := bufio.NewReaderSize(&consoleInput{session: session}, 32)
	got, err := input.ReadString('\n')
	if err != nil || got != want+"\n" {
		t.Fatal("large reply was truncated", err)
	}
	if len(session.events) != 1 {
		t.Fatal("reader requested a second answer for one message")
	}
	if session.pending != "" {
		t.Fatal("prompt polluted subsequent output")
	}
}
func TestConsolePrivateReplyNeverEntersEvents(t *testing.T) {
	session := testConsoleSession(context.Background())
	_, _ = session.Write([]byte("API key (hidden): "))
	session.private <- "do-not-display-private-value"
	command := &cobra.Command{}
	command.SetContext(context.WithValue(context.Background(), consoleSessionKey{}, session))
	value, err := installPassword(command, nil)
	if err != nil || string(value) != "do-not-display-private-value" {
		t.Fatal("private reply unavailable", err)
	}
	event := <-session.events
	if !event.Secret || strings.Contains(event.Text, string(value)) {
		t.Fatal("private value entered event channel")
	}
	if session.pending != "" {
		t.Fatal("private prompt not consumed")
	}
}
func TestConsolePlanPresentationOmitsConnectionValues(t *testing.T) {
	session := testConsoleSession(context.Background())
	plan := install.Plan{Version: 1, Name: "demo", Summary: "Reviewed recipe", Connection: `{"command":"echo","env":{"TOKEN":"private-connection-value","INPUT":"${DEMO_PRIVATE_INPUT}"}}`}
	showPlan(session, plan, []string{"codex"})
	event := <-session.events
	raw, _ := json.Marshal(event.Plan)
	if event.Kind != "plan" || strings.Contains(string(raw), "private-connection-value") || !strings.Contains(string(raw), "DEMO_PRIVATE_INPUT") {
		t.Fatal("unsafe or incomplete plan presentation")
	}
}
func TestConsoleCancelledInputUnblocks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	session := testConsoleSession(ctx)
	result := make(chan error, 1)
	go func() { _, err := (&consoleInput{session: session}).Read(make([]byte, 32)); result <- err }()
	<-session.events
	cancel()
	select {
	case err := <-result:
		if err == nil {
			t.Fatal("cancelled reader returned success")
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled reader remained blocked")
	}
}
func TestConsoleNonTerminalPreservesLegacyWorkflow(t *testing.T) {
	command := &cobra.Command{}
	command.SetIn(strings.NewReader("input"))
	called := false
	err := withInstallConsole(command, nil, false, func(c *cobra.Command, _ []string) error {
		called = true
		if consoleSessionFor(c) != nil {
			t.Fatal("console changed non-terminal command context")
		}
		return nil
	})
	if err != nil || !called {
		t.Fatal("legacy command did not run", err)
	}
}
func TestConsoleRuntimeAndRepairEvents(t *testing.T) {
	session := testConsoleSession(context.Background())
	_, _ = session.Write([]byte("Preparing a repair plan with gemini. No files change.\nStarting the repaired MCP and verifying initialize plus tools/list...\n"))
	events := []tui.ConsoleEvent{<-session.events, <-session.events, <-session.events}
	if events[0].Kind != "provider" || events[0].Text != "gemini" || events[2].Text != "Verifying connection" {
		t.Fatal("repair progress not correctly presented", events)
	}
}
