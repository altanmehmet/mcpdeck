"""Verify real Codex/Copilot CLI config readers against isolated MCPDeck writes.

Usage: python3 tests/client_config_smoke.py /absolute/path/to/mcpdeck
Requires both CLIs on PATH. Does not log in, call a model, or call an MCP tool.
"""
import json
import os
import pathlib
import shutil
import subprocess
import sys
import tempfile

binary = str(pathlib.Path(sys.argv[1]).resolve())
clients = {name: shutil.which(name) for name in ("codex", "copilot")}
assert all(clients.values()), "Install Codex and Copilot CLI to run this optional test"
with tempfile.TemporaryDirectory(prefix="mcpdeck-real-config-") as temporary:
    root = pathlib.Path(temporary)
    env = dict(PATH=os.environ.get("PATH", ""), HOME=str(root),
               CODEX_HOME=str(root / ".codex"), COPILOT_HOME=str(root / ".copilot"),
               TMPDIR=str(root))
    (root / ".codex").mkdir()
    (root / ".copilot").mkdir()
    deck = root / "deck.json"
    deck.write_text(json.dumps({"version": 1, "servers": {}, "profiles": {
        "codex": {"target_path": str(root / ".codex/config.toml"), "format": "codex", "enabled_servers": []},
        "copilot-cli": {"target_path": str(root / ".copilot/mcp-config.json"), "format": "copilot-cli", "enabled_servers": []},
    }}))

    def run(arguments):
        result = subprocess.run(arguments, cwd=root, env=env, text=True,
                                capture_output=True, timeout=30)
        assert result.returncode == 0, result.stderr
        return result.stdout

    def configured(name):
        value = json.loads(run([clients[name], "mcp", "list", "--json"]))
        return "mcpdeck-fixture" in json.dumps(value)

    run([binary, "--config", str(deck), "add", "--name", "mcpdeck-fixture", "--command", sys.executable])
    for client in clients:
        assert configured(client), client + " did not read MCPDeck's configuration"
    run([binary, "--config", str(deck), "disable", "mcpdeck-fixture"])
    for client in clients:
        assert not configured(client), client + " retained disabled MCP"
    run([binary, "--config", str(deck), "enable", "mcpdeck-fixture"])
    for client in clients:
        assert configured(client), client + " did not read re-enabled MCP"
    run([binary, "--config", str(deck), "remove", "mcpdeck-fixture", "--yes"])
    for client in clients:
        assert not configured(client), client + " retained removed MCP"
print("PASS: real Codex and Copilot CLI config readers: add, disable, enable, remove")
