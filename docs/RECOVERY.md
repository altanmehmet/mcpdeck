# Synchronization and recovery

Every changed MCP sync target is backed up with mode `0600` as
`<target>.mcpdeck-backup`. An unchanged sync preserves the existing backup.
The backup contains the complete previous configuration and may contain secrets;
do not attach it to issues or commit it. Only the most recent changed version is
retained. Newly created target files have no previous-file backup.

## Inspect and retry a partial synchronization

```sh
mcpdeck sync status
mcpdeck sync --retry-failed
```

Installation, repair, normal sync, and panel sync record target outcomes in
`<deck-file>.sync-status.json`. The report contains profile names, target paths,
and statuses, never connection settings or error output. `status` shows the last
attempt, not live connection health. Use `mcpdeck doctor` and an actual agent tool
call to check connections.

Retry uses the current deck and only targets marked `failed`; successful targets
are skipped. If the profile or target path changed, review it and run a normal
sync explicitly. With no report or no failures, retry makes no changes. A failed
report write is reported separately; run a normal sync after resolving that error.

In the panel, use **Retry** or **Ctrl+R**. The key can be reassigned in Help.
On a narrow terminal the button may be outside the visible area; use the keyboard.

## Restore the previous agent configuration

```sh
# Preview the operation without writing:
mcpdeck sync restore --profile cursor
# Review the backup locally, then confirm:
mcpdeck sync restore --profile cursor --yes
```

Restore replaces the **complete** agent configuration, including non-MCP settings.
Any changes made since that backup will be replaced. The current file becomes the
backup, so the same confirmed command can undo the restore. Invalid or missing
backups are rejected. This command restores sync backups, not discovered-removal
backups (`.mcpdeck-remove-backup`).

Deck selections remain unchanged. A later sync reapplies those selections; change
them first if the rollback should persist. Reload the agent's MCP connections.

## Concurrent writers

MCPDeck config writers lock the resolved target path and re-read the file before
replacement. Existing symlinks are preserved. Detected external changes cause a
conflict and require a retry instead of overwriting that edit. Other applications
do not honor MCPDeck locks: a small interval between the final check and rename
remains, so avoid editing the same configuration during synchronization.

If a process crashes, a `<target>.mcpdeck-lock` directory may remain. Remove it
only after confirming no MCPDeck writer is still running. Multi-agent sync is
not an all-or-nothing transaction; successes are retained and failures are retried.
