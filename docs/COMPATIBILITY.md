# Compatibility

MCPDeck has three separate integration surfaces. They should not be read as one blanket support claim.

| Capability | Scope |
| --- | --- |
| MCP configuration | Agent-specific files are discovered, rendered and backed up. Adapter tests exercise formats and preservation of unrelated settings. |
| Installation planning | A locally installed CLI or configured API provider proposes recipes. The user reviews commands before they run. |
| Global instructions | Personal/global files are read or updated. Some agents need a manual step; project-specific instructions keep their scope. |

## MCP configuration adapters

Built-in profiles: Cursor, Claude Desktop, Windsurf, Claude Code, Codex, Copilot in VS Code, Copilot CLI and Antigravity.

Additional profiles are detected when the relevant app/configuration is present: Gemini CLI, OpenCode, Zed, Cline, Continue, Amazon Q, Kiro, Roo Code, Qwen Code and TraeCode. TraeCode can use a project MCP file; that does not make project instruction files part of shared global instructions.

Missing apps or configuration files do not establish support. Do not use `--all-profiles` unless you intentionally want to create absent configured targets. [Global instruction paths and manual targets](INSTRUCTIONS.md).

## Installation planners

Codex, Claude Code, Gemini CLI, Copilot CLI and Antigravity CLI (`agy`) are detected through executable locations. Check your machine with:

```sh
mcpdeck planner detected
```

A planner must be installed and authenticated separately. The model picker supports retaining a default or entering a model ID; `agy` can populate its picker from `agy models`. API options include OpenAI, Anthropic, Gemini and OpenAI-compatible endpoints. Provider account usage may apply.

## Platforms and test coverage

| Platform | CLI distribution | Verification |
| --- | --- | --- |
| macOS ARM64 / x64 | Homebrew or archive | macOS CI, archive/installer checks; local ARM64 terminal and OS-vault checks. |
| Linux ARM64 / x64 | Homebrew or archive | Linux CI and native installer checks; ARM64 packages are cross-compiled. |
| Windows ARM64 / x64 | PowerShell bootstrap or ZIP | Native runners test both architectures, installer lifecycle, credential storage and runtime forwarding. |

CI runs fixture-backed MCP and planner tests. Historical live tests cover specific Codex/Copilot sessions and macOS `agy` planning; they do not establish live behavior for all agents. Linux OS-vault use needs `secret-tool` and an unlocked Secret Service; its live keyring integration is not covered by the native Windows/macOS evidence. See [test records](TEST-SONUCLARI.md) and the checks attached to each release.

## Known limits

- MCP initialization and `tools/list` do not prove actual tool/account/database access.
- OAuth, licenses, missing runtimes and account setup can require manual steps.
- MCPDeck is not an OS sandbox. Read [credentials and safety](../SECURITY.md).
- Publisher-signed archives are distinct from Apple notarization or Windows Authenticode signing; those platform signatures are not present.
- The public distribution is the CLI. Desktop work is not included in these releases.
