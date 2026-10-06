package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"os"
	"os/exec"
	"strings"
)

type installFinishedMsg struct{ err error }

func (m Model) startInstall() (tea.Model, tea.Cmd) {
	exe, err := os.Executable()
	if err != nil {
		m.status = err.Error()
		return m, nil
	}
	command := exec.Command(exe, "--config", m.store.Path, "install", "--wait")
	return m, tea.ExecProcess(command, func(err error) tea.Msg { return installFinishedMsg{err} })
}

func (m Model) startInstallRequest(name, request string) (tea.Model, tea.Cmd) {
	exe, err := os.Executable()
	if err != nil {
		m.form.message = err.Error()
		return m, nil
	}
	command := exec.Command(exe, "--config", m.store.Path, "install", "--name", name, "--wait", request)
	m.form = nil
	return m, tea.ExecProcess(command, func(err error) tea.Msg { return installFinishedMsg{err} })
}

func (m Model) startRepairRequest(name, request string) (tea.Model, tea.Cmd) {
	exe, err := os.Executable()
	if err != nil {
		m.status = err.Error()
		return m, nil
	}
	args := []string{"--config", m.store.Path, "repair", name, "--wait"}
	if strings.TrimSpace(request) != "" {
		args = append(args, request)
	}
	command := exec.Command(exe, args...)
	m.status = "Repair chat opened for " + name
	return m, tea.ExecProcess(command, func(err error) tea.Msg { return installFinishedMsg{err} })
}
