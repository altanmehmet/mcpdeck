"""Opt-in native OS-vault test. Uses and removes a disposable credential only."""
import json
import pathlib
import subprocess
import sys
import tempfile

binary = str(pathlib.Path(sys.argv[1]).resolve())
with tempfile.TemporaryDirectory(prefix="mcpdeck-vault-test-") as temporary:
    root = pathlib.Path(temporary)
    deck = root / "deck.json"
    target = root / "agent.json"
    child = root / "fixture.py"
    secret = "fixture-only spaces $ ` Türkçe"
    child.write_text('''import json, os, sys
if os.environ.get("FIXTURE_TOKEN") != "fixture-only spaces $ ` Türkçe":
    sys.exit(4)
for line in sys.stdin:
    request=json.loads(line)
    if "id" not in request: continue
    result={"tools": []} if request["method"]=="tools/list" else {"protocolVersion":"2025-11-25","capabilities":{"tools":{}},"serverInfo":{"name":"fixture","version":"1"}}
    print(json.dumps({"jsonrpc":"2.0","id":request["id"],"result":result}),flush=True)
''', encoding="utf-8")
    deck.write_text(json.dumps({"version": 1, "servers": {"fixture": {"command": sys.executable, "args": [str(child)], "env": {"FIXTURE_TOKEN": "${TOKEN}"}, "variables": {"TOKEN": "replace-me"}}}, "profiles": {"fixture-agent": {"target_path": str(target), "enabled_servers": ["fixture"]}}}), encoding="utf-8")

    def run(*args, input=None):
        result = subprocess.run([binary, "--config", str(deck), *args], input=input, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=40)
        assert secret.encode() not in result.stdout + result.stderr, "private value in console output"
        assert result.returncode == 0, result.stderr.decode(errors="replace")
        return result.stdout

    try:
        run("vault", "bind", "fixture", "TOKEN", "--stdin", input=secret.encode())
        assert secret not in deck.read_text(), "secret persisted in deck"
        assert "vault://" in deck.read_text(), "missing vault reference"
        run("sync")
        synced = target.read_text()
        assert secret not in synced and "vault://" not in synced and "FIXTURE_TOKEN" not in synced
        assert '"vault"' in synced and '"exec"' in synced
        requests = [{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {}}, {"jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": {}}]
        output = run("vault", "exec", "fixture", input=("\n".join(json.dumps(x) for x in requests)+"\n").encode())
        responses = [json.loads(line) for line in output.splitlines()]
        assert [r["id"] for r in responses] == [1, 2], "runtime altered MCP request IDs"
        assert responses[1]["result"] == {"tools": []}
        run("vault", "forget", "fixture", "TOKEN")
        failure = subprocess.run([binary, "--config", str(deck), "vault", "exec", "fixture"], input=b"", capture_output=True, timeout=40)
        assert failure.returncode != 0 and secret.encode() not in failure.stdout+failure.stderr
        print("PASS: native vault bind, reference-only sync, private runtime env, transparent stdio, forget and fail-closed restart")
    finally:
        reference=json.loads(deck.read_text())["servers"]["fixture"]["variables"]["TOKEN"]
        if reference.startswith("vault://"):
            subprocess.run([binary, "--config", str(deck), "vault", "forget", "fixture", "TOKEN"], capture_output=True, timeout=40)
