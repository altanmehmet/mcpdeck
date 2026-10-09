# MCPDeck

Manage MCP servers and shared personal instructions across coding agents from one terminal.

**Open source · MIT · alpha software.** Works on macOS, Linux and Windows.
MCPDeck runs locally: no hosted dashboard or application database is required.

## Why use it?

If you use several coding agents, their MCP configuration and personal rules can drift.
MCPDeck provides one terminal panel to discover, install, enable, disable, update and remove
MCP servers, and distribute shared instructions while preserving existing personal text.

- **Natural-language installation:** enter an MCP name and describe what you need. A connected
  planning agent researches a recipe; you review its commands before execution.
- **Multiple agents:** synchronize supported configuration formats, inspect existing MCPs,
  and remove externally configured entries with backups and change checks.
- **Shared instructions:** view existing global files, edit one agent, or distribute one
  shared text. Project instructions retain their scope.
- **Recovery:** inspect partial sync failures, retry failed targets, and restore backups.
- **Terminal controls:** mouse and keyboard navigation, configurable shortcuts, and clipboard support.

## Install

### macOS / Linux with Homebrew

```sh
brew install altanmehmet/mcpdeck/mcpdeck
mcpdeck
```

To update: `brew update` followed by `brew upgrade altanmehmet/mcpdeck/mcpdeck`.
Packages and the binary formula live in [homebrew-mcpdeck](https://github.com/altanmehmet/homebrew-mcpdeck).

### Windows

Open PowerShell or Windows Terminal:

```powershell
irm https://raw.githubusercontent.com/altanmehmet/homebrew-mcpdeck/main/install.ps1 | iex
mcpdeck
```

The bootstrap downloads a pinned x64/ARM64 ZIP, verifies its checksum, and updates your
user PATH. No WSL, Go or administrator privileges are needed.
[Windows installation, removal and test coverage](docs/WINDOWS.md).

### From source

Requires Go 1.24 or newer:

```sh
git clone https://github.com/altanmehmet/mcpdeck.git
cd mcpdeck
go build -trimpath -o mcpdeck .
sh scripts/install.sh
mcpdeck
```

## First run

1. Open `mcpdeck`. It detects known agent configuration locations.
2. Choose **New MCP** (`n`), enter a name and describe the connection you want.
3. Select an available planner: Codex, Claude, Gemini, or a supported API provider.
4. Review sources, commands and target agents. Approve with `y` when satisfied.
5. Enter private connection values only in the hidden input prompts.
6. MCPDeck installs the recipe, checks MCP initialize/tool discovery, then syncs settings.
   Start a new agent session to load the configuration; account/tool access needs a real tool call.

Choose **Instructions** (`g`) to manage shared personal rules. `Ctrl+S` reviews changes;
press it again to save and distribute. **Existing** opens current global files;
**Add one** appends a rule without replacing earlier shared guidance.

Interactive installation and repair use a dedicated terminal conversation: a fixed
input area, scrollable history, a separate plan view, actual setup step states and
masked private inputs. Press `F2` to inspect exact commands and sources, `Ctrl+S`
to continue from chat to approval, and `Ctrl+C` to request cancellation. Use
`--plain` for the original line-based view; scripts and noninteractive approvals
retain their existing output. [Console controls](docs/INSTALL-CONSOLE.md).

| Task | Command / key |
| --- | --- |
| Open the terminal panel | `mcpdeck` |
| List MCP servers / agents | `mcpdeck list` / `mcpdeck agents` |
| Connect an MCP | `mcpdeck connect "Add the official Git MCP"` |
| More panel actions | `m` |
| Add a shared instruction | `mcpdeck instructions append 'Never log credentials.'` |
| List existing global instruction files | `mcpdeck instructions files` |
| Repair/update an MCP | `mcpdeck update <name>` |
| Retry failed synchronization | `mcpdeck sync --retry-failed` |
| Help / customize shortcuts | `?` / `a` |

## Supported agents and limits

MCP adapters include Codex, Claude Desktop/Code, Copilot VS Code/CLI, Cursor, Windsurf,
Antigravity, Gemini CLI, OpenCode, Zed, Cline, Roo Code, Continue, Amazon Q, Kiro,
Qwen Code and TraeCode. Configuration support is broader than live account test coverage.
Global instruction adapters are a separate feature; some agents show **manual** targets.
[Instruction locations and limits](docs/INSTRUCTIONS.md).

This alpha does **not** guarantee unattended installation of every MCP. Provider login,
OAuth authorization, missing runtimes, licenses or database compatibility can need action.
Installation recipes run with your user permissions and are not an operating-system sandbox.
MCPDeck never treats a handshake as proof of database or account access.

Windows executables are not Authenticode signed; macOS packages are not notarized.
Archive publisher signatures and SHA-256 checksums are available for manual verification.
[Security policy](SECURITY.md) · [Package verification](docs/RELEASING.md).

## Desktop client

A local macOS desktop client is available under `desktop/`. It shares the terminal
configuration and supports reviewed MCP activation, agent modes, synchronization
recovery, installation planning and shared instruction editing.
[Build and desktop usage](desktop/README.md).

## Documentation and contributing

- [Getting started (Turkish)](docs/CLI-BASLANGIC.md)
- [Installation chat and troubleshooting (Turkish)](docs/AKILLI-KURULUM.md)
- [Shared instructions and repeatable tests](docs/INSTRUCTIONS.md)
- [Backup recovery](docs/RECOVERY.md)
- [Test record and coverage limits](docs/TEST-SONUCLARI.md)
- [Detailed Turkish reference](docs/REFERENCE-TR.md)
- [Contributing and branch policy](CONTRIBUTING.md)

Please report reproducible bugs and include your OS, MCPDeck version and agent version.
Do not post tokens, passwords, private connection values or complete personal configuration.
Use GitHub private vulnerability reporting for security issues.

Licensed under [MIT](LICENSE). Binary packages also include third-party license notices.
