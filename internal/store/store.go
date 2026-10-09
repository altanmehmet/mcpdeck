package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/altanmehmet/mcpdeck/internal/catalog"
	"github.com/altanmehmet/mcpdeck/internal/model"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Store struct {
	Path string
	// DisableDiscovery keeps isolated desktop demos inside their explicit profiles.
	DisableDiscovery bool
}

func Default() Store {
	home, _ := os.UserHomeDir()
	return Store{Path: filepath.Join(home, ".config", "mcpdeck", "deck.json")}
}
func Load() (*model.Deck, error) { return Default().Load() }
func Save(d *model.Deck) error   { return Default().Save(d) }
func Defaults() *model.Deck {
	return &model.Deck{Version: 1, Servers: catalog.Presets(), Profiles: DefaultProfiles()}
}
func DefaultProfiles() map[string]model.ProfileConfig {
	h, _ := os.UserHomeDir()
	copilotHome := os.Getenv("COPILOT_HOME")
	if strings.TrimSpace(copilotHome) == "" {
		copilotHome = filepath.Join(h, ".copilot")
	} else {
		copilotHome = ExpandUserPath(copilotHome, h)
	}
	claude := filepath.Join(h, ".config", "Claude", "claude_desktop_config.json")
	if runtime.GOOS == "darwin" {
		claude = filepath.Join(h, "Library", "Application Support", "Claude", "claude_desktop_config.json")
	}
	if runtime.GOOS == "windows" {
		claude = filepath.Join(os.Getenv("APPDATA"), "Claude", "claude_desktop_config.json")
	}
	code := filepath.Join(h, ".config", "Code", "User", "mcp.json")
	if runtime.GOOS == "darwin" {
		code = filepath.Join(h, "Library", "Application Support", "Code", "User", "mcp.json")
	}
	if runtime.GOOS == "windows" {
		code = filepath.Join(os.Getenv("APPDATA"), "Code", "User", "mcp.json")
	}
	codexDir := os.Getenv("CODEX_HOME")
	if codexDir == "" {
		codexDir = filepath.Join(h, ".codex")
	}
	return map[string]model.ProfileConfig{
		"cursor":      {TargetPath: filepath.Join(h, ".cursor", "mcp.json"), EnabledServers: []string{}, RemoteFormat: "url"},
		"claude":      {TargetPath: claude, EnabledServers: []string{}, RemoteFormat: "stdio"},
		"windsurf":    {TargetPath: filepath.Join(h, ".codeium", "windsurf", "mcp_config.json"), EnabledServers: []string{}, RemoteFormat: "serverUrl"},
		"claude-code": {TargetPath: filepath.Join(h, ".claude.json"), EnabledServers: []string{}, RemoteFormat: "http"},
		"codex":       {TargetPath: filepath.Join(codexDir, "config.toml"), EnabledServers: []string{}, Format: "codex"},
		"copilot":     {TargetPath: code, EnabledServers: []string{}, Format: "vscode"},
		"copilot-cli": {TargetPath: filepath.Join(copilotHome, "mcp-config.json"), EnabledServers: []string{}, Format: "copilot-cli"},
		"antigravity": {TargetPath: filepath.Join(h, ".gemini", "config", "mcp_config.json"), EnabledServers: []string{}, RemoteFormat: "serverUrl"},
	}
}

func ExpandUserPath(path, home string) string {
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return filepath.Clean(path)
}

// AdditionalProfiles returns adapters that are discovered only when their app
// or MCP configuration is present. This keeps the built-in profile set stable
// while letting newly supported clients join existing decks automatically.
func AdditionalProfiles() map[string]model.ProfileConfig {
	h, _ := os.UserHomeDir()
	zedPath := filepath.Join(h, ".config", "zed", "settings.json")
	if runtime.GOOS == "darwin" {
		zedPath = filepath.Join(h, ".zed", "settings.json")
	} else if runtime.GOOS == "windows" {
		zedPath = filepath.Join(os.Getenv("APPDATA"), "Zed", "settings.json")
	} else if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		zedPath = filepath.Join(xdg, "zed", "settings.json")
	}
	profiles := map[string]model.ProfileConfig{
		"gemini-cli":   {TargetPath: filepath.Join(h, ".gemini", "settings.json"), EnabledServers: []string{}, Format: "gemini-cli", RemoteFormat: "url"},
		"opencode":     {TargetPath: filepath.Join(h, ".config", "opencode", "opencode.json"), EnabledServers: []string{}, Format: "opencode", RemoteFormat: "url"},
		"zed":          {TargetPath: zedPath, EnabledServers: []string{}, Format: "zed", RemoteFormat: "url"},
		"cline":        {TargetPath: filepath.Join(h, ".cline", "mcp.json"), EnabledServers: []string{}, Format: "cline", RemoteFormat: "http"},
		"continue":     {TargetPath: filepath.Join(h, ".continue", "mcpServers", "mcpdeck.json"), EnabledServers: []string{}, Format: "continue", RemoteFormat: "http"},
		"amazon-q":     {TargetPath: filepath.Join(h, ".aws", "amazonq", "mcp.json"), EnabledServers: []string{}, RemoteFormat: "http"},
		"amazon-q-ide": {TargetPath: filepath.Join(h, ".aws", "amazonq", "agents", "default.json"), EnabledServers: []string{}, RemoteFormat: "http"},
		"kiro":         {TargetPath: filepath.Join(h, ".kiro", "settings", "mcp.json"), EnabledServers: []string{}, RemoteFormat: "url"},
		"qwen-code":    {TargetPath: filepath.Join(h, ".qwen", "settings.json"), EnabledServers: []string{}, Format: "qwen", RemoteFormat: "url"},
	}
	traePath := os.Getenv("TRAE_MCP_CONFIG")
	if strings.TrimSpace(traePath) == "" {
		if cwd, err := os.Getwd(); err == nil {
			traePath = filepath.Join(cwd, ".trae", "mcp.json")
		}
	} else {
		traePath = ExpandUserPath(traePath, h)
	}
	if traePath != "" {
		if info, err := os.Stat(filepath.Dir(traePath)); err == nil && info.IsDir() {
			profiles["trae"] = model.ProfileConfig{TargetPath: traePath, EnabledServers: []string{}, Format: "trae", RemoteFormat: "url"}
		}
	}
	for key, path := range extensionMCPConfigs("rooveterinaryinc.roo-cline", "mcp_settings.json", "roo-code") {
		profiles[key] = model.ProfileConfig{TargetPath: path, EnabledServers: []string{}, Format: "roo-code", RemoteFormat: "http"}
	}
	for key, path := range extensionMCPConfigs("saoudrizwan.claude-dev", "cline_mcp_settings.json", "cline-vscode") {
		profiles[key] = model.ProfileConfig{TargetPath: path, EnabledServers: []string{}, Format: "cline", RemoteFormat: "http"}
	}
	return profiles
}

func extensionMCPConfigs(extensionID, filename, keyPrefix string) map[string]string {
	h, _ := os.UserHomeDir()
	roots := map[string]string{}
	switch runtime.GOOS {
	case "darwin":
		base := filepath.Join(h, "Library", "Application Support")
		for editor, slug := range map[string]string{"Code": "vscode", "Code - Insiders": "vscode-insiders", "Cursor": "cursor", "Windsurf": "windsurf", "VSCodium": "vscodium"} {
			roots[slug] = filepath.Join(base, editor, "User", "globalStorage")
		}
	case "windows":
		base := os.Getenv("APPDATA")
		for editor, slug := range map[string]string{"Code": "vscode", "Code - Insiders": "vscode-insiders", "Cursor": "cursor", "Windsurf": "windsurf", "VSCodium": "vscodium"} {
			roots[slug] = filepath.Join(base, editor, "User", "globalStorage")
		}
	default:
		base := os.Getenv("XDG_CONFIG_HOME")
		if base == "" {
			base = filepath.Join(h, ".config")
		}
		for editor, slug := range map[string]string{"Code": "vscode", "Code - Insiders": "vscode-insiders", "Cursor": "cursor", "Windsurf": "windsurf", "VSCodium": "vscodium"} {
			roots[slug] = filepath.Join(base, editor, "User", "globalStorage")
		}
	}
	profiles := map[string]string{}
	for slug, root := range roots {
		dir := filepath.Join(root, extensionID, "settings")
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			profiles[keyPrefix+"-"+slug] = filepath.Join(dir, filename)
		}
	}
	return profiles
}

// IsInstalledAgent recognizes a supported agent by its standard config file,
// executable, or macOS application bundle. Custom profile paths are never
// treated as a detected built-in client.
func IsInstalledAgent(key string, p model.ProfileConfig) bool {
	profiles := DefaultProfiles()
	for name, candidate := range AdditionalProfiles() {
		profiles[name] = candidate
	}
	base, ok := profiles[key]
	if !ok || p.TargetPath != base.TargetPath {
		return false
	}
	if info, err := os.Stat(p.TargetPath); err == nil && !info.IsDir() {
		return true
	}
	if key == "continue" {
		home, _ := os.UserHomeDir()
		if info, err := os.Stat(filepath.Join(home, ".continue")); err == nil && info.IsDir() {
			return true
		}
	}
	if strings.HasPrefix(key, "roo-code-") || strings.HasPrefix(key, "cline-vscode-") {
		extensionDir := filepath.Dir(filepath.Dir(p.TargetPath))
		if info, err := os.Stat(extensionDir); err == nil && info.IsDir() {
			return true
		}
	}
	executables := map[string]string{
		"cursor": "cursor", "claude-code": "claude", "codex": "codex",
		"copilot": "code", "copilot-cli": "copilot", "windsurf": "windsurf",
		"antigravity": "antigravity", "gemini-cli": "gemini", "opencode": "opencode", "zed": "zed",
		"cline": "cline", "continue": "cn", "amazon-q": "q", "amazon-q-ide": "q", "kiro": "kiro-cli", "qwen-code": "qwen",
	}
	if binary := executables[key]; binary != "" {
		if _, err := exec.LookPath(binary); err == nil {
			return true
		}
	}
	apps := map[string][]string{
		"cursor": {"Cursor.app"}, "claude": {"Claude.app"},
		"codex": {"Codex.app", "ChatGPT.app"}, "copilot": {"Visual Studio Code.app"},
		"windsurf": {"Windsurf.app"}, "antigravity": {"Antigravity.app"},
		"opencode": {"OpenCode.app"}, "kiro": {"Kiro.app"}, "zed": {"Zed.app"},
	}
	if runtime.GOOS == "darwin" {
		home, _ := os.UserHomeDir()
		for _, app := range apps[key] {
			for _, root := range []string{"/Applications", filepath.Join(home, "Applications")} {
				if info, err := os.Stat(filepath.Join(root, app)); err == nil && info.IsDir() {
					return true
				}
			}
		}
	}
	return false
}

// AgentDetected reports whether a known client is installed or a configured
// custom profile already has a target file.
func AgentDetected(key string, p model.ProfileConfig) bool {
	if IsInstalledAgent(key, p) {
		return true
	}
	path := p.TargetPath
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[2:])
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func discoverInstalledProfiles(d *model.Deck) {
	if d.Profiles == nil {
		return
	}
	for key, profile := range AdditionalProfiles() {
		if _, exists := d.Profiles[key]; exists || !IsInstalledAgent(key, profile) {
			continue
		}
		d.Profiles[key] = profile
	}
	// Migrate the old Copilot CLI default when COPILOT_HOME is now set. A
	// user-supplied custom target is left untouched.
	if profile, ok := d.Profiles["copilot-cli"]; ok {
		home, _ := os.UserHomeDir()
		legacy := filepath.Join(home, ".copilot", "mcp-config.json")
		current := DefaultProfiles()["copilot-cli"].TargetPath
		if profile.TargetPath == legacy && current != legacy {
			profile.TargetPath = current
			d.Profiles["copilot-cli"] = profile
		}
	}
}
func (s Store) Load() (*model.Deck, error) {
	b, err := os.ReadFile(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		d := Defaults()
		if !s.DisableDiscovery {
			discoverInstalledProfiles(d)
		}
		return d, nil
	}
	if err != nil {
		return nil, err
	}
	if err = ensurePrivateFile(s.Path); err != nil {
		return nil, fmt.Errorf("deck permissions are too open; cannot secure %s", s.Path)
	}
	var d model.Deck
	if err = json.Unmarshal(b, &d); err != nil {
		return nil, fmt.Errorf("invalid deck JSON: %w", err)
	}
	if !s.DisableDiscovery {
		discoverInstalledProfiles(&d)
	}
	if err = d.Validate(); err != nil {
		return nil, err
	}
	return &d, nil
}

// Update reloads under a cross-process lock, preventing cache writes from overwriting UI changes.
func (s Store) Update(fn func(*model.Deck) error) error {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0700); err != nil {
		return err
	}
	lock := s.Path + ".lock"
	deadline := time.Now().Add(3 * time.Second)
	for {
		err := os.Mkdir(lock, 0700)
		if err == nil {
			break
		}
		if !errors.Is(err, os.ErrExist) {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("deck is locked: %s (remove only if no mcpdeck writer is running)", lock)
		}
		time.Sleep(25 * time.Millisecond)
	}
	defer os.Remove(lock)
	d, err := s.Load()
	if err != nil {
		return err
	}
	if err = fn(d); err != nil {
		return err
	}
	if err = d.Validate(); err != nil {
		return err
	}
	b, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	return AtomicWrite(s.Path, append(b, '\n'))
}
func (s Store) Save(d *model.Deck) error {
	return s.Update(func(current *model.Deck) error { *current = *d; return nil })
}
func AtomicWrite(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".mcpdeck-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if err = ensurePrivateFile(f.Name()); err != nil {
		return err
	}
	if _, err = f.Write(b); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
