"""Real PTY smoke test: natural-language form, mouse sync, toggles and removal.

Usage: python3 tests/tui_smoke.py /absolute/path/to/mcpdeck
All agent files live in a temporary directory; no external MCP is started.
"""
import fcntl
import json
import os
import pathlib
import pty
import select
import struct
import subprocess
import sys
import tempfile
import termios
import time


def run(binary):
    with tempfile.TemporaryDirectory(prefix="mcpdeck-tui-") as folder:
        root = pathlib.Path(folder)
        profiles = {}
        for name, fmt, remote in [
            ("cursor", "", "url"), ("claude", "", "stdio"),
            ("claude-code", "", "http"), ("codex", "codex", ""),
            ("copilot", "vscode", ""), ("copilot-cli", "copilot-cli", ""),
            ("windsurf", "", "serverUrl"), ("antigravity", "", "serverUrl"),
        ]:
            profiles[name] = dict(target_path=str(root / (name + ".config")),
                                  enabled_servers=[], format=fmt, remote_format=remote)
        config = root / "deck.json"
        servers = {
            "a-local": {"command": "echo", "args": ["fixture"]},
            "b-remote": {"url": "https://example.com/mcp", "headers": {
                "Authorization": "Bearer ${MCPDECK_PTY_TOKEN}"}},
        }
        config.write_text(json.dumps(dict(version=1, servers=servers, profiles=profiles)))
        master, slave = pty.openpty()
        fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 30, 120, 0, 0))
        env = dict(os.environ, HOME=str(root), TERM="xterm-256color", NO_COLOR="1",
                   MCPDECK_PTY_TOKEN="fixture$literal")
        process = subprocess.Popen([binary, "--config", str(config)], stdin=slave,
                                   stdout=slave, stderr=slave, env=env, start_new_session=True)
        os.close(slave)
        output = bytearray()

        def drain():
            if select.select([master], [], [], 0.05)[0]:
                try:
                    output.extend(os.read(master, 65536))
                except OSError:
                    pass

        def wait(predicate):
            deadline = time.monotonic() + 10
            while time.monotonic() < deadline:
                drain()
                if predicate():
                    return
                if process.poll() is not None:
                    raise AssertionError("TUI exited unexpectedly")
            raise AssertionError("TUI action did not complete: " + repr(bytes(output[-600:])))

        def text(value):
            wait(lambda: value.encode() in output)

        def click(x, y):
            os.write(master, ("\x1b[<0;%d;%dM\x1b[<0;%d;%dm" % (x, y, x, y)).encode())

        def paste(value):
            os.write(master, b"\x1b[200~" + value.encode() + b"\x1b[201~")

        def deck():
            return json.loads(config.read_text())

        try:
            text("MCPDECK")
            assert b"?1002h" in output or b"?1003h" in output, "mouse reporting not enabled"
            click(3, 3)
            text("MCPDECK / NEW MCP")
            paste("oracle-fixture")
            os.write(master, b"\t")
            paste("Use the existing local runtime. Ask for credentials privately.")
            text("Use the existing local runtime")
            os.write(master, b"\x1b")
            for _ in range(4):
                drain()
            click(16, 4)  # Enable selected local MCP for all agents.
            wait(lambda: all("a-local" in p["enabled_servers"] for p in deck()["profiles"].values()))
            click(30, 7)  # Select remote MCP without toggling it.
            click(16, 4)
            wait(lambda: all(set(p["enabled_servers"]) == {"a-local", "b-remote"}
                             for p in deck()["profiles"].values()))
            wait(lambda: all(pathlib.Path(p["target_path"]).exists() and "b-remote" in pathlib.Path(p["target_path"]).read_text()
                             for p in deck()["profiles"].values()))
            click(30, 6)  # Select the local MCP again.
            for name, profile in deck()["profiles"].items():
                saved = pathlib.Path(profile["target_path"]).read_text()
                assert "a-local" in saved and "b-remote" in saved and "fixture$literal" in saved, name
            assert b"fixture$literal" not in output, "credential shown in terminal"
            click(24, 6)
            wait(lambda: "a-local" not in deck()["profiles"]["cursor"]["enabled_servers"])
            assert "a-local" in deck()["profiles"]["codex"]["enabled_servers"]
            click(32, 4)
            wait(lambda: all("a-local" not in p["enabled_servers"] for p in deck()["profiles"].values()))
            os.write(master, b"?")
            text("MCPDECK / HELP")
            os.write(master, b"\x1b")
            for _ in range(4):
                drain()
            os.write(master, b"x")
            text("MCPDECK / REMOVE MCP")
            os.write(master, b"y")
            wait(lambda: "a-local" not in deck()["servers"])
            for profile in deck()["profiles"].values():
                assert "a-local" not in pathlib.Path(profile["target_path"]).read_text()
            os.write(master, b"q")
            wait(lambda: process.poll() is not None)
            assert process.returncode == 0
            print("PASS: real PTY natural-language form, eight-agent sync, single/all toggles, secret-safe output, removal, help and exit")
        finally:
            if process.poll() is None:
                process.terminate()
                process.wait(timeout=5)
            os.close(master)


if __name__ == "__main__":
    run(str(pathlib.Path(sys.argv[1]).resolve()))
