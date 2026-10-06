//go:build !windows

package process

import (
	"os/exec"
	"syscall"
)

func configureProcess(c *exec.Cmd)       { c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }
func startProcess(c *exec.Cmd) error     { return c.Start() }
func releaseProcess(c *exec.Cmd)         {}
func terminateProcess(c *exec.Cmd) error { return syscall.Kill(-c.Process.Pid, syscall.SIGTERM) }
func forceProcess(c *exec.Cmd) error     { return syscall.Kill(-c.Process.Pid, syscall.SIGKILL) }
