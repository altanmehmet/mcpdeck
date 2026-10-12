package install

import (
	"github.com/altanmehmet/mcpdeck/internal/testutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPlannerPathFindsUserInstallAndPreservesPathPriority(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	t.Setenv("NVM_DIR", filepath.Join(home, ".nvm"))
	t.Setenv("NVM_BIN", "")
	t.Setenv("PATH", t.TempDir())
	name := "mcpdeck-planner-fixture"
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	write := func(path string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("fixture"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	fallback := filepath.Join(home, ".nvm", "versions", "node", "v24.0.0", "bin", name+suffix)
	write(fallback)
	path, err := PlannerPath(name)
	if err != nil || path != fallback {
		t.Fatal(path, err)
	}
	priority := filepath.Join(t.TempDir(), name+suffix)
	write(priority)
	t.Setenv("PATH", filepath.Dir(priority))
	path, err = PlannerPath(name)
	if err != nil || path != priority {
		t.Fatal("PATH priority changed", path, err)
	}
}
