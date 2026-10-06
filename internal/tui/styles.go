package tui

import "github.com/charmbracelet/lipgloss"

var title = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81"))
var selected = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231")).Background(lipgloss.Color("24"))
var muted = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
var primaryButton = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231")).Background(lipgloss.Color("24")).Padding(0, 2)
var quietButton = lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Background(lipgloss.Color("235")).Padding(0, 2)
