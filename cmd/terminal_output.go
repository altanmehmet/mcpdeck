package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
)

func confirmYes(input *bufio.Reader, out io.Writer, label string) bool {
	fmt.Fprint(out, label+" [y/N]: ")
	answer, err := input.ReadString('\n')
	if err != nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		return true
	}
	return false
}

// Plain output stays readable in logs; terminal output gets a restrained accent.
func outputWidth(w io.Writer) int {
	if file, ok := w.(*os.File); ok && term.IsTerminal(file.Fd()) {
		if width, _, err := term.GetSize(file.Fd()); err == nil {
			return max(30, min(100, width-4))
		}
	}
	return 88
}
func outputHeading(w io.Writer, label string) {
	label = safeText(label)
	if file, ok := w.(*os.File); ok && term.IsTerminal(file.Fd()) && os.Getenv("NO_COLOR") == "" {
		label = "\x1b[1;36m" + label + "\x1b[0m"
	}
	fmt.Fprintf(w, "\n%s\n", label)
}
func outputParagraph(w io.Writer, text string) {
	fmt.Fprintln(w, ansi.Wrap(safeText(text), outputWidth(w), ""))
}
