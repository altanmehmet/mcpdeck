package install

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestVersionProbeDoesNotInheritSecrets(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fixture")
	}
	path := filepath.Join(t.TempDir(), "version-tool")
	script := "#!/bin/sh\nif [ -n \"$MCPDECK_PRIVATE_TEST\" ]; then echo leaked; exit 2; fi\nprintf 'runtime 21 password=not-a-real-password\\n'\n"
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MCPDECK_PRIVATE_TEST", "secret")
	version, usable := probeVersion(context.Background(), path, "--version")
	if !usable || !strings.Contains(version, "runtime 21") || strings.Contains(version, "not-a-real-password") || strings.Contains(version, "leaked") {
		t.Fatal("probe output or inherited environment was unsafe", version)
	}
}

func TestEnvironmentReportsInstalledJavaHome(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fixture")
	}
	home := t.TempDir()
	if err := os.Mkdir(filepath.Join(home, "bin"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "bin", "java"), []byte("#!/bin/sh\necho 'openjdk version 21.fixture'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("JAVA_HOME", home)
	prompt := machinePlannerPrompt(context.Background(), "Install Oracle")
	if !strings.Contains(prompt, home) || !strings.Contains(prompt, "21.fixture") || !strings.Contains(prompt, "connection-store flag") {
		t.Fatal("planner did not receive verified Java location and limits")
	}
}

func TestVersionProbeRespectsCanceledInventory(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, usable := probeVersion(ctx, os.Args[0], "--version"); usable {
		t.Fatal("version probe ignored the inventory cancellation")
	}
}

func TestFollowUpDoesNotBlockLegacyRecipeInstallation(t *testing.T) {
	p := recipe()
	legacyDigest := p.Digest()
	decoded, err := Decode([]byte(`{"version":1,"name":"demo","connection":"{\"command\":\"echo\",\"args\":[]}","summary":"","sources":null,"requirements":null,"steps":null,"manual":null}`))
	if err != nil || decoded.Digest() != legacyDigest {
		t.Fatal("legacy recipe or approval digest changed", err)
	}
	p.FollowUp = []string{"Configure a database connection in the installed software."}
	if err := p.Preflight(); err != nil {
		t.Fatal("post-install action incorrectly blocked software installation", err)
	}
}
