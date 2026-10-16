# Support

## Questions and troubleshooting

Use [GitHub Discussions](https://github.com/altanmehmet/mcpdeck/discussions) for setup questions, usage and ideas. MCPDeck is maintained as an open-source alpha project; response times are best effort, not a support SLA.

Before reporting a problem:

1. Run `mcpdeck --version` and check the [latest CLI release](https://github.com/altanmehmet/homebrew-mcpdeck/releases).
2. Run `mcpdeck doctor`. For planner detection, also check `mcpdeck planner detected`.
3. For synchronization problems, inspect `mcpdeck sync status`. Keep your backups; [recovery steps](docs/RECOVERY.md) explain how to inspect and restore changes.
4. Check [agent compatibility](docs/COMPATIBILITY.md) and the relevant agent/runtime version.

Inspect diagnostics before sharing them. Paths and server names can reveal private information even when credential values are hidden.

## Reproducible bugs

[Open a bug report](https://github.com/altanmehmet/mcpdeck/issues/new?template=bug_report.yml) with your OS/architecture, MCPDeck version, agent version, exact steps and expected/actual behavior. Include the install method and a minimal sanitized example if relevant.

Do not attach your full deck, agent settings, OS credential-store contents, raw planner conversations, connection strings, tokens or passwords. Reproduce with synthetic values where possible. For a feature or agent adapter request, use the matching [issue form](https://github.com/altanmehmet/mcpdeck/issues/new/choose).

## Security reports

Use [private vulnerability reporting](https://github.com/altanmehmet/mcpdeck/security/advisories/new), not a public issue. [Security policy](SECURITY.md).
