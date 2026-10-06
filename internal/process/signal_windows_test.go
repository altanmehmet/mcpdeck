package process

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/altanmehmet/mcpdeck/internal/model"
	"github.com/altanmehmet/mcpdeck/internal/testutil"
	"golang.org/x/sys/windows"
)

func TestWindowsBatchWithSpacesAndInjectionRejection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fixture with spaces.cmd")
	if err := os.WriteFile(path, []byte("@echo off\r\necho %~1\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "argument with spaces")
	configureProcess(cmd)
	output, err := cmd.Output()
	if err != nil || string(output) != "argument with spaces\r\n" {
		t.Fatalf("batch launch: %q %v", output, err)
	}
	for _, argument := range []string{"x&echo injected", "%PATH%", "x\"y", "x\ny", "!variable!"} {
		cmd := exec.CommandContext(ctx, path, argument)
		configureProcess(cmd)
		if cmd.Err == nil {
			t.Fatal("shell expansion accepted")
		}
	}
}

func TestWindowsSetupTerminatesDescendants(t *testing.T) {
	executable := testutil.NativeExecutable(t, `package main
import("os"; "os/exec"; "strconv"; "time")
func main() {
 if os.Args[1] == "child" { for { time.Sleep(time.Second) } }
 child := exec.Command(os.Args[0], "child")
 child.Stdout, child.Stderr = os.Stdout, os.Stderr
 if child.Start() != nil { os.Exit(1) }
 if os.WriteFile(os.Args[2], []byte(strconv.Itoa(child.Process.Pid)), 0600) != nil { os.Exit(2) }
 if os.Args[1] == "exit" { return }
 for { time.Sleep(time.Second) }
}`)
	for _, mode := range []string{"cancel", "exit"} {
		t.Run(mode, func(t *testing.T) {
			marker := filepath.Join(t.TempDir(), "child.pid")
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, executable, mode, marker)
			cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
			done := make(chan error, 1)
			go func() { done <- RunSetup(cmd) }()
			var pid int
			deadline := time.Now().Add(5 * time.Second)
			for time.Now().Before(deadline) {
				if raw, err := os.ReadFile(marker); err == nil {
					pid, _ = strconv.Atoi(strings.TrimSpace(string(raw)))
					if pid > 0 {
						break
					}
				}
				time.Sleep(10 * time.Millisecond)
			}
			if pid == 0 {
				cancel()
				<-done
				t.Fatal("descendant did not start")
			}
			child, err := windows.OpenProcess(windows.SYNCHRONIZE|windows.PROCESS_TERMINATE, false, uint32(pid))
			if err != nil {
				t.Fatal(err)
			}
			defer windows.CloseHandle(child)
			defer windows.TerminateProcess(child, 1) // Fixture-only cleanup on failure.
			if mode == "cancel" {
				cancel()
			}
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				cancel()
				t.Fatal("setup did not finish")
			}
			state, err := windows.WaitForSingleObject(child, 2000)
			if err != nil || state != windows.WAIT_OBJECT_0 {
				t.Fatalf("descendant survived: %d %v", state, err)
			}
		})
	}
}

func TestWindowsNodeShimPreservesComplexArguments(t *testing.T) {
	executable := testutil.NativeExecutable(t, `package main
import("encoding/json"; "os")
func main() { json.NewEncoder(os.Stdout).Encode(os.Args[1:]) }
`)
	dir := t.TempDir()
	data, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "node.exe"), data, 0600); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "cli.js")
	if err := os.WriteFile(script, []byte("// fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	shim := filepath.Join(dir, "fixture.cmd")
	// Independent npm-generated fixture: changing recognition must not silently
	// change the test input along with the implementation.
	template := strings.ReplaceAll(`@ECHO off
GOTO start
:find_dp0
SET dp0=%~dp0
EXIT /b
:start
SETLOCAL
CALL :find_dp0

IF EXIST "%dp0%\node.exe" (
  SET "_prog=%dp0%\node.exe"
) ELSE (
  SET "_prog=node"
)

endLocal & goto #_undefined_# 2>NUL || title %COMSPEC% & set PATHEXT=%PATHEXT:;.JS;=;% & "%_prog%"  "%dp0%\cli.js" %*
`, "\n", "\r\n")
	if err := os.WriteFile(shim, []byte(template), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	args := []string{`{"url":"https://example.test?a=1&b=2"}`, "%PATH%", "!literal!", "中文 🚀", "line\nnext"}
	cmd := exec.CommandContext(ctx, shim, args...)
	configureProcess(cmd)
	output, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	var actual []string
	if err := json.Unmarshal(output, &actual); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(actual, append([]string{script}, args...)) {
		t.Fatal("Node shim arguments changed")
	}
	// An explicit child PATH must not be replaced by the parent's Node runtime.
	custom := filepath.Join(dir, "custom-runtime")
	parent := filepath.Join(dir, "parent-runtime")
	for _, directory := range []string{custom, parent} {
		if err := os.Mkdir(directory, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, "node.exe"), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Remove(filepath.Join(dir, "node.exe")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", parent)
	cmd = exec.CommandContext(ctx, shim, args...)
	cmd.Env = append(os.Environ(), "PATH="+custom)
	configureProcess(cmd)
	if cmd.Err == nil {
		t.Fatal("custom child PATH was ignored during Node resolution")
	}
	manager := New(&model.Deck{Servers: map[string]model.ServerConfig{
		"custom": {Command: shim, Args: args, Env: map[string]string{"PATH": custom}},
	}}, time.Minute)
	defer manager.KillAll()
	if _, err := manager.GetOrSpawn("custom"); err == nil {
		t.Fatal("manager resolved Node before applying its custom environment")
	}
	// Ordinary custom-PATH arguments continue to use the original batch program.
	cmd = exec.CommandContext(ctx, shim, "safe")
	cmd.Env = append(os.Environ(), "PATH="+custom)
	configureProcess(cmd)
	output, err = cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(output, &actual); err != nil {
		t.Fatal(err)
	}
	if len(actual) != 2 || actual[1] != "safe" {
		t.Fatal("custom-PATH batch launch changed")
	}
	// Additional commands must prevent recognition; unsafe arguments stay rejected.
	if err := os.WriteFile(shim, []byte("echo custom\r\n"+template), 0600); err != nil {
		t.Fatal(err)
	}
	cmd = exec.CommandContext(ctx, shim, args...)
	configureProcess(cmd)
	if cmd.Err == nil {
		t.Fatal("modified batch shim was treated as a Node executable")
	}
}
