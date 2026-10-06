package tui

import (
	"encoding/base64"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
	"reflect"
	"testing"
	"unicode/utf16"
)

func TestNativeClipboardUnicodeAndLiteralShellCharacters(t *testing.T) {
	value := "MCPDeck ç 中文 🚀\r\n%PATH% & <quoted>"
	data := nativeClipboardBytes(value)
	if binary.LittleEndian.Uint16(data[len(data)-2:]) != 0 {
		t.Fatal("UTF16 terminator missing")
	}
	var units []uint16
	for i := 0; i < len(data)-2; i += 2 {
		units = append(units, binary.LittleEndian.Uint16(data[i:]))
	}
	if !reflect.DeepEqual(utf16.Decode(units), []rune(value)) {
		t.Fatal("clipboard text changed")
	}
	t.Setenv("SSH_CONNECTION", "")
	t.Setenv("SSH_TTY", "")
	if handled, err := copyNativeClipboard("invalid\x00text"); !handled || err == nil {
		t.Fatal("NUL clipboard text accepted")
	}
	t.Setenv("SSH_CONNECTION", "fixture")
	if handled, _ := copyNativeClipboard(value); handled {
		t.Fatal("remote terminal clipboard intercepted")
	}
}

// A disposable runner permits a real clipboard check without replacing a user's clipboard.
func TestNativeClipboardOnWindowsRunner(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") != "true" {
		t.Skip("requires disposable Windows CI desktop")
	}
	t.Setenv("SSH_CONNECTION", "")
	t.Setenv("SSH_TTY", "")
	value := "MCPDeck ç 中文 🚀 %PATH% & <quoted>"
	handled, err := copyNativeClipboard(value)
	if !handled || err != nil {
		t.Fatalf("copy failed: %v", err)
	}
	directory, err := windows.GetSystemDirectory()
	if err != nil {
		t.Fatal(err)
	}
	script := "[Console]::Write([Convert]::ToBase64String([Text.Encoding]::Unicode.GetBytes((Get-Clipboard -Raw))))"
	output, err := exec.Command(filepath.Join(directory, "WindowsPowerShell", "v1.0", "powershell.exe"), "-NoProfile", "-NonInteractive", "-Command", script).Output()
	if err != nil {
		t.Fatal(err)
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(output)))
	if err != nil {
		t.Fatal(err)
	}
	expected := nativeClipboardBytes(value)
	expected = expected[:len(expected)-2]
	if !reflect.DeepEqual(data, expected) {
		t.Fatal("native clipboard text changed")
	}
}
