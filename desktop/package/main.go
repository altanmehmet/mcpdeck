// package produces small, platform-specific desktop archives. No developer
// dependencies, caches, source code or fixed browser runtime enter the archive.
package main

import (
	"archive/zip"
	"crypto/sha256"
	"debug/macho"
	"debug/pe"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

const maxApplicationBytes = 35 * 1024 * 1024
const maxArchiveBytes = 20 * 1024 * 1024

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	platform := flag.String("platform", runtime.GOOS, "darwin or windows")
	arch := flag.String("arch", runtime.GOARCH, "arm64 or amd64")
	version := flag.String("version", "0.1.0", "Package version")
	source := flag.String("source", "", "Application executable or macOS .app")
	output := flag.String("output", "build/bin", "Output directory")
	flag.Parse()
	if (*platform != "darwin" && *platform != "windows") || (*arch != "arm64" && *arch != "amd64") || *source == "" {
		return fmt.Errorf("platform, architecture and application source are required")
	}
	if strings.ContainsAny(*version, "/\\\x00\r\n") || *version == "" {
		return fmt.Errorf("invalid version")
	}
	if err := os.MkdirAll(*output, 0755); err != nil {
		return err
	}
	stage, err := os.MkdirTemp("", "mcpdeck-desktop-package-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	platformName := *platform
	if platformName == "darwin" {
		platformName = "macos"
	}
	name := "MCPDeck-" + platformName + "-" + *arch
	packageDir := filepath.Join(stage, name)
	if err = os.Mkdir(packageDir, 0755); err != nil {
		return err
	}
	binary := *source
	if *platform == "darwin" {
		binary = filepath.Join(*source, "Contents", "MacOS", "MCPDeck")
		if err = copyTree(*source, filepath.Join(packageDir, "MCPDeck.app")); err != nil {
			return err
		}
	} else {
		if err = copyFile(*source, filepath.Join(packageDir, "MCPDeck.exe"), 0755); err != nil {
			return err
		}
		for _, file := range []string{"Install.cmd", "install.ps1", "uninstall.ps1"} {
			if err = copyFile(filepath.Join("installer", file), filepath.Join(packageDir, file), 0644); err != nil {
				return err
			}
		}
	}
	if err := validateBinary(binary, *platform, *arch); err != nil {
		return err
	}
	info, err := os.Stat(binary)
	if err != nil {
		return err
	}
	applicationBytes := info.Size()
	if *platform == "darwin" {
		applicationBytes, err = treeBytes(*source)
		if err != nil {
			return err
		}
	}
	if applicationBytes > maxApplicationBytes {
		return fmt.Errorf("application exceeds the 35 MiB size budget")
	}
	hash, err := hashFile(binary)
	if err != nil {
		return err
	}
	manifest := map[string]any{"application": "MCPDeck Desktop", "version": *version, "platform": *platform, "architecture": *arch, "sha256": hash, "binary_bytes": info.Size(), "application_bytes": applicationBytes}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(packageDir, "package.json"), data, 0644); err != nil {
		return err
	}
	if err = copyFile("../LICENSE", filepath.Join(packageDir, "LICENSE"), 0644); err != nil {
		return err
	}
	notices, err := licenseNotices(*platform, *arch)
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(packageDir, "THIRD_PARTY_NOTICES.txt"), notices, 0644); err != nil {
		return err
	}
	readme := "MCPDeck Desktop\n\nmacOS: extract this ZIP, then drag MCPDeck.app into Applications or ~/Applications.\nWindows: extract this ZIP and double-click Install.cmd; no administrator account is needed.\nPortable Windows: run MCPDeck.exe directly. WebView2 is required; if absent, the app offers Microsoft's small runtime installer.\nNo Go or Node is required to run MCPDeck. Selected MCP servers may need their own runtimes.\nThis package does not include a full Chromium browser. Runtime and MCP caches are separate from application size.\nOn Windows, close the app and agents using its Bridge before installing an update.\nDesktop and CLI use the same deck. Installing does not change agent configuration.\nUnsigned alpha: macOS is locally ad-hoc signed, not notarized; Windows is not Authenticode signed.\n"
	if err = os.WriteFile(filepath.Join(packageDir, "README.txt"), []byte(readme), 0644); err != nil {
		return err
	}
	archive := filepath.Join(*output, name+".zip")
	if err = zipTree(packageDir, archive); err != nil {
		return err
	}
	archiveInfo, err := os.Stat(archive)
	if err != nil {
		return err
	}
	if archiveInfo.Size() > maxArchiveBytes {
		os.Remove(archive)
		return fmt.Errorf("archive exceeds the 20 MiB size budget")
	}
	archiveHash, err := hashFile(archive)
	if err != nil {
		return err
	}
	if err = os.WriteFile(archive+".sha256", []byte(archiveHash+"  "+filepath.Base(archive)+"\n"), 0644); err != nil {
		return err
	}
	manifest["archive_bytes"] = archiveInfo.Size()
	data, _ = json.MarshalIndent(manifest, "", "  ")
	if err = os.WriteFile(archive+".size.json", data, 0644); err != nil {
		return err
	}
	fmt.Printf("%s: application %.2f MiB, download %.2f MiB\n", archive, float64(applicationBytes)/1048576, float64(archiveInfo.Size())/1048576)
	return nil
}
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func copyFile(source, target string, mode os.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	_, writeErr := io.Copy(out, in)
	closeErr := out.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}
func copyTree(source, target string) error {
	return filepath.WalkDir(source, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("linked application assets are not supported")
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		dst := filepath.Join(target, rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0755)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("non-regular application asset")
		}
		return copyFile(path, dst, info.Mode().Perm())
	})
}
func zipTree(source, target string) error {
	f, err := os.Create(target)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(f)
	err = filepath.WalkDir(source, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("archive cannot contain symlinks")
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		name, err := filepath.Rel(filepath.Dir(source), path)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(name)
		if d.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}
		w, err := zw.CreateHeader(header)
		if err != nil || d.IsDir() {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, in)
		in.Close()
		return err
	})
	closeZip := zw.Close()
	closeFile := f.Close()
	if err != nil {
		os.Remove(target)
		return err
	}
	if closeZip != nil {
		return closeZip
	}
	return closeFile
}
func licenseNotices(platform, arch string) ([]byte, error) {
	cmd := exec.Command("go", "list", "-deps", "-f", `{{if .Module}}{{if not .Module.Main}}{{.Module.Path}}|{{.Module.Version}}|{{.Module.Dir}}{{end}}{{end}}`, ".")
	cmd.Env = append(os.Environ(), "GOOS="+platform, "GOARCH="+arch)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("cannot enumerate dependency licenses: %w", err)
	}
	lines := strings.Split(string(out), "\n")
	sort.Strings(lines)
	seen := map[string]bool{}
	var b strings.Builder
	b.WriteString("MCPDeck Desktop third-party notices\n")
	goroot := runtime.GOROOT()
	runtimeLicense, err := os.ReadFile(filepath.Join(goroot, "LICENSE"))
	if err != nil {
		runtimeLicense, err = os.ReadFile(filepath.Join(goroot, "..", "LICENSE"))
	}
	if err != nil {
		return nil, err
	}
	b.WriteString("\nGo runtime and standard library\n\n")
	b.Write(runtimeLicense)
	for _, line := range lines {
		parts := strings.Split(line, "|")
		if len(parts) != 3 || seen[line] {
			continue
		}
		seen[line] = true
		b.WriteString("\n\n" + parts[0] + " " + parts[1] + "\n\n")
		found := false
		for _, name := range []string{"LICENSE", "LICENSE.txt", "LICENSE.md", "LICENSE-MIT", "LICENCE", "COPYING"} {
			raw, e := os.ReadFile(filepath.Join(parts[2], name))
			if e == nil {
				b.Write(raw)
				found = true
			}
		}
		if !found && parts[0] == "github.com/mattn/go-localereader" && parts[1] == "v0.0.1" {
			path := filepath.Join(parts[2], "README.md")
			hash, e := hashFile(path)
			if e != nil || hash != "0c5c52517f13becd7a1e1234f1e8a5ba370f5a5078cee9ac058c2d534e267fcb" {
				return nil, fmt.Errorf("upstream license declaration changed")
			}
			raw, e := os.ReadFile(path)
			if e != nil {
				return nil, e
			}
			b.Write(raw)
			found = true
		}
		if !found {
			return nil, fmt.Errorf("missing license for %s", parts[0])
		}
		if raw, e := os.ReadFile(filepath.Join(parts[2], "NOTICE")); e == nil {
			b.WriteString("\n")
			b.Write(raw)
		}
	}
	for _, module := range []string{"react", "react-dom", "@phosphor-icons/react"} {
		raw, e := os.ReadFile(filepath.Join("frontend", "node_modules", module, "LICENSE"))
		if e != nil {
			return nil, fmt.Errorf("missing frontend license: %s", module)
		}
		b.WriteString("\n\n" + module + "\n\n")
		b.Write(raw)
	}
	return []byte(b.String()), nil
}

func treeBytes(root string) (int64, error) {
	var size int64
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		size += info.Size()
		return nil
	})
	return size, err
}
func validateBinary(path, platform, arch string) error {
	if platform == "darwin" {
		f, err := macho.Open(path)
		if err != nil {
			return fmt.Errorf("source is not a Mach-O executable")
		}
		defer f.Close()
		wanted := macho.CpuArm64
		if arch == "amd64" {
			wanted = macho.CpuAmd64
		}
		if f.Cpu != wanted {
			return fmt.Errorf("macOS executable architecture does not match package")
		}
		return nil
	}
	f, err := pe.Open(path)
	if err != nil {
		return fmt.Errorf("source is not a Windows PE executable")
	}
	defer f.Close()
	wanted := uint16(pe.IMAGE_FILE_MACHINE_ARM64)
	if arch == "amd64" {
		wanted = pe.IMAGE_FILE_MACHINE_AMD64
	}
	if f.Machine != wanted {
		return fmt.Errorf("Windows executable architecture does not match package")
	}
	return nil
}
