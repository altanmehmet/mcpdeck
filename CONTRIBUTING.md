# Contributing

MCPDeck is a terminal-first Go application. Keep changes small, preserve
existing profile formats, and add a focused test for every adapter or security
behavior you change.

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
- Start `feat/<topic>`, `fix/<topic>`, `docs/<topic>`, or `chore/<topic>` from
  `develop` and open a pull request targeting `develop`.
- Use squash merge for topic branches. Delete the topic branch after merging;
  keep `main` and `develop`.
- Promote `develop` to `main` through a release pull request using a merge
  commit to preserve branch ancestry. Tag versions only after release validation.
- For urgent fixes, branch from `main`, submit a pull request to `main`, and
  merge `main` back into `develop` afterward.
- Use commit titles such as `feat: add an adapter`, `fix: preserve user config`,
  and `docs: clarify installation`.

Both Linux and macOS CI jobs must pass before merging. Review the diff for
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
