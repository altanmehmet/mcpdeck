# Security policy

## Report privately

Use [GitHub private vulnerability reporting](https://github.com/altanmehmet/mcpdeck/security/advisories/new). Do not open a public issue with an unpatched exploit, real credentials or private configuration. Include the affected version, platform, impact and a minimal reproduction with synthetic data.

The maintainer handles reports on a best-effort basis. There is no guaranteed response SLA, bug bounty or independent security certification. Coordinate disclosure before publishing sensitive reproduction details.

## Supported versions

The latest published CLI alpha is the maintained release. Older alphas may not receive backported fixes. Source changes do not update already installed binaries; install the new release when a fix is published. Desktop experiments are outside the current release channel.

## What MCPDeck protects

- Private installation fields do not enter the planner prompt or installation chat. Ordinary requests, chat and supplied documentation do reach the chosen provider: do not put secrets there.
- Optional `--vault` stores values in macOS Keychain, Windows Credential Manager or Linux Secret Service. Direct sync writes a local runtime reference, not the retrieved value. Missing/locked credentials fail without plaintext fallback. [Vault behavior and migration](docs/VAULT.md).
- Vault values in process arguments/URLs are rejected. Remote vault endpoints require HTTPS; local loopback HTTP remains available.
- Config changes use backups and conflict checks. Unix files are private (`0600`) and Windows uses user-scoped permissions where supported.
- Release archives carry SHA-256 checksums and an OpenSSH publisher signature. Verify them with the trusted allowed-signers file from the repository; a checksum alone is not independent proof of origin. [Verification](docs/RELEASING.md).

## What it does not protect

MCPDeck is not an operating-system sandbox. Reviewed installation commands run with your user permissions. Agents with unrestricted local command access, administrators and compromised MCPs can still access private data. A necessary runtime credential is available to its MCP process; MCP responses may disclose data. Full erasure of Go strings and child-process environments is not guaranteed.

Vault storage is opt-in. Existing plaintext deck values, agent files, import files and recovery backups are not retroactively encrypted. Rotate migrated credentials and inspect old copies. Provider authentication/OAuth is separate from MCP credential storage. macOS packages are not notarized and Windows executables are not Authenticode signed.

## Verification and known dependency records

As of 2026-10-16, Go 1.27.2 source/binary scans and current CI report no reachable known vulnerabilities in the checked CLI paths. This is a dated observation, not a guarantee. CI runs Go vulnerability checks and a redacted git-history secret scan; arbitrary third-party MCP runtimes and providers are outside those Go checks.

The dependency graph still contains unused-code advisories: [GO-2026-5024](https://pkg.go.dev/vuln/GO-2026-5024) in `x/sys`, and [GO-2026-6629](https://pkg.go.dev/vuln/GO-2026-6629) / [GO-2026-5970](https://pkg.go.dev/vuln/GO-2026-5970) in `x/text`. Windows scanning reports the affected functions as not called. The modules themselves are required by Windows/runtime/terminal support. The fixed `x/sys` version requires Go 1.25; changing the existing Go 1.24 language minimum is a separate compatibility decision. Do not suppress these records or describe the dependency graph as advisory-free.
