"""Optional paid-account test: real agent calls an MCPDeck-installed local tool.

Usage: python3 tests/live_agent_smoke.py ./mcpdeck codex|copilot
Uses existing login; Codex auth is copied into a temporary private directory.
Never prints account files or model output. No real agent configuration is edited.
"""
import json
import os
import pathlib
import shutil
import subprocess
import sys
import tempfile

binary = str(pathlib.Path(sys.argv[1]).resolve())
client = sys.argv[2]
assert client in ("codex", "copilot")
executable = shutil.which(client)
assert executable, "Client is not installed"
fixture = '''import json, pathlib, secrets, sys
log = pathlib.Path(sys.argv[1])
for line in sys.stdin:
    request = json.loads(line)
    if "id" not in request: continue
    method = request["method"]
    if method == "initialize":
        result = {"protocolVersion":request["params"]["protocolVersion"],
                  "capabilities":{"tools":{}}, "serverInfo":{"name":"fixture","version":"1"}}
    elif method == "tools/list":
        result = {"tools":[{"name":"proof","description":"Return a random test proof", "inputSchema":{"type":"object","properties":{},"additionalProperties":False}}]}
    elif method == "tools/call" and request["params"]["name"] == "proof":
        proof = secrets.token_hex(24)
        with log.open("a") as output: output.write(proof + "\\n")
        result = {"content":[{"type":"text","text":proof}]}
    else:
        print(json.dumps({"jsonrpc":"2.0","id":request["id"],"error":{"code":-32601,"message":"Unknown method"}}),flush=True)
        continue
    print(json.dumps({"jsonrpc":"2.0","id":request["id"],"result":result}),flush=True)
'''
with tempfile.TemporaryDirectory(prefix="mcpdeck-live-") as temporary:
    root = pathlib.Path(temporary)
    os.chmod(root, 0o700)
    env = dict(PATH=os.environ.get("PATH", ""), HOME=str(root),
               CODEX_HOME=str(root / ".codex"), COPILOT_HOME=str(root / ".copilot"),
               TMPDIR=str(root))
    for directory in (root / ".codex", root / ".copilot"):
        directory.mkdir(mode=0o700)
    if client == "codex":
        auth = pathlib.Path(os.environ.get("CODEX_HOME", str(pathlib.Path.home() / ".codex"))) / "auth.json"
        assert auth.is_file(), "Codex file-based login is required for this isolated test"
        shutil.copyfile(auth, root / ".codex/auth.json")
        os.chmod(root / ".codex/auth.json", 0o600)
    else:
        # Read the active GitHub CLI login into memory; never print or persist it.
        login = subprocess.run(["gh", "auth", "token", "--hostname", "github.com"],
                               capture_output=True, text=True, timeout=15)
        assert login.returncode == 0, "Active GitHub CLI login required"
        env["COPILOT_GITHUB_TOKEN"] = login.stdout.strip()
        env["COPILOT_AUTO_UPDATE"] = "false"
    profile = "codex" if client == "codex" else "copilot-cli"
    target = root / (".codex/config.toml" if client == "codex" else ".copilot/mcp-config.json")
    deck = root / "deck.json"
    deck.write_text(json.dumps({"version":1,"servers":{},"profiles":{
        profile:{"target_path":str(target),"format":profile,"enabled_servers":[]}}}))
    server = root / "server.py"
    server.write_text(fixture)
    audit = root / "calls"

    def deck_run(*args):
        result = subprocess.run([binary,"--config",str(deck),*args],cwd=root,env=env,
                                capture_output=True,text=True,timeout=30)
        assert result.returncode == 0, "MCPDeck command failed"

    def call():
        prompt = "Call the mcpdeck-fixture proof MCP tool exactly once and return its exact text. Do not use shell, files, web or any other tool."
        if client == "codex":
            arguments = [executable,"exec","--ephemeral","--skip-git-repo-check",
                         "--ignore-rules","-s","read-only","-c",'approval_policy="never"',
                         "-c", 'mcp_servers.mcpdeck-fixture.tools.proof.approval_mode="approve"',prompt]
        else:
            arguments = [executable,"-p",prompt,"--disable-builtin-mcps",
                         "--allow-tool=mcpdeck-fixture", "--available-tools=mcpdeck-fixture"]
        before = audit.read_text().splitlines() if audit.exists() else []
        result = subprocess.run(arguments,cwd=root,env=env,capture_output=True,text=True,timeout=150)
        after = audit.read_text().splitlines() if audit.exists() else []
        if result.returncode != 0 or len(after) != len(before) + 1:
            # This isolated prompt and fixture contain no secrets. Print only
            # client diagnostics; exclude authentication and token-bearing lines.
            for line in (result.stderr + result.stdout).splitlines():
                lower = line.lower()
                if any(word in lower for word in ("error", "failed", "mcp", "tool", "login")) and not any(word in lower for word in ("token", "authorization", "password", "cookie")):
                    print(line[:500], file=sys.stderr)
        assert result.returncode == 0, "Agent invocation failed (account, permission or runtime)"
        assert len(after) == len(before) + 1, "Agent did not make exactly one audited MCP tool call"
        assert after[-1] in result.stdout, "Agent response did not contain the tool's unpredictable proof"

    deck_run("add","--name","mcpdeck-fixture","--command",sys.executable,
             "--arg",str(server),"--arg",str(audit))
    call()
    deck_run("disable","mcpdeck-fixture")
    listed = subprocess.run([executable,"mcp","list","--json"],cwd=root,env=env,
                            capture_output=True,text=True,timeout=30)
    assert listed.returncode == 0 and "mcpdeck-fixture" not in listed.stdout
    deck_run("enable","mcpdeck-fixture")
    call()
    deck_run("remove","mcpdeck-fixture","--yes")
    listed = subprocess.run([executable,"mcp","list","--json"],cwd=root,env=env,
                            capture_output=True,text=True,timeout=30)
    assert listed.returncode == 0 and "mcpdeck-fixture" not in listed.stdout
print(f"PASS: {client}: two real audited tool calls; disable, enable and remove verified")
