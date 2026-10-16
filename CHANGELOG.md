# Changelog

Changes are grouped by released CLI version. Published binaries live in [homebrew-mcpdeck](https://github.com/altanmehmet/homebrew-mcpdeck/releases); source tags live in [this repository](https://github.com/altanmehmet/mcpdeck/releases).

## 0.1.0-alpha.8 — 2026-10-16

- Add optional OS-vault storage for installation/repair credentials, private binding/deletion commands and a New MCP form toggle. Agent settings receive local runtime references.
- Require HTTPS for remote vault connections; allow loopback HTTP for local MCP services. Reject vault values in URLs and process arguments.
- Select patched Go 1.27.2 for source builds and check vulnerabilities before packaging.
- Accept agy structured plans serialized as either a JSON object or a JSON string.
- Refresh the project overview, compatibility documentation, support/security guidance and contribution/issue templates for the public CLI alpha.

The language minimum remains Go 1.24. Unused-code advisories in existing Windows-related dependencies are recorded in the security policy; the affected functions are not reported as reachable. This release is not a claim of complete security or universal live agent coverage.

## 0.1.0-alpha.7 — 2026-10-13

- Add Antigravity CLI (`agy`) planning and model selection from `agy models`.
- Isolate agy planning settings and constrain planning tools.
- Scroll long model lists and align mouse selection with visible choices.
- Fix a PTY test race during expected terminal exit.

## 0.1.0-alpha.6 — 2026-10-12

- Fix Space input in the installation conversation.
- Improve CLI planner discovery, including Copilot and user-installed executables.
- Add CLI model selection and planner detection diagnostics.

## 0.1.0-alpha.5 — 2026-10-12

- Improve name-first MCP installation and startup with empty agent configurations.
- Correct global instruction home-path handling.
- Distribute pinned CLI archives through Homebrew and the Windows bootstrap.

Earlier source/desktop experiments remain in [historical release notes](https://github.com/altanmehmet/mcpdeck/releases). They are not the current distribution channel.
