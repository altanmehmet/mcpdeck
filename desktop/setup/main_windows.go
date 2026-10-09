//go:build windows

package main

import (
	"context"
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

func message(text string, flags uint32) int32 {
	body, _ := windows.UTF16PtrFromString(text)
	title, _ := windows.UTF16PtrFromString("MCPDeck Setup")
	answer, _ := windows.MessageBox(0, body, title, flags)
	return answer
}
func main() {
	verifyOnly := len(os.Args) == 2 && os.Args[1] == "--verify"
	if len(os.Args) > 1 && !verifyOnly {
		message("Unknown setup option.", 0x10)
		os.Exit(1)
	}
	if !verifyOnly && message("Install MCPDeck for your user account?\n\nA Start menu shortcut will be created. No administrator account is required. Close MCPDeck and any agents using its Bridge before updating.\n\nThe application opens after installation.", 0x24) != 6 {
		return
	}
	stage, err := os.MkdirTemp("", "mcpdeck-setup-")
	if err != nil {
		message("Cannot prepare temporary setup files.", 0x10)
		os.Exit(1)
	}
	defer os.RemoveAll(stage)
	root, err := extractPackage(payload, stage, runtime.GOARCH)
	if err == nil && !verifyOnly {
		systemDir, e := windows.GetSystemDirectory()
		if e != nil {
			err = e
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			command := exec.CommandContext(ctx, filepath.Join(systemDir, "WindowsPowerShell", "v1.0", "powershell.exe"), "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", filepath.Join(root, "install.ps1"), "-NoLaunch")
			command.WaitDelay = 2 * time.Second
			command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
			output, e := command.CombinedOutput()
			cancel()
			if e != nil {
				detail := strings.TrimSpace(string(output))
				if len(detail) > 1500 {
					detail = detail[:1500]
				}
				err = fmt.Errorf("Close MCPDeck and its Bridge connections, then retry.\n\n%s", detail)
			}
		}
	}
	if err != nil {
		if !verifyOnly {
			message("Installation could not finish.\n\n"+err.Error(), 0x10)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if verifyOnly {
		fmt.Println("Installer payload verified.")
		return
	}
	prefix := filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "MCPDeck")
	application := exec.Command(filepath.Join(prefix, "MCPDeck.exe"))
	application.Dir = prefix
	if e := application.Start(); e != nil {
		message("MCPDeck was installed. Open it from the Start menu.", 0x40)
	} else {
		_ = application.Process.Release()
	}

}
