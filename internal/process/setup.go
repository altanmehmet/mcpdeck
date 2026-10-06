package process

import (
	"os/exec"
	"time"
)

// RunSetup bounds pipe/process lifetime for a CommandContext installation step.
func RunSetup(cmd *exec.Cmd) error {
	configureProcess(cmd)
	cmd.Cancel = func() error { return forceProcess(cmd) }
	cmd.WaitDelay = 2 * time.Second
	err := startProcess(cmd)
	if err == nil {
		err = cmd.Wait()
	}
	// Install steps must finish before verification; do not leave background children.
	if cmd.Process != nil {
		_ = forceProcess(cmd)
	}
	return err
}
