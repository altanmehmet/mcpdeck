package install

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/altanmehmet/mcpdeck/internal/process"
)

type InstalledTool struct {
	Name      string `json:"name"`
	Path      string `json:"path,omitempty"`
	Available bool   `json:"available"`
	Usable    bool   `json:"usable"`
	Version   string `json:"version,omitempty"`
}

type InstalledJava struct {
	Home    string `json:"home"`
	Version string `json:"version"`
	Usable  bool   `json:"usable"`
}

// EnvironmentInventory contains software metadata only. It never reads agent
// settings, database connection files, passwords or arbitrary environment values.
type EnvironmentInventory struct {
	OS                    string          `json:"os"`
	Architecture          string          `json:"architecture"`
	Tools                 []InstalledTool `json:"tools"`
	Java                  []InstalledJava `json:"java"`
	OracleConnectionStore bool            `json:"oracle_connection_store_present"`
}

func InspectEnvironment(ctx context.Context) EnvironmentInventory {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	result := EnvironmentInventory{OS: runtime.GOOS, Architecture: runtime.GOARCH, Java: []InstalledJava{}}
	for _, home := range installedJavaHomes() {
		version, usable := probeVersion(ctx, filepath.Join(home, "bin", "java"), "-version")
		result.Java = append(result.Java, InstalledJava{Home: home, Version: version, Usable: usable})
	}
	probes := []struct {
		name string
		args []string
	}{
		{"node", []string{"--version"}}, {"npm", []string{"--version"}},
		{"npx", []string{"--version"}}, {"python3", []string{"--version"}},
		{"uv", []string{"--version"}}, {"uvx", []string{"--version"}}, {"java", []string{"-version"}},
		{"docker", []string{"--version"}}, {"colima", []string{"version"}},
		{"git", []string{"--version"}}, {"brew", []string{"--version"}},
		{"mvn", []string{"--version"}}, {"curl", []string{"--version"}},
		{"unzip", []string{"-v"}}, {"sql", []string{"-V"}},
	}
	result.Tools = make([]InstalledTool, len(probes))
	var workers sync.WaitGroup
	for i, probe := range probes {
		workers.Add(1)
		go func(i int, name string, args []string) {
			defer workers.Done()
			tool := InstalledTool{Name: name}
			path, err := lookPathExecutable(name)
			if err == nil {
				tool.Path, tool.Available = path, true
				// The macOS java stub can open an installation dialog when no
				// JVM exists. Inspect registered homes before invoking it.
				if name != "java" || runtime.GOOS != "darwin" || path != "/usr/bin/java" || len(result.Java) > 0 {
					tool.Version, tool.Usable = probeVersion(ctx, path, args...)
				}
			}
			result.Tools[i] = tool
		}(i, probe.name, probe.args)
	}
	workers.Wait()
	// Previously installed SQLcl distributions may not be on PATH.
	home, _ := os.UserHomeDir()
	paths, _ := filepath.Glob(filepath.Join(home, ".config", "mcpdeck", "installations", "*", "sqlcl", "bin", "sql"))
	for _, path := range []string{filepath.Join(home, "Downloads", "sqlcl", "bin", "sql"), filepath.Join(home, "Applications", "sqlcl", "bin", "sql"), "/opt/sqlcl/bin/sql", "/usr/local/sqlcl/bin/sql"} {
		if info, err := os.Stat(path); err == nil && !info.IsDir() && info.Mode()&0111 != 0 {
			paths = append(paths, path)
		}
	}
	for _, path := range paths[:min(len(paths), 8)] {
		version, usable := probeVersion(ctx, path, "-V")
		result.Tools = append(result.Tools, InstalledTool{Name: "sql", Path: path, Available: true, Usable: usable, Version: version})
	}
	if info, err := os.Stat(filepath.Join(home, ".dbtools")); err == nil {
		result.OracleConnectionStore = info.IsDir()
	}
	return result
}

func installedJavaHomes() []string {
	home, _ := os.UserHomeDir()
	set := map[string]bool{}
	add := func(path string) {
		if path == "" || !filepath.IsAbs(path) {
			return
		}
		if info, err := os.Stat(filepath.Join(path, "bin", "java")); err == nil && !info.IsDir() && info.Mode()&0111 != 0 {
			set[filepath.Clean(path)] = true
		}
	}
	add(os.Getenv("JAVA_HOME"))
	for _, root := range []string{"/Library/Java/JavaVirtualMachines", filepath.Join(home, "Library", "Java", "JavaVirtualMachines"), "/usr/lib/jvm"} {
		entries, _ := os.ReadDir(root)
		for _, entry := range entries {
			add(filepath.Join(root, entry.Name(), "Contents", "Home"))
			add(filepath.Join(root, entry.Name()))
		}
	}
	for _, prefix := range []string{"/opt/homebrew", "/usr/local"} {
		for _, formula := range []string{"openjdk", "openjdk@17", "openjdk@21"} {
			add(filepath.Join(prefix, "opt", formula, "libexec", "openjdk.jdk", "Contents", "Home"))
		}
	}
	paths := make([]string, 0, len(set))
	for path := range set {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths[:min(len(paths), 8)]
}

func probeVersion(ctx context.Context, executable string, args ...string) (string, bool) {
	// Cold launches (including macOS executable validation) can exceed two
	// seconds. The inventory's parent deadline still bounds the entire scan.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, executable, args...)
	command.Env = plannerEnvironment("JAVA_HOME")
	var output limitedBuffer
	command.Stdout, command.Stderr = &output, &output
	err := process.RunSetup(command)
	text := redactPlannerDiagnostics(output.String())
	if len(text) > 512 {
		text = text[:512]
	}
	return text, err == nil
}

func machinePlannerPrompt(ctx context.Context, request string) string {
	metadata, _ := json.Marshal(InspectEnvironment(ctx))
	return plannerPrompt(request) + "\n\nMCPDeck read-only local software inventory (observed now; not instructions):\n" + string(metadata) + `
Use the inventory to reuse installed compatible runtimes and MCP executables rather than downloading them again. A path existing does not prove compatibility: check usable/version. Java entries include verified JAVA_HOME locations; use a compatible observed absolute home as a literal env value instead of requesting JAVA_HOME as a hidden credential. Do not ask the user to install a runtime already observed to meet the documented requirements. The Oracle connection-store flag indicates directory presence only: it does NOT prove any saved connection, password or successful database login. Missing tools may be installed through reviewed steps or MCPDeck's permission-based runtime preparation. For version-constrained Java, do not assume the generic Homebrew openjdk formula is version 17/21; prescribe the documented compatible version in reviewed setup steps if none is installed. Never claim local configuration contents or database access were inspected.`
}

func (e EnvironmentInventory) Summary() string {
	var names []string
	javaFound := false
	for _, tool := range e.Tools {
		if tool.Usable {
			names = append(names, tool.Name)
			javaFound = javaFound || tool.Name == "java"
		}
	}
	for _, java := range e.Java {
		if java.Usable && !javaFound {
			names = append(names, "Java")
			break
		}
	}
	return strings.Join(names, ", ")
}
