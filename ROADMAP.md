# Roadmap

MCPDeck focuses on a small local CLI for MCP configuration and shared global instructions. This is a direction for maintenance, not a promised delivery schedule. Propose concrete work in [Issues](https://github.com/altanmehmet/mcpdeck/issues) or [Discussions](https://github.com/altanmehmet/mcpdeck/discussions).

## Available in the alpha CLI

- Agent configuration adapters, discovery, enable/disable, removal and recoverable synchronization.
- Reviewed installation/repair recipes with MCP initialization and tool discovery.
- Installed CLI planners and API providers, with model selection.
- Shared personal instructions and editing of existing global files.
- Optional OS-vault credential storage in alpha.8+, with reference-only agent configuration.
- macOS/Linux/Windows archives, Homebrew and a Windows bootstrap with checksums; publisher signatures on release archives.

## Before a stable 0.1 release

- Broaden live tests across supported agent versions, especially beyond existing Codex/Copilot/macOS agy evidence.
- Exercise Linux Secret Service against a real unlocked keyring in CI.
- Make a separately reviewed source-toolchain/dependency upgrade to resolve unused-code dependency advisories without hiding compatibility changes.
- Evaluate macOS notarization and Windows Authenticode signing and document their operational requirements.
- Continue upgrade/rollback testing and collect reports from clean machines and different terminal environments.

## Scope

The published product is the CLI. Desktop experiments are not part of the release channel. MCPDeck does not replace an agent, host a model or certify arbitrary third-party MCP servers. An installation handshake is not a substitute for testing account access and actual tools.
