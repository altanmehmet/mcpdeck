#!/usr/bin/env python3
"""Real binary installation flow, with a local MCP and no account/network calls."""
import json
import os
import pathlib
import re
import subprocess
import sys
import tempfile

binary = str(pathlib.Path(sys.argv[1]).resolve())
with tempfile.TemporaryDirectory(prefix="mcpdeck-install-smoke-") as temporary:
    root = pathlib.Path(temporary)
    environment = dict(os.environ, HOME=str(root))
    server = root / "server.py"
    server.write_text('''import json, sys
for line in sys.stdin:
    request = json.loads(line)
    if "id" not in request: continue
    if request["method"] == "initialize":
        result = {"protocolVersion":"2025-11-25","capabilities":{"tools":{}}}
    elif request["method"] == "tools/list":
        result = {"tools":[{"name":"health","inputSchema":{"type":"object"}}]}
    else: continue
    print(json.dumps({"jsonrpc":"2.0","id":request["id"],"result":result}),flush=True)
''')
    profiles = {}
    for name, fmt, remote in [("cursor", "", "url"), ("claude", "", "stdio"),
                              ("claude-code", "", "http"), ("codex", "codex", ""),
                              ("copilot", "vscode", ""), ("copilot-cli", "copilot-cli", ""),
                              ("windsurf", "", "serverUrl"), ("antigravity", "", "serverUrl")]:
        target = root / (name + ".config")
        target.write_text("keep = true\n" if fmt == "codex" else '{"keep":true}')
        profiles[name] = {"target_path": str(target), "format": fmt,
                          "remote_format": remote, "enabled_servers": []}
    deck = root / "deck.json"
    deck.write_text(json.dumps({"version": 1, "servers": {}, "profiles": profiles}))
    plan = root / "plan.json"
    plan.write_text(json.dumps({"version": 1, "name": "smoke", "summary": "Local fixture",
        "sources": [], "requirements": [sys.executable], "manual": [],
        "steps": [{"description": "Create local marker", "command": "/usr/bin/touch",
                   "args": ["installed"], "directory": "."}],
        "connection": json.dumps({"command": sys.executable, "args": [str(server)],
                                  "env": {"TOKEN": "${SMOKE_TOKEN}"}})}))

    def run(*args, data=None, success=True):
        result = subprocess.run([binary, "--config", str(deck), *args], input=data,
                                text=True, capture_output=True, timeout=30,
                                env=environment)
        assert (result.returncode == 0) == success, result.stderr + result.stdout
        assert "fixture$secret" not in result.stdout + result.stderr
        return result.stdout

    preview = run("install", "--plan", str(plan), "--plan-only")
    match = re.search(r"Plan ID: ([a-f0-9]{64})", preview)
    assert match, preview
    digest = match.group(1)
    assert not (root / "installations").exists()
    run("install", "--plan", str(plan), "--approve", "wrong", success=False)
    output = run("install", "--plan", str(plan), "--approve", digest,
                 "--values-stdin", data='{"SMOKE_TOKEN":"fixture$secret"}')
    assert "Verified: 1 tools. Updated 8 agent configurations." in output, output
    assert (root / "installations/smoke/installed").exists()
    for profile in profiles.values():
        target = pathlib.Path(profile["target_path"])
        assert "smoke" in target.read_text() and "keep" in target.read_text()
        assert "smoke" not in pathlib.Path(str(target) + ".mcpdeck-backup").read_text()
    run("disable", "smoke")
    for profile in profiles.values():
        assert "smoke" not in pathlib.Path(profile["target_path"]).read_text()
    run("enable", "smoke")
    for profile in profiles.values():
        assert "smoke" in pathlib.Path(profile["target_path"]).read_text()
print("PASS: binary plan approval, setup, real MCP probe, eight agents, backups, disable/enable")
