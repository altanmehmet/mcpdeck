# Release preparation

MCPDeck is alpha software. The public `altanmehmet/mcpdeck` repository hosts the
MIT-licensed source. The separate `altanmehmet/homebrew-mcpdeck` repository hosts
the tap and binary alpha releases. Pre-launch development history is retained
in a private archive; the public source starts from a reviewed, sanitized snapshot.

## Build and verify a local package

```sh
MCPDECK_VERSION=0.1.0-dev sh scripts/package-cli.sh
cd dist
shasum -a 256 -c mcpdeck-*.sha256
```

Archives contain the executable, installer, MIT license, and guides including
recovery. Existing archive filenames are preserved. Versioned builds embed
`MCPDECK_VERSION` in `mcpdeck --version`; source builds default to `0.1.0`.
Checksums detect corruption when obtained from a trusted source; they are not
signatures or proof of publisher identity.

Packages also include `THIRD_PARTY_NOTICES.txt`, generated from the licenses of
modules linked on all four supported platforms and the Go runtime. Packaging
fails if a license cannot be found.

The **packages** GitHub Actions workflow is manually dispatched for a selected
branch. It produces macOS/Linux tar archives and Windows ZIPs, each for
arm64/amd64, with SHA-256 files. An optional `version` input embeds a release
version; omitting it produces a development snapshot.
On `main`, a follow-up job signs the combined manifest using the dedicated
`MCPDECK_RELEASE_SIGNING_KEY` repository secret and verifies it before uploading
the signed artifact. Other branches only build unsigned snapshots. The signing
key is never committed and is removed from the runner by an exit trap.
Artifacts expire after 14 days and do not create a release. Cross compilation does
not prove that the binaries work on those machines; native release testing is
still required.

## Public Homebrew binary tap

```sh
brew tap altanmehmet/mcpdeck
brew install altanmehmet/mcpdeck/mcpdeck
brew test altanmehmet/mcpdeck/mcpdeck
mcpdeck
```

The public formula downloads a versioned archive with a pinned SHA-256 checksum.
It requires neither source repository access nor Go. The current published version
is `0.1.0-alpha.3`; installing through Homebrew does not make it a stable release.
Homebrew checks the archive checksum; it does not independently validate the
OpenSSH publisher signature. Manual signature verification is described below.

[Alpha packages and release notes](https://github.com/altanmehmet/homebrew-mcpdeck/releases/tag/v0.1.0-alpha.3).
Actual macOS arm64 Homebrew installation and `brew test` passed. Public binary CI
also verified signatures and ran native installers on its macOS/Linux runners.

If `command -v mcpdeck` still points to a previous manual installation, that
binary can shadow Homebrew. Back it up and point the old command path at
`$(brew --prefix)/bin/mcpdeck`, or update PATH to select Homebrew. Do not leave a
separate stale copy in front of Homebrew if you expect `brew upgrade` to update
the command you run. Uninstalling Homebrew leaves any manually created symlink
and manual backup for you to clean up.

## Private source development formula

`Formula/mcpdeck.rb` is a HEAD-only formula that builds the `main` branch from
source with Go. It has no fabricated stable release URL or checksum. To try it,
copy it into a local Homebrew tap, then install with `--HEAD`. This development
formula is separate from the binary
formula in the public tap.

Validate installation and `brew test` on supported machines before advertising
their support. Native testing covers macOS arm64, the macOS/Linux CI runners, and Windows
x64/ARM64 source plus ZIP lifecycle checks. Linux ARM64 archives remain cross
compiled. Disclose coverage for the exact published version; next-release CI
results do not retroactively validate older archives.

## Publisher signature verification

Download all archives, `SHA256SUMS`, and `SHA256SUMS.sig` from the same
versioned public release. Obtain the verifier and allowed signers from an
independently trusted checkout of the tap, rather than trusting a key bundled
with the download you are checking:

```sh
sh scripts/verify-packages.sh /absolute/path/to/downloads keys/allowed_signers
```

The dedicated Ed25519 public key fingerprint is:
`SHA256:M8NS9J5AVuMDXSordI+V7dVzHdWKWcXTAMr/WolzzK8`.
The alpha.1/alpha.2 key (`SHA256:sd7JZx1++fUe2ETVq60wqJowhKNXS7Sk9b/qgmOo7fQ`)
remains in `keys/allowed_signers` for historical verification. A new dedicated
key is used by the public source repository; private keys never enter Git.
The signature covers the entire manifest using namespace `mcpdeck-release`.
Verification rejects unexpected paths, missing/duplicate entries, symlinks,
changed archives and manifests. It requires OpenSSH with `ssh-keygen -Y` support.
Tests generate disposable keys and verify both valid and rejected inputs.

The verifier accepts the original four-platform manifest and the six-platform
manifest including both Windows ZIPs, requiring the complete set in either case.

This is an archive publisher signature, **not Windows Authenticode, Apple
Developer ID signing or notarization**. No valid Developer ID identity is installed on the development
Mac. Apple signing requires the owner's Apple Developer certificate and
notarization credentials; these must be provided securely outside the repository.

For key rotation, generate a new dedicated key, replace the encrypted repository
secret and the public allowed signers in both repositories, publish the new
fingerprint through a trusted channel, and retain old public keys separately to
verify historical releases. Do not reuse an SSH login key for package signing.

## Publishing an alpha

1. Complete the checks below and merge the source changes into `main`.
2. Dispatch **packages** on `main` with the same release `version` for all six
   archives. Download the `mcpdeck-signed-snapshots` artifact after all jobs pass.
3. Verify using `sh scripts/verify-packages.sh dist keys/allowed_signers`.
   Local signing supports `sh scripts/sign-packages.sh dist /secure/path/to/dedicated-key --windows`;
   without `--windows`, the historical four-platform manifest is preserved.
4. Create a **draft prerelease** in `altanmehmet/homebrew-mcpdeck`, attach the
   archives, manifest and signature, and disclose native/live test coverage.
5. Update the tap formula's version, immutable URLs and four archive checksums.
   Update the public Windows bootstrap's pinned version, both ZIP checksums and
   exact archive contents together. Windows alpha.3 includes
   `uninstall.ps1` (nine files); the previous alpha.2 bootstrap expects eight and will reject
   the new ZIP. Run the public distribution CI against the draft assets before
   publishing. Keep existing release assets immutable.
6. Publish the prerelease, run a real Homebrew install/test, then publish the tap
   formula. A rollback uses a new version; never replace assets of an existing
   published version. This preserves the formula's pinned checksums.

## Release checklist

1. macOS/Linux and Windows CI jobs pass: tests, vet, build and installation smoke.
   Unix CI also exercises real PTY behavior; Windows terminal interaction still
   requires manual acceptance testing described in [WINDOWS.md](WINDOWS.md).
2. Run the official Go vulnerability scanner with an up-to-date Go compiler.
3. Verify the live agent flow for each supported client: add, actual tool call,
   disable, enable, remove. Mark untested adapters explicitly in the release notes.
4. Test an interrupted install, a failed target sync, retry, and backup restoration.
5. Validate archive installation on native macOS/Linux architectures.
6. Merge a reviewed `develop` → `main` release PR using a merge commit.
7. Only then create a version tag and publish archives, checksums, release notes,
   and publisher provenance/signatures. Do not tag an unfinished release.
8. Enable GitHub branch protections when the account plan/repository visibility
   permits it; PR policy currently relies on maintainer discipline.

For locally installed Codex and Copilot CLI, run
`python3 tests/client_config_smoke.py /absolute/path/to/mcpdeck`. This verifies
their real configuration readers in a temporary HOME through add/disable/enable/
remove. It does not authorize accounts or make actual agent/MCP tool calls.

For an optional live test using existing paid account access:

```sh
python3 tests/live_agent_smoke.py /absolute/path/to/mcpdeck codex
python3 tests/live_agent_smoke.py /absolute/path/to/mcpdeck copilot
```

These tests install an isolated local fixture through MCPDeck, demand an actual
tool call with a random server-generated proof, verify disable/remove through
the real client reader, and call the tool again after re-enabling it. Codex uses
a temporary copy of file-based login; Copilot uses the active `gh` login token
in memory. Only the fixture tool is approved. Existing user configurations are
not changed. Tests consume provider account usage and are not run in public CI.
OAuth servers and other agents remain outside this test's scope.

OAuth/account authorization and database privileges must be checked with actual
tool calls. An MCP handshake and tool list alone do not prove access.


For existing Oracle and Sentry installations, run:

```sh
python3 tests/real_mcp_smoke.py /absolute/path/to/mcpdeck
# Only after explicitly selecting a saved test connection:
python3 tests/real_mcp_smoke.py /absolute/path/to/mcpdeck --oracle-connection SAVED_TEST_CONNECTION
```

This reads the managed Oracle entry and the Copilot CLI Sentry entry into a private
isolated deck, verifies both real tool lists, lists saved Oracle connections and
calls Sentry `whoami`. It preserves existing configurations and does not print
account payloads. The explicit connection option additionally runs only
`SELECT 1 AS MCPDECK_HEALTH FROM DUAL` and disconnects its own SQLcl session.
Existing Sentry authorization does not prove first-time OAuth login or refresh.
