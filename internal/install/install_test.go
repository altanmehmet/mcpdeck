package install

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/altanmehmet/mcpdeck/internal/testutil"
)

func TestHelper(t *testing.T) { testutil.Serve() }
func recipe() Plan            { return Plan{Version: 1, Name: "demo", Connection: `{"command":"echo","args":[]}`} }
func TestShellExecutableAliasesRejected(t *testing.T) {
	for _, command := range []string{"cmd.exe", "PowerShell.EXE", "pwsh.exe", "BASH", filepath.Join(t.TempDir(), "SH.EXE")} {
		t.Run(command, func(t *testing.T) {
			p := recipe()
			p.Steps = []Step{{Command: command}}
			if p.Validate() == nil {
				t.Fatal("shell alias accepted as an installation step")
			}
		})
	}
	for _, command := range []string{"node.exe", "uv", "cmd-helper.exe", "powershell-language-server.exe"} {
		p := recipe()
		p.Steps = []Step{{Command: command}}
		if err := p.Validate(); err != nil {
			t.Fatalf("valid non-shell command %q rejected: %v", command, err)
		}
	}
}

func TestPlanValidationAndApproval(t *testing.T) {
	p := recipe()
	for _, step := range []Step{{Command: "sh"}, {Command: "sudo"}, {Command: "echo", Directory: "../escape"}, {Command: "echo", Directory: "/tmp"}} {
		p.Steps = []Step{step}
		if p.Validate() == nil {
			t.Fatalf("accepted unsafe step %#v", step)
		}
	}
	p = recipe()
	command := "touch"
	if runtime.GOOS == "windows" {
		command = testutil.NativeExecutable(t, `package main; import "os"; func main() { if os.WriteFile(os.Args[1], nil, 0600) != nil { os.Exit(1) } }`)
	}
	p.Steps = []Step{{Command: command, Args: []string{"created"}}}
	dir := t.TempDir()
	old := p.Digest()
	p.Steps[0].Args = []string{"changed"}
	if p.Run(context.Background(), dir, old, &bytes.Buffer{}) == nil {
		t.Fatal("changed plan executed")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Fatal("unapproved mutation")
	}
	if err := p.Run(context.Background(), dir, p.Digest(), &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "changed")); err != nil {
		t.Fatal(err)
	}
	p.Manual = []string{"Sign in first"}
	if p.Preflight() == nil {
		t.Fatal("manual blocker ignored")
	}
	if p.PreflightAfterManualReview(false) == nil {
		t.Fatal("manual blocker bypassed without confirmation")
	}
	if err := p.PreflightAfterManualReview(true); err != nil {
		t.Fatal("reviewed manual prerequisites should allow preflight", err)
	}
	placeholder := p
	placeholder.Connection = `{"command":"false","args":[]}`
	if !placeholder.HasPlaceholderConnection() || placeholder.PreflightAfterManualReview(true) == nil {
		t.Fatal("manual confirmation must not make a placeholder plan executable")
	}
	manualDir := t.TempDir()
	if err := p.Run(context.Background(), manualDir, p.Digest(), &bytes.Buffer{}); err == nil {
		t.Fatal("Run bypassed manual review")
	}
	if err := p.RunAfterManualReview(context.Background(), manualDir, p.Digest(), &bytes.Buffer{}, true); err != nil {
		t.Fatal("explicit manual review should allow the approved plan", err)
	}
	raw, _ := json.Marshal(p)
	if _, err := Decode(append(raw, []byte(`{}`)...)); err == nil {
		t.Fatal("trailing JSON accepted")
	}
}
func TestBindSecretAndInstallPath(t *testing.T) {
	p := recipe()
	p.Connection = `{"command":"${INSTALL_DIR}/bin/server","args":["${INSTALL_DIR}/data"],"env":{"TOKEN":"${INSTALL_TOKEN}"}}`
	if _, err := p.Bind("/tmp/path with spaces", nil); err == nil {
		t.Fatal("missing secret accepted")
	}
	item, err := p.Bind("/tmp/path with spaces", map[string]string{"INSTALL_TOKEN": "literal$secret", "INSTALL_DIR": "/wrong"})
	if err != nil {
		t.Fatal(err)
	}
	if item.Server.Command != "/tmp/path with spaces/bin/server" || item.Server.Variables["INSTALL_TOKEN"] != "literal$secret" {
		t.Fatal("incorrect bindings")
	}
}

func TestRejectUnexecutableGeneratedPlan(t *testing.T) {
	p := recipe()
	p.Requirements = []string{"Node.js and npm must be installed"}
	if p.Validate() == nil {
		t.Fatal("prose accepted as executable")
	}
	p.Requirements = []string{"node", "npm", filepath.Join(t.TempDir(), "path with spaces", "node")}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	p.Connection = `{"command":"${NODE_EXECUTABLE}","args":[]}`
	if p.Validate() == nil {
		t.Fatal("unsupported command input accepted")
	}
}

func TestHomebrewRuntimePackageMapping(t *testing.T) {
	for executable, want := range map[string]string{
		"node": "node", "npm": "node", "npx": "node",
		"python3": "python", "pip3": "python", "uvx": "uv",
		"docker": "docker", "colima": "colima", "mvn": "maven",
		"java": "openjdk", "git": "git", "go": "go", "cargo": "rust",
	} {
		got, ok := homebrewFormula(executable)
		if !ok || got != want {
			t.Errorf("homebrewFormula(%q) = %q, %t; want %q", executable, got, ok, want)
		}
	}
	if _, ok := homebrewFormula("unknown-runtime"); ok {
		t.Fatal("unknown runtime was mapped to a guessed package")
	}
}

func TestRealMCPProbeAndFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	count, err := Probe(ctx, "demo", testutil.Config())
	if err != nil || count != 2 {
		t.Fatalf("probe %d: %v", count, err)
	}
	cfg := testutil.Config()
	cfg.Command = "/nonexistent/mcpdeck-executable"
	if _, err := Probe(ctx, "demo", cfg); err == nil {
		t.Fatal("unstartable MCP accepted")
	}
}
func TestStepSymlinkEscape(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(dir, "escape")); err != nil {
		t.Fatal(err)
	}
	p := recipe()
	p.Steps = []Step{{Command: "touch", Args: []string{"bad"}, Directory: "escape"}}
	if err := p.Run(context.Background(), dir, p.Digest(), &bytes.Buffer{}); err == nil {
		t.Fatal("symlink escape accepted")
	}
	if _, err := os.Stat(filepath.Join(outside, "bad")); !os.IsNotExist(err) {
		t.Fatal("outside write")
	}
}
func TestPlannerProtocol(t *testing.T) {
	// A fixture executable checks CLI options and returns a recipe; no model/account is contacted.
	dir := t.TempDir()
	exe := filepath.Join(dir, "planner")
	p := recipe()
	raw, _ := json.Marshal(p)
	script := `#!/bin/sh
if [ -n "$MCPDECK_PRIVATE_TEST" ]; then exit 3; fi
output=""
while [ "$#" -gt 0 ]; do
 if [ "$1" = "--output-last-message" ]; then shift; output="$1"; fi
 shift
done
cat >/dev/null
cat > "$output" <<'RECIPE'
` + string(raw) + "\nRECIPE\n"
	if err := os.WriteFile(exe, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		exe = testutil.PlannerExecutable(t, string(raw), "", 0, "", true)
	}
	t.Setenv("MCPDECK_PRIVATE_TEST", "secret")
	got, err := Generate(context.Background(), exe, "Read public docs")
	if err != nil || got.Name != p.Name {
		t.Fatal(got, err)
	}
	if strings.Contains(string(Schema()), "secret") {
		t.Fatal("secret in schema")
	}
}

func TestPlannerFailureShowsRedactedDiagnostics(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "planner")
	script := "#!/bin/sh\necho 'Operation not permitted: Authorization: Bearer secret-token-value' >&2\nexit 7\n"
	if err := os.WriteFile(exe, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		exe = testutil.PlannerExecutable(t, "", "Operation not permitted: Authorization: Bearer secret-token-value", 7, "", false)
	}
	_, err := Generate(context.Background(), exe, "Read public docs")
	if err == nil || !strings.Contains(err.Error(), "Operation not permitted") || !strings.Contains(err.Error(), "exit status 7") {
		t.Fatal("planner diagnostic missing", err)
	}
	if strings.Contains(err.Error(), "secret-token-value") {
		t.Fatal("planner diagnostic leaked a credential", err)
	}
}

func TestDockerRequirementExplainsStoppedColima(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Colima runtime is not supported on Windows")
	}
	dir := t.TempDir()
	docker := filepath.Join(dir, "docker")
	script := "#!/bin/sh\nif [ \"$1\" = info ]; then exit 1; fi\nif [ \"$1\" = context ]; then echo colima; exit 0; fi\nexit 2\n"
	if err := os.WriteFile(docker, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	p := recipe()
	p.Requirements = []string{"docker"}
	err := p.CheckRequirements()
	if err == nil || !strings.Contains(err.Error(), "Colima runtime is stopped") {
		t.Fatal("stopped Docker runtime was not explained", err)
	}
}

func TestStartColimaAfterConfirmation(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Colima startup integration is macOS-only")
	}
	dir := t.TempDir()
	marker := filepath.Join(dir, "colima-started")
	colima := filepath.Join(dir, "colima")
	colimaScript := "#!/bin/sh\n: > '" + marker + "'\n"
	if err := os.WriteFile(colima, []byte(colimaScript), 0700); err != nil {
		t.Fatal(err)
	}
	docker := filepath.Join(dir, "docker")
	dockerScript := "#!/bin/sh\nif [ \"$1\" = info ] && [ -f '" + marker + "' ]; then exit 0; fi\nif [ \"$1\" = context ]; then echo colima; exit 0; fi\nexit 1\n"
	if err := os.WriteFile(docker, []byte(dockerScript), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	if err := StartColima(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("Colima did not start after confirmation", err)
	}
}

func TestInstallStepFailureShowsRedactedDiagnostics(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "step-fixture")
	script := "#!/bin/sh\necho 'Docker pull failed: ORACLE_CONNECTION_STRING=secret-pass@db.example:1521/ORCL daemon unavailable' >&2\nexit 9\n"
	if err := os.WriteFile(exe, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		exe = testutil.PlannerExecutable(t, "", "Docker pull failed: ORACLE_CONNECTION_STRING=secret-pass@db.example:1521/ORCL daemon unavailable", 9, "", false)
	}
	p := recipe()
	p.Steps = []Step{{Command: exe, Directory: "."}}
	err := p.Run(context.Background(), t.TempDir(), p.Digest(), &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "daemon unavailable") {
		t.Fatal("installation diagnostic missing", err)
	}
	if strings.Contains(err.Error(), "secret-pass") {
		t.Fatal("installation diagnostic leaked a password", err)
	}
}
