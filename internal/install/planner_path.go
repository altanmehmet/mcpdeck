package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// PlannerPath preserves PATH precedence, then checks common user install locations.
// Shell aliases are not executable paths and shell startup files are never sourced.
func PlannerPath(name string) (string, error) {
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}
	home, _ := os.UserHomeDir()
	roots := []string{filepath.Join(home, ".local", "bin"), filepath.Join(home, ".npm-global", "bin"), filepath.Join(home, ".volta", "bin"), filepath.Join(home, ".bun", "bin")}
	if bin := os.Getenv("NVM_BIN"); filepath.IsAbs(bin) {
		roots = append(roots, bin)
	}
	nvm := os.Getenv("NVM_DIR")
	if nvm == "" {
		nvm = filepath.Join(home, ".nvm")
	}
	roots = append(roots, filepath.Join(nvm, "current", "bin"))
	versions, _ := filepath.Glob(filepath.Join(nvm, "versions", "node", "*", "bin"))
	if len(versions) <= 64 {
		for i := len(versions) - 1; i >= 0; i-- {
			roots = append(roots, versions[i])
		}
	}
	if runtime.GOOS == "darwin" {
		roots = append(roots, "/opt/homebrew/bin", "/usr/local/bin")
	}
	if runtime.GOOS == "windows" && os.Getenv("APPDATA") != "" {
		roots = append(roots, filepath.Join(os.Getenv("APPDATA"), "npm"))
	}
	names := []string{name}
	if runtime.GOOS == "windows" {
		names = append(names, name+".exe", name+".cmd", name+".bat")
	}
	for _, root := range roots {
		for _, file := range names {
			if path, err := exec.LookPath(filepath.Join(root, file)); err == nil {
				return path, nil
			}
		}
	}
	if name == "codex" {
		return CodexPath()
	}
	return "", fmt.Errorf("%s CLI not found", name)
}
