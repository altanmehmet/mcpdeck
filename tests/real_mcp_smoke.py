"""Optional read-only checks of existing Oracle/Sentry through MCPDeck.

python3 tests/real_mcp_smoke.py ./mcpdeck
python3 tests/real_mcp_smoke.py ./mcpdeck --oracle-connection SAVED_TEST_CONNECTION

Uses a private temporary deck and never rewrites real agent configurations or
prints credentials/account results. Explicit connection selection runs SELECT 1
in a separate SQLcl session. Server-side audit logging may record these calls.
Requires a managed Oracle server and a Copilot CLI Sentry entry.
"""
import argparse
import json
import os
import pathlib
import queue
import subprocess
import tempfile
import threading

from real_mcp_results import validate_oracle_health, validate_oracle_result

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("binary", type=pathlib.Path)
parser.add_argument("--deck", type=pathlib.Path,
                    default=pathlib.Path.home() / ".config/mcpdeck/deck.json")
parser.add_argument("--sentry-config", type=pathlib.Path,
                    default=pathlib.Path.home() / ".copilot/mcp-config.json")
parser.add_argument("--oracle-connection", help="Saved TEST connection for SELECT 1 FROM DUAL")
options = parser.parse_args()
actual = json.loads(options.deck.read_text())
external = json.loads(options.sentry_config.read_text())
sentry = external["mcpServers"]["sentry"]
connection_fields = ("command", "args", "env", "url", "headers", "transport", "variables")
servers = {
    "oracle": {k: v for k, v in actual["servers"]["oracle"].items() if k in connection_fields},
    "sentry": {k: v for k, v in sentry.items() if k in connection_fields},
}

with tempfile.TemporaryDirectory(prefix="mcpdeck-real-mcp-") as temporary:
    root = pathlib.Path(temporary)
    os.chmod(root, 0o700)
    deck = root / "deck.json"
    deck.write_text(json.dumps({"version": 1, "servers": servers, "profiles": {
        "probe": {"target_path": str(root / "unused.json"), "enabled_servers": list(servers)}
    }}))
    deck.chmod(0o600)
    proc = subprocess.Popen([str(options.binary.resolve()), "--config", str(deck),
                             "bridge", "--profile", "probe"], stdin=subprocess.PIPE,
                            stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, text=True)
    replies = queue.Queue()

    def read_responses():
        for line in proc.stdout:
            try:
                replies.put(json.loads(line))
            except ValueError:
                replies.put({"id": -1})
        replies.put({"id": -1})

    threading.Thread(target=read_responses, daemon=True).start()

    def send(method, params=None, request_id=1):
        message = {"jsonrpc": "2.0", "method": method}
        if request_id is not None:
            message["id"] = request_id
        if params is not None:
            message["params"] = params
        proc.stdin.write(json.dumps(message) + "\n")
        proc.stdin.flush()
        if request_id is None:
            return
        while True:
            response = replies.get(timeout=90)
            assert response.get("id") != -1, "Bridge closed or returned an invalid response"
            if response.get("id") == request_id:
                return response

    def call(name, arguments, request_id):
        response = send("tools/call", {"name": name, "arguments": arguments}, request_id)
        assert "result" in response and not response["result"].get("isError"), name + " failed"
        if name.startswith("oracle__"):
            validate_oracle_result(response["result"])
        return response["result"]

    try:
        initialized = send("initialize", {
            "protocolVersion": "2024-11-05", "capabilities": {},
            "clientInfo": {"name": "MCPDeck integration test", "version": "1"}
        })
        assert "result" in initialized, "Bridge initialization failed"
        send("notifications/initialized", request_id=None)
        response = send("tools/list", {}, 2)
        assert "result" in response, "Real MCP tools/list failed"
        tools = response["result"]["tools"]
        print("PASS: MCPDeck initializes Oracle/Sentry and exposes", len(tools), "real tools")
        metadata = {"model": "GPT-5", "mcp_client": "Codex"}
        call("oracle__list-connections", metadata, 3)
        print("PASS: Oracle saved connection discovery through MCPDeck")
        if options.oracle_connection:
            call("oracle__connect", dict(metadata, connection_name=options.oracle_connection), 6)
            try:
                health = call("oracle__run-sql", dict(metadata, sql="SELECT 1 AS MCPDECK_HEALTH FROM DUAL"), 7)
                validate_oracle_health(health)
                print("PASS: Oracle SELECT 1 FROM DUAL through MCPDeck")
            finally:
                call("oracle__disconnect", metadata, 8)
        identity = call("sentry__search_sentry_tools", {
            "query": "whoami current authenticated user", "limit": 1
        }, 4)
        assert "whoami" in json.dumps(identity), "Sentry identity tool not found"
        call("sentry__execute_sentry_tool", {"name": "whoami", "arguments": {}}, 5)
        print("PASS: Sentry authenticated whoami through MCPDeck; account details not printed")
    finally:
        proc.stdin.close()
        try:
            proc.wait(timeout=10)
        except subprocess.TimeoutExpired:
            proc.kill()
            proc.wait()
