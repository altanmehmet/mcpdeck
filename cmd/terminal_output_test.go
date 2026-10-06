package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/altanmehmet/mcpdeck/internal/install"
)

func TestPlanOutputIsWrappedAndSeparated(t *testing.T) {
	p := install.Plan{Version: 1, Name: "demo", Summary: strings.Repeat("A readable explanation. ", 12), Connection: `{"command":"echo","args":[]}`, Steps: []install.Step{{Command: "echo", Args: []string{"safe"}, Description: "Install the MCP"}}, FollowUp: []string{"Connect your account after installation."}}
	var out bytes.Buffer
	showPlan(&out, p, []string{"codex", "cursor"})
	for _, title := range []string{"MCP / demo", "INSTALLATION STEPS", "CONNECTION", "NEXT STEPS"} {
		if !strings.Contains(out.String(), "\n"+title+"\n") {
			t.Fatalf("missing section %s", title)
		}
	}
	for _, line := range strings.Split(out.String(), "\n") {
		if ansi.StringWidth(line) > 88 {
			t.Fatalf("unwrapped line: %s", line)
		}
	}
	if strings.Contains(out.String(), "\x1b") {
		t.Fatal("color codes in redirected output")
	}
}
