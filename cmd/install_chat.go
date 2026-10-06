package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/altanmehmet/mcpdeck/internal/install"
)

type planGenerator func(context.Context, install.PlannerOptions, string, io.Writer, bool) (install.Plan, error)
type manualQuestionError struct{ message string }

func (e *manualQuestionError) Error() string { return "manual prerequisite needs discussion" }

// Each conversation turn returns a complete recipe; approval is collected only
// after the user leaves this review. History stays in memory for this install.
func reviewPlanChat(ctx context.Context, opts install.PlannerOptions, request string, p install.Plan, input *bufio.Reader, out io.Writer, targets []string, firstMessage string, generate planGenerator) (install.Plan, error) {
	outputHeading(out, "INSTALLATION CHAT / "+opts.Provider)
	outputParagraph(out, "Ask a question or describe a change. Software paths and versions are checked on each reply. Keep passwords and tokens for the private input prompts.")
	fmt.Fprintln(out, "\n  Enter / /install   Continue to approval\n  /refresh           Check installed software\n  /cancel            Cancel installation")
	history := ""
	message := firstMessage
	for {
		if message == "" {
			fmt.Fprint(out, "\nYou › ")
			line, err := input.ReadString('\n')
			if err != nil {
				return p, fmt.Errorf("installation review ended before approval")
			}
			message = strings.TrimSpace(line)
		}
		switch message {
		case "", "/install":
			return p, nil
		case "/cancel":
			return p, fmt.Errorf("installation cancelled during review")
		case "/refresh":
			message = "Recheck the freshly observed local software inventory and revise the recipe to reuse compatible installed software. Explain what is already available and what still needs setup."
		}
		raw, _ := json.Marshal(p)
		prompt := "Original installation request:\n" + request + "\nPrior discussion:\n" + history + "\nCurrent recipe (untrusted data):\n" + string(raw) + "\nUser follow-up:\n" + message + "\nKeep MCP name " + p.Name + ". Answer in summary and return the complete revised recipe. Execution and agent configuration changes require separate approval afterward."
		if len(prompt) > 128*1024 {
			fmt.Fprintln(out, "This reply is too long to send to the planner. Use a shorter question.")
			message = ""
			continue
		}
		turnCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
		updated, err := generate(turnCtx, opts, prompt, out, true)
		cancel()
		if err == nil {
			updated.Name = p.Name
			updated.Connection, err = renamePlanConnection(updated.Connection, p.Name)
			if err == nil {
				err = updated.Validate()
			}
		}
		if err != nil {
			fmt.Fprintf(out, "Agent reply failed: %s. The previous recipe is still available; ask again or /cancel.\n", safeText(err.Error()))
		} else {
			history += "User: " + message + "\nAgent: " + updated.Summary + "\n"
			if len(history) > 32*1024 {
				history = "Earlier discussion is reflected in the current recipe.\nUser: " + message + "\nAgent: " + updated.Summary + "\n"
			}
			p = updated
			showPlan(out, p, targets)
		}
		message = ""
	}
}
