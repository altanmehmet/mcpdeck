package process

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Recognize only npm cmd-shim's unmodified Node template, with no environment
// assignments or runtime flags. Custom batch programs keep their original path.
const nodeShimHead = "@ECHO off\r\nGOTO start\r\n:find_dp0\r\nSET dp0=%~dp0\r\nEXIT /b\r\n:start\r\nSETLOCAL\r\nCALL :find_dp0\r\n\r\nIF EXIST \"%dp0%\\node.exe\" (\r\n  SET \"_prog=%dp0%\\node.exe\"\r\n) ELSE (\r\n  SET \"_prog=node\"\r\n)\r\n\r\nendLocal & goto #_undefined_# 2>NUL || title %COMSPEC% & set PATHEXT=%PATHEXT:;.JS;=;% & \"%_prog%\"  \"%dp0%\\"
const nodeShimTail = "\" %*\r\n"

func resolveNodeShim(c *exec.Cmd) bool {
	file, err := os.Open(c.Path)
	if err != nil {
		return false
	}
	data, err := io.ReadAll(io.LimitReader(file, 65537))
	file.Close()
	if err != nil || len(data) > 65536 {
		return false
	}
	text := string(data)
	if !strings.HasPrefix(text, nodeShimHead) || !strings.HasSuffix(text, nodeShimTail) {
		return false
	}
	target := strings.TrimSuffix(strings.TrimPrefix(text, nodeShimHead), nodeShimTail)
	if target == "" || strings.ContainsAny(target, "%\"\r\n\x00") || filepath.IsAbs(target) {
		return false
	}
	script, err := filepath.Abs(filepath.Join(filepath.Dir(c.Path), target))
	if err != nil {
		return false
	}
	if info, err := os.Stat(script); err != nil || !info.Mode().IsRegular() {
		return false
	}
	runtime := filepath.Join(filepath.Dir(c.Path), "node.exe")
	if info, err := os.Stat(runtime); err == nil {
		if !info.Mode().IsRegular() {
			return false
		}
	} else if os.IsNotExist(err) {
		// Go's LookPath searches the parent environment. A custom child PATH
		// must retain batch semantics rather than silently choosing another Node.
		for _, entry := range c.Environ() {
			key, value, ok := strings.Cut(entry, "=")
			if ok && strings.EqualFold(key, "PATH") && value != os.Getenv("PATH") {
				return false
			}
		}
		runtime, err = exec.LookPath("node.exe")
		if err != nil {
			return false
		}
	} else {
		return false
	}
	runtime, err = filepath.Abs(runtime)
	if err != nil {
		return false
	}
	c.Path = runtime
	c.Args = append([]string{runtime, script}, c.Args[1:]...)
	return true
}
