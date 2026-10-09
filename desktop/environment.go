package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Finder does not normally inherit the interactive shell's package-manager PATH.
// Append known install locations without launching a shell or changing credentials.
func prepareDesktopEnvironment() {
	if runtime.GOOS != "darwin" {
		return
	}
	userHome, err := os.UserHomeDir()
	if err != nil {
		return
	}
	current := os.Getenv("PATH")
	if current == "" {
		current = "/usr/bin:/bin:/usr/sbin:/sbin"
	}
	next := desktopPath(current, []string{"/opt/homebrew/bin", "/usr/local/bin", filepath.Join(userHome, ".local", "bin"), filepath.Join(userHome, "go", "bin"), filepath.Join(userHome, ".cargo", "bin")})
	if next != os.Getenv("PATH") {
		_ = os.Setenv("PATH", next)
	}
}
func desktopPath(current string, candidates []string) string {
	paths := filepath.SplitList(current)
	seen := map[string]bool{}
	for _, p := range paths {
		seen[p] = true
	}
	for _, p := range candidates {
		if seen[p] || !filepath.IsAbs(p) {
			continue
		}
		info, err := os.Stat(p)
		if err != nil || !info.IsDir() {
			continue
		}
		paths = append(paths, p)
		seen[p] = true
	}
	return strings.Join(paths, string(os.PathListSeparator))
}
