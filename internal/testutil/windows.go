package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
)

func SetHome(t *testing.T, home string) {
	t.Helper()
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
		t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
		t.Setenv("LOCALAPPDATA", filepath.Join(home, "AppData", "Local"))
	}
}

// NativeExecutable builds a small, account-free fixture with the current Go
// toolchain. This avoids depending on a POSIX shell on native Windows runners.
func NativeExecutable(t *testing.T, source string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "fixture.go")
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(dir, "fixture.exe")
	goBinary := filepath.Join(runtime.GOROOT(), "bin", "go.exe")
	if output, err := exec.Command(goBinary, "build", "-o", executable, path).CombinedOutput(); err != nil {
		t.Fatalf("build native fixture: %s %v", output, err)
	}
	return executable
}

func PlannerExecutable(t *testing.T, output, diagnostic string, exitCode int, marker string, codex bool) string {
	t.Helper()
	return NativeExecutable(t, `package main
import("fmt"; "io"; "os"; "strings")
func main() {
 if os.Getenv("MCPDECK_PRIVATE_TEST") != "" || os.Getenv("MCPDECK_INSTALL_PASSWORD") != "" { os.Exit(3) }
 for i, arg := range os.Args { if arg == "--model" && (i+1 == len(os.Args) || os.Args[i+1] != "fixture-model") { os.Exit(5) } }
 input, _ := io.ReadAll(os.Stdin)
 if !strings.Contains(string(input), `+strconv.Quote(marker)+`) { os.Exit(4) }
 output := `+strconv.Quote(output)+`
 if `+strconv.FormatBool(codex)+` {
  schema, sandbox, approval := false, false, false
  for i, arg := range os.Args {
   if arg == "--output-schema" && i+1 < len(os.Args) { _, err := os.Stat(os.Args[i+1]); schema = err == nil }
   if arg == "--sandbox" && i+1 < len(os.Args) { sandbox = os.Args[i+1] == "read-only" }
   if arg == "-c" && i+1 < len(os.Args) { approval = approval || os.Args[i+1] == "approval_policy=never" }
  }
  if !schema || !sandbox || !approval { os.Exit(6) }
  for i, arg := range os.Args { if arg == "--output-last-message" && i+1 < len(os.Args) { if os.WriteFile(os.Args[i+1], []byte(output), 0600) != nil { os.Exit(3) }; output = "" } }
 }
 fmt.Print(output)
 fmt.Fprint(os.Stderr, `+strconv.Quote(diagnostic)+`)
 os.Exit(`+strconv.Itoa(exitCode)+`)
}`)
}
