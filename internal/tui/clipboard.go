package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/aymanbagabas/go-osc52/v2"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

type mousePoint struct{ x, y int }
type clipboardResultMsg struct{ err error }

func selectionText(view string, start, end mousePoint, width, height int) string {
	lines := strings.Split(ansi.Strip(view), "\n")
	if start.y > end.y || (start.y == end.y && start.x > end.x) {
		start, end = end, start
	}
	start.x, end.x = max(0, min(width-1, start.x)), max(0, min(width-1, end.x))
	start.y, end.y = max(0, min(height-1, start.y)), max(0, min(height-1, end.y))
	if start.y >= len(lines) || end.y >= len(lines) || (start.y == end.y && end.x < start.x) {
		return ""
	}
	selected := make([]string, 0, end.y-start.y+1)
	for y := start.y; y <= end.y; y++ {
		line := lines[y]
		left, right := 0, width
		if y == start.y {
			left = start.x
		}
		if y == end.y {
			right = end.x + 1
		}
		selected = append(selected, strings.TrimRight(ansi.Cut(line, left, right), " "))
	}
	return strings.TrimSpace(strings.Join(selected, "\n"))
}

func copyToTerminalClipboard(value string) tea.Cmd {
	return func() tea.Msg {
		if handled, err := copyNativeClipboard(value); handled {
			return clipboardResultMsg{err: err}
		}
		sequence := osc52.New(value).Limit(64 * 1024)
		if os.Getenv("TMUX") != "" {
			sequence = sequence.Tmux()
		} else if os.Getenv("STY") != "" {
			sequence = sequence.Screen()
		}
		_, err := fmt.Fprint(os.Stderr, sequence.String())
		return clipboardResultMsg{err: err}
	}
}
