# Native Windows installation

MCPDeck runs directly in PowerShell or Windows Terminal. WSL, Go and administrator
access are not required to run the packaged application. It remains alpha software.
Use a 64-bit Windows system. Published alpha.2 has native x64 coverage; its ARM64
archive was cross compiled. The next release's source and ZIP installation now
pass native x64 and ARM64 CI, including host/toolchain architecture checks.
Interactive Windows Terminal behavior and real Windows agent accounts still
require manual acceptance testing.

## One-command public installer

```powershell
irm https://raw.githubusercontent.com/altanmehmet/homebrew-mcpdeck/main/install.ps1 | iex
mcpdeck
```

The public distribution installer chooses x64/ARM64, downloads a pinned release
and checks its embedded SHA-256 before extracting or running it. The current
PowerShell process and User PATH are updated. It needs no private repository
access. Treat the public installer repository as part of your trust boundary.

## Install a ZIP

Download `mcpdeck-windows-amd64.zip` for Intel/AMD x64 or
`mcpdeck-windows-arm64.zip` for Windows ARM64 from the
[public releases](https://github.com/altanmehmet/homebrew-mcpdeck/releases).
Only select a release that actually contains Windows assets. Extract the ZIP,
open PowerShell in that folder, then run:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\install.ps1
mcpdeck
```

`ExecutionPolicy Bypass` applies to this PowerShell process. The installer does
not change your system execution policy. Corporate policy can still prevent
script execution; follow your organization's rules.

The default destination is `%LOCALAPPDATA%\MCPDeck\versions\<version>`. The
installer updates the current process and your **User PATH**. Existing terminal
windows must be reopened; when launching the installer in a child PowerShell
process, reopen the parent terminal as well before running `mcpdeck`.

An explicit destination is supported:

```powershell
.\install.ps1 -Prefix "$env:LOCALAPPDATA\MCPDeck"
```

For a portable installation, use `-NoPathUpdate`, then run the installed
`mcpdeck.exe` by its full path. No registry PATH change is made in that mode.
You can also run `mcpdeck.exe` from the extracted ZIP.

## Updates and configuration

Run the new release's installer. Each version has its own directory so a running
older executable is not overwritten. Installing the same binary twice is safe;
an existing version with different binary contents is rejected. Older directories
are retained. Stop older sessions, open a new terminal and run `mcpdeck sync`.
For profiles already in bridge mode this updates the owned `mcpdeck` command to
the running version and preserves a backup. Check `mcpdeck sync status` and restart
the target clients. Only remove an older directory after checking every explicit
bridge reference; a custom/unmanaged entry is not migrated automatically.

The default deck is under `%USERPROFILE%\.config\mcpdeck`; `--config` can select
another file. Application-specific profiles use Windows paths, including
`%APPDATA%` for desktop clients and `%USERPROFILE%` for CLI clients. Custom paths
are preserved. Check detected profiles in the panel before syncing. MCPDeck's
written configuration files use a protected DACL allowing the current user and
SYSTEM; existing broad deck permissions are repaired before loading.

MCP servers can require Node, Java, Python or Docker. These are requirements of
the chosen server. Check `mcpdeck environment` and use the installation chat to
review missing prerequisites. Account login, OAuth and database permissions can
still require your input. Native `.exe` planning agents are recommended.
Unmodified npm Node `.cmd` shims without runtime flags or environment assignments
are resolved to Node and their script directly, preserving complex arguments
without shell expansion. A custom child PATH without an adjacent Node executable
retains its original batch behavior; select an explicit runtime for complex
arguments in that case. Other `.cmd`/`.bat` programs support ordinary arguments
but reject shell expansion characters and embedded quotes. Use a direct runtime
or `.exe` for custom batch programs with complex arguments.
Windows processes start suspended and join a private Job Object before running.
Cancellation and backend shutdown close that job and terminate its associated
child processes. Processes started independently through other system services
are outside this job.

## Uninstall (next release)

The ZIP includes `uninstall.ps1`. Close MCPDeck and any clients using its bridge,
then run the script from the extracted package:

```powershell
.\uninstall.ps1
```

It asks for confirmation, checks installation ownership, removes managed
application versions and their User PATH entries, and preserves your deck,
backups and agent configurations. A custom installation uses `-Prefix`; portable
installations use `-NoPathUpdate`. Unknown files, junctions and legacy versions
without ownership metadata cause the operation to stop before deletion. For
alpha.2, manually remove only the verified legacy application folder after checking
bridge references; the new installer does not adopt or remove older versions. Reinstall the application before using retained bridge entries.
The script is not included in the already published alpha.2 package.

## Manual acceptance test

1. Install, reopen Windows Terminal, run `mcpdeck --version` and `mcpdeck --help`.
2. Run `mcpdeck environment`; check the listed runtime and agent paths.
3. Open `mcpdeck`, navigate using keyboard and mouse, resize the terminal, and
   test copying. Local Windows copying uses the Unicode system clipboard; SSH
   sessions use terminal OSC52 support. `--no-mouse`
   allows normal terminal text selection.
4. Add an MCP using **New MCP** and your logged-in planning agent. Review the
   commands, approve them, and supply secrets only in hidden inputs.
5. Check the MCP verification result and each agent's sync result. Restart the
   target agent and request an actual MCP tool call.
6. Disable, restart the target agent and verify the MCP is unavailable. Enable
   it and repeat the tool call. Remove it and check unrelated entries remain.
7. Review backups and `mcpdeck sync status`. Retry any failed targets.

Automated Windows CI exercises Go tests, vet, vulnerability scanning, ZIP
creation, installation into paths with spaces, repeat installation, User PATH,
private ACLs and a local MCP handshake plus eight profile configuration lifecycles.
These fixture checks do not prove actual Windows agent accounts, OAuth, Oracle
database access or interactive terminal behavior. Archive publisher signatures
are separate from Windows Authenticode; the executable is not Authenticode signed.
