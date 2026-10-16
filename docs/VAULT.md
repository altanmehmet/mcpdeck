# Private MCP credentials

Planning agents need input names such as `API_TOKEN`, `DB_USER` and `DB_PASSWORD`, not their values. Enter values only in MCPDeck's separate masked input. Ordinary chat and documentation files are sent to the planner: never paste credentials there.

## New installations

```sh
mcpdeck install "Install the documented MCP for my service" --vault
```

In the New MCP form, Ctrl+K (or clicking the OS VAULT row) enables the same option.

Planning, chat and approval finish before private inputs are collected. MCPDeck saves bound values in macOS Keychain or Windows Credential Manager. Linux requires `secret-tool` and an unlocked Secret Service. No library or background service is bundled. Missing/locked vaults fail without saving plaintext as a fallback.

Deck variables contain `vault://<opaque-id>` references. Direct sync writes a per-server `mcpdeck vault exec <server>` wrapper into agent settings instead of resolved credentials. The wrapper retrieves credentials locally at runtime; it forwards the MCP's stdio protocol without changing tool names. Remote connections use the existing pinned MCPDeck stdio adapter. Bridge mode also resolves credentials locally. Existing connections and the default install behavior remain compatible; vault storage is opt-in.

Vault credentials require HTTPS for remote MCP endpoints. Plain HTTP is allowed only for `localhost` and literal loopback IP addresses, so local MCP services remain usable. Credentials must not cross a network in plaintext ([OWASP guidance](https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html#28-transport-layer-security-tls-everywhere)).

Vault inputs must appear in environment values or HTTP headers. Secrets in executable names, arguments or URLs are rejected because those can expose them through process listings. Recipes using password arguments need a documented environment/header connection before vault installation can proceed.

## Existing installations

```sh
mcpdeck vault bind my-server API_TOKEN
mcpdeck sync
```

The bind command reads a new value in a masked terminal field. `--stdin` accepts exact bytes from a password manager or protected pipeline; a trailing newline is part of the value. Never put the secret in a shell command or argument. The input variable must already exist in that server's bound variables. Usernames can be bound the same way.

Binding updates the deck reference; run sync and reload the agents to replace their old settings. Earlier plaintext settings, recovery backups or imported files may retain old values. Rotate migrated credentials; migration does not silently delete recovery data. Each update creates a new vault item so a failed repair does not overwrite credentials for the previous connection. Old items can be removed in the OS credential-store UI after verification.

```sh
mcpdeck vault forget my-server API_TOKEN
```

Forgetting deletes the current OS credential. The reference remains; the MCP fails until rebound. There is no command that prints stored secrets. Values are limited to 2560 bytes.

## Protection boundary

MCPDeck's private input path never adds credential values to planner prompts or the installation chat. Planner repair requests contain input names, not vault IDs or retrieved values. Runtime stderr is suppressed to avoid third-party credential diagnostics. The MCP necessarily receives its credentials and can contact its configured service. Its tool responses still reach the agent and could contain sensitive information; the vault cannot sanitize arbitrary MCP behavior.

This is OS credential storage, not a sandbox against other programs running as the same user, an administrator, or a compromised MCP. Do not promise that an agent with unrestricted local command access cannot retrieve the user's OS credentials. MCPDeck does not disable the agent's own permissions or migrate its provider login/API authentication into this vault. macOS may show Keychain access prompts. Windows entries persist locally for the same signed-in user ([Microsoft CREDENTIALW documentation](https://learn.microsoft.com/en-us/windows/win32/api/wincred/ns-wincred-credentialw)). Go strings and child environments can retain secret bytes in process memory; complete memory erasure is not guaranteed.
