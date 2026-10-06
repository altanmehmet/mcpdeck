package cmd

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/altanmehmet/mcpdeck/internal/install"
)

func TestInstallChatRevisesRecipeBeforeApproval(t *testing.T) {
	p := install.Plan{Version: 1, Name: "oracle", Connection: `{"command":"echo","args":[]}`, Manual: []string{"Install Java"}}
	before := p.Digest()
	var output bytes.Buffer
	calls := 0
	generate := func(ctx context.Context, opts install.PlannerOptions, request string, out io.Writer, interactive bool) (install.Plan, error) {
		calls++
		if !strings.Contains(request, "Java already exists") || !strings.Contains(request, "Current recipe") || !interactive {
			t.Fatal("follow-up did not include conversation context")
		}
		updated := p
		updated.Name = "different-name"
		updated.Summary = "Use the installed Java 21 runtime."
		updated.Manual = nil
		updated.FollowUp = []string{"Save a database connection after SQLcl installation."}
		return updated, nil
	}
	got, err := reviewPlanChat(context.Background(), install.PlannerOptions{Provider: "codex"}, "Install Oracle MCP", p, bufio.NewReader(strings.NewReader("Java already exists\n/install\n")), &output, []string{"codex", "cursor"}, "", generate)
	if err != nil || calls != 1 || got.Name != "oracle" || got.Digest() == before || len(got.Manual) != 0 {
		t.Fatalf("recipe was not correctly revised: %+v, %v", got, err)
	}
	if !strings.Contains(output.String(), "After installation:") {
		t.Fatal("post-install setup was not visible during review")
	}
}

func TestInstallChatFailureKeepsReviewedRecipe(t *testing.T) {
	p := install.Plan{Version: 1, Name: "demo", Connection: `{"command":"echo","args":[]}`}
	var output bytes.Buffer
	generate := func(context.Context, install.PlannerOptions, string, io.Writer, bool) (install.Plan, error) {
		return install.Plan{}, fmt.Errorf("provider unavailable")
	}
	got, err := reviewPlanChat(context.Background(), install.PlannerOptions{Provider: "codex"}, "Install demo", p, bufio.NewReader(strings.NewReader("Explain this\n/install\n")), &output, nil, "", generate)
	if err != nil || got.Digest() != p.Digest() || !strings.Contains(output.String(), "previous recipe") {
		t.Fatal("planner failure discarded the previously reviewed plan", err)
	}
}

func TestManualPrerequisiteQuestionReturnsToConnectedAgent(t *testing.T) {
	p := install.Plan{Manual: []string{"Confirm compatible Java"}}
	_, err := confirmManualPrerequisites(p, bufio.NewReader(strings.NewReader("Why do I need another Java install?\n")), &bytes.Buffer{}, true)
	var question *manualQuestionError
	if !errors.As(err, &question) || !strings.Contains(question.message, "Java") {
		t.Fatal("a question should return to the installation conversation")
	}
}
