// Package install implements reviewed installation recipes. Planning never executes a recipe.
package install

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/altanmehmet/mcpdeck/internal/model"
)

var executableName = regexp.MustCompile(`^[A-Za-z0-9_.+-]+$`)
var commandVariable = regexp.MustCompile(`\$\{[^}]+\}|\$[A-Za-z_][A-Za-z0-9_]*`)

type Step struct {
	Description string   `json:"description"`
	Command     string   `json:"command"`
	Args        []string `json:"args"`
	Directory   string   `json:"directory"`
}

type Plan struct {
	Version      int      `json:"version"`
	Name         string   `json:"name"`
	Summary      string   `json:"summary"`
	Sources      []string `json:"sources"`
	Requirements []string `json:"requirements"`
	Steps        []Step   `json:"steps"`
	Connection   string   `json:"connection"`
	Manual       []string `json:"manual"`
	FollowUp     []string `json:"follow_up,omitempty"`
}

func Decode(raw []byte) (Plan, error) {
	var p Plan
	if len(raw) > 1024*1024 {
		return p, fmt.Errorf("plan exceeds 1 MiB")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if dec.Decode(&p) != nil {
		return p, fmt.Errorf("invalid installation plan")
	}
	var extra any
	if dec.Decode(&extra) != io.EOF {
		return p, fmt.Errorf("expected one installation plan")
	}
	return p, p.Validate()
}

func (p Plan) Validate() error {
	if p.Version != 1 || len(p.Steps) > 32 {
		return fmt.Errorf("unsupported plan version or too many steps")
	}
	if p.Name == "mcpdeck" || !model.NamePattern.MatchString(p.Name) || strings.Contains(p.Name, "__") || len(p.Name) > 40 {
		return fmt.Errorf("invalid installation name")
	}
	for _, required := range p.Requirements {
		if strings.ContainsAny(required, "\x00\r\n") || (!filepath.IsAbs(required) && !executableName.MatchString(required)) {
			return fmt.Errorf("requirements must contain executable names or absolute executable paths, not descriptions")
		}
	}
	for _, s := range p.Steps {
		if s.Command == "" || strings.ContainsAny(s.Command, "\x00\r\n") {
			return fmt.Errorf("invalid step executable")
		}
		// Windows shell aliases are the same prohibited executables even when
		// their extension or casing differs from the recipe's command name.
		commandName := strings.TrimSuffix(strings.ToLower(filepath.Base(s.Command)), ".exe")
		switch commandName {
		case "sudo", "su", "sh", "bash", "zsh", "fish", "cmd", "powershell", "pwsh":
			return fmt.Errorf("shell and privileged steps are unsupported; use an executable with separate arguments")
		}
		dir := filepath.Clean(s.Directory)
		if filepath.IsAbs(dir) || filepath.VolumeName(dir) != "" || strings.HasPrefix(dir, "/") || (runtime.GOOS == "windows" && strings.HasPrefix(dir, `\`)) || dir == ".." || strings.HasPrefix(dir, ".."+string(filepath.Separator)) {
			return fmt.Errorf("step directory must remain inside installation directory")
		}
	}
	items, err := model.Import(p.Connection, p.Name, map[string]string{"INSTALL_DIR": "/mcpdeck-install"})
	if err != nil {
		return fmt.Errorf("invalid planned connection: %w", err)
	}
	if len(items) != 1 || items[0].Name != p.Name || items[0].Disabled {
		return fmt.Errorf("plan must define one enabled connection matching its name")
	}
	command := strings.ReplaceAll(items[0].Server.Command, "${INSTALL_DIR}", "/installation")
	if commandVariable.MatchString(command) {
		return fmt.Errorf("connection command must be a literal executable; only ${INSTALL_DIR} is supported there")
	}
	return nil
}

// Digest binds approval to the exact recipe, including commands and documentation sources.
func (p Plan) Digest() string {
	b, _ := json.Marshal(p)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func (p Plan) Bind(dir string, values map[string]string) (model.ImportedServer, error) {
	bindings := map[string]string{"INSTALL_DIR": dir}
	for k, v := range values {
		if k != "INSTALL_DIR" {
			bindings[k] = v
		}
	}
	items, err := model.Import(p.Connection, p.Name, bindings)
	if err != nil {
		return model.ImportedServer{}, err
	}
	item := items[0]
	if len(item.Required) > 0 {
		return item, fmt.Errorf("missing inputs: %s", strings.Join(item.Required, ", "))
	}
	// Commands are normally literal in MCPDeck; only our reserved installation path expands here.
	item.Server.Command = strings.ReplaceAll(item.Server.Command, "${INSTALL_DIR}", dir)
	return item, nil
}
