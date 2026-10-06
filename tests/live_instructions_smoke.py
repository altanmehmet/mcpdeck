"""Optional paid-account instruction uptake test; private temporary HOME only.

Usage: python3 tests/live_instructions_smoke.py ./mcpdeck codex|copilot
Never prints login files, instruction values or provider response contents.
"""
import json
import os
import pathlib
import secrets
import shutil
import subprocess
import sys
import tempfile

binary = str(pathlib.Path(sys.argv[1]).resolve())
client = sys.argv[2]
assert client in ("codex", "copilot")
executable = shutil.which(client)
assert executable, "Client is not installed"
with tempfile.TemporaryDirectory(prefix="mcpdeck-live-instructions-") as temporary:
    root = pathlib.Path(temporary)
    root.chmod(0o700)
    env = dict(PATH=os.environ.get("PATH", ""), HOME=str(root),
               CODEX_HOME=str(root / ".codex"), COPILOT_HOME=str(root / ".copilot"),
               TMPDIR=str(root))
    for directory in (root / ".codex", root / ".copilot"):
        directory.mkdir(mode=0o700)
    if client == "codex":
        auth = pathlib.Path(os.environ.get("CODEX_HOME", str(pathlib.Path.home() / ".codex"))) / "auth.json"
        assert auth.is_file(), "Codex file-based login required"
        shutil.copyfile(auth, root / ".codex/auth.json")
        (root / ".codex/auth.json").chmod(0o600)
    else:
        login = subprocess.run(["gh", "auth", "token", "--hostname", "github.com"],
                               capture_output=True, text=True, timeout=15)
        assert login.returncode == 0, "Active GitHub CLI login required"
        env["COPILOT_GITHUB_TOKEN"] = login.stdout.strip()
        env["COPILOT_AUTO_UPDATE"] = "false"
    profile = "codex" if client == "codex" else "copilot-cli"
    target = root / (".codex/config.toml" if client == "codex" else ".copilot/mcp-config.json")
    target.write_text("" if client == "codex" else '{"mcpServers":{}}')
    deck = root / "deck.json"
    deck.write_text(json.dumps(dict(version=1, servers={}, profiles={
        profile: dict(target_path=str(target), format=profile, enabled_servers=[])})))

    def manage(*arguments):
        result = subprocess.run([binary, "--config", str(deck), "instructions", *arguments],
                                cwd=root, env=env, capture_output=True, text=True, timeout=30)
        assert result.returncode == 0, "MCPDeck instruction command failed"

    prompt = ("What is MCPDECK_SMOKE_REPLY assigned in your loaded personal instructions? "
              "Return only its exact value, or ABSENT if it is not assigned. Do not use any tools.")

    def verify(expected):
        if client == "codex":
            answer = root / "answer.txt"
            answer.unlink(missing_ok=True)
            arguments = [executable, "exec", "--ephemeral", "--skip-git-repo-check",
                         "--ignore-user-config", "--ignore-rules", "-s", "read-only",
                         "-c", 'approval_policy="never"', "--output-last-message", str(answer), prompt]
        else:
            arguments = [executable, "-s", "-p", prompt, "--disable-builtin-mcps",
                         "--available-tools=mcpdeck-instruction-smoke-no-tools",
                         "--deny-tool=shell", "--deny-tool=write", "--deny-tool=url",
                         "--no-auto-update", "--no-ask-user"]
        result = subprocess.run(arguments, cwd=root, env=env, capture_output=True,
                                text=True, timeout=150)
        assert result.returncode == 0, "Agent invocation failed; output withheld"
        actual = answer.read_text() if client == "codex" else result.stdout
        assert actual.strip() == expected, "Agent did not return the loaded instruction value; output withheld"

    for stage in ("add", "update"):
        proof = secrets.token_hex(24)
        manage("set", "For the instruction loading smoke check only: MCPDECK_SMOKE_REPLY = " + proof +
               ". If asked for this value, return only the value. Do not use tools for this check.")
        verify(proof)
        print("PASS:", client, "new session follows", stage, "of shared instruction")
    manage("clear", "--yes")
    verify("ABSENT")
    print("PASS:", client, "new session no longer receives removed instruction")
