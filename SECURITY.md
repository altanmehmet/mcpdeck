# Security policy

MCPDeck writes MCP configuration files and, only after an explicit review,
executes installation commands with the current user's permissions. It is not
an operating-system sandbox.

## Safe use

- Review every installation source, command, argument, and target before typing
  `y` at the approval prompt.
- Keep credentials in private input prompts or environment references. Do not
  put tokens in a plan step, issue, log, or chat message.
- Deck files, agent configuration files, and removal backups are written with
  mode `0600` where the platform permits it. Values are not encrypted by
  MCPDeck; use the operating system's secret store or agent-managed OAuth when
  available.
- Project-scoped MCP files are code and configuration from the project. Review
  them before enabling discovery or synchronization.

## Reporting

Please do not publish an unpatched vulnerability with credentials or a
reproduction secret. Open a private security report through the repository
hosting service, or contact the maintainer listed in the project metadata.
