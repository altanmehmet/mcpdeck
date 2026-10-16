# Contributing

MCPDeck is a terminal-first Go application. Keep changes small, preserve
existing profile formats, and add a focused test for every adapter or security
behavior you change.

Read the [community guidelines](CODE_OF_CONDUCT.md) and use [private reporting](SECURITY.md) for vulnerabilities. For larger changes, open an issue to agree on the problem and compatibility requirements first.

Source builds select the patched compiler recommended by `go.mod` (currently Go 1.27.2). The language minimum remains Go 1.24; leave automatic toolchain selection enabled for secure release builds.

Before opening a pull request, run:

```sh
gofmt -w <changed-go-files>
go test ./...
go vet ./...
```

Adapter changes should include tests for discovery, sync, removal, local stdio,
remote HTTP/SSE, and preservation of unrelated configuration. Never commit
tokens, private URLs, database passwords, generated user configuration, or
compiled binaries.

## Branches and pull requests

- `main` is the release branch. Keep it deployable; alpha status is documented
  in the README and does not imply a production release.
- `develop` is the integration branch for the next release.
- Start `feat/<topic>`, `fix/<topic>`, `docs/<topic>`, or `chore/<topic>` (or `codex/<topic>` for Codex work) from
  `develop` and open a pull request targeting `develop`.
- Use squash merge for topic branches. Delete the topic branch after merging;
  keep `main` and `develop`.
- Promote `develop` to `main` through a release pull request using a merge
  commit to preserve branch ancestry. Tag versions only after release validation.
- For urgent fixes, branch from `main`, submit a pull request to `main`, and
  merge `main` back into `develop` afterward.
- Use commit titles such as `feat: add an adapter`, `fix: preserve user config`,
  and `docs: clarify installation`.

Required Linux, macOS, Windows x64/ARM64 and security checks must pass before merging. Review the diff for
credentials, unrelated files, backwards compatibility, and configuration safety.
Do not push directly to `main` or `develop`, force-push them, or delete them.

The public repository protects both branches: pull requests, required CI checks,
resolved conversations and up-to-date branches; force pushes and deletion are
prohibited. Require a non-author review once another maintainer is available.
The initial public snapshot has been reviewed and tested in the pre-launch
private repository; that earlier history remains archived privately.

## Terminal dependency compatibility

The current Bubble Tea 1 / Lip Gloss 1 stack uses cellbuf with ANSI 0.10 APIs.
ANSI 0.11 changes style methods and does not compile with that cellbuf version.
Keep the compatible versions pinned in `go.mod`. Dependabot still proposes patch
updates, but ANSI minor/major updates require a coordinated, separately reviewed
terminal stack migration. Do not disable CI to accept an incompatible update.
If a security fix requires the blocked series, prioritize the migration explicitly.

## Review and release

Explain the user-visible problem, the final behavior and tests actually run. Name any remaining manual checks. Adapter fixes should state which agents were exercised with fixtures versus live sessions. Add a changelog entry for user-visible changes. Never present a successful handshake as proof of real account access.

Only maintainers publish tagged CLI releases. Package builds must pass vulnerability checks, verify all architecture checksums and verify the publisher signature before upload. Homebrew and the Windows bootstrap must pin the same version and the matching archive hashes. See [release operations](docs/RELEASING.md).
