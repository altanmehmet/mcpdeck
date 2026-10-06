"""Interactive install/repair console in a real Unix PTY; isolated fixtures only.

python3 tests/install_console_smoke.py ./mcpdeck
"""
import fcntl
import json
import os
import pathlib
import pty
import select
import signal
import struct
import subprocess
import sys
import tempfile
import termios
import time

binary = str(pathlib.Path(sys.argv[1]).resolve())
with tempfile.TemporaryDirectory(prefix="mcpdeck-chat-console-") as temporary:
    root = pathlib.Path(temporary)
    config = root / "deck.json"
    target = root / "agent.json"
    target.write_text('{"mcpServers":{"keep":{"command":"existing"}}}')
    original = target.read_bytes()
    config.write_text(json.dumps(dict(version=1, servers={}, profiles={
        "cursor": dict(target_path=str(target), enabled_servers=[])})))
    server = root / "server.py"
    server.write_text('''import json, sys
for line in sys.stdin:
    request = json.loads(line)
    if "id" not in request: continue
    if request["method"] == "initialize":
        result = {"protocolVersion":request["params"]["protocolVersion"], "capabilities":{"tools":{}}, "serverInfo":{"name":"fixture","version":"1"}}
    else:
        result = {"tools":[{"name":"hello", "inputSchema":{"type":"object","properties":{}}}]}
    print(json.dumps({"jsonrpc":"2.0","id":request["id"],"result":result}),flush=True)
''')
    recipe = dict(version=1, name="console-demo", summary="Use the existing Python runtime. The plan is ready for review.",
                  sources=["https://example.com/official-fixture-docs"], requirements=[sys.executable], manual=[],
                  steps=[dict(description="Create a local test marker", command=sys.executable,
                              args=["-c", "from pathlib import Path; Path('installed').write_text('ok')"], directory=".")],
                  connection=json.dumps(dict(command=sys.executable, args=[str(server)], env={"TOKEN":"${CONSOLE_PRIVATE_TOKEN}"})))
    (root / "recipe.json").write_text(json.dumps(recipe))
    planner = root / "planner"
    planner.write_text("#!" + sys.executable + "\n" + '''import json, pathlib, sys, time
root=pathlib.Path(__file__).parent
prompt=sys.stdin.read()
with (root/'prompts.txt').open('a') as file: file.write(prompt+'\\n')
if 'Trigger failure' in prompt: print('controlled fixture failure', file=sys.stderr); sys.exit(1)
if 'Slow request' in prompt: time.sleep(30)
time.sleep(0.3)
plan=json.loads((root/'recipe.json').read_text())
if 'Explain the runtime' in prompt: plan['summary']='The existing Python runtime is reused. No new Python installation is needed.'
output=pathlib.Path(sys.argv[sys.argv.index('--output-last-message')+1])
output.write_text(json.dumps(plan))
''')
    planner.chmod(0o700)
    env = dict(os.environ, HOME=str(root), CODEX_HOME=str(root / ".codex"),
               COPILOT_HOME=str(root / ".copilot"), TERM="xterm-256color", NO_COLOR="1")

    class Session:
        def __init__(self, arguments, width=120, height=30):
            self.master, slave = pty.openpty()
            fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", height, width, 0, 0))
            self.process = subprocess.Popen([binary, "--config", str(config), *arguments],
                                            cwd=root, env=env, stdin=slave, stdout=slave,
                                            stderr=slave, start_new_session=True)
            os.close(slave)
            self.output = bytearray()
        def wait(self, predicate):
            deadline = time.monotonic() + 20
            while time.monotonic() < deadline:
                if select.select([self.master], [], [], 0.05)[0]:
                    try: self.output.extend(os.read(self.master, 65536))
                    except OSError: pass
                if predicate(): return
                if self.process.poll() is not None:
                    assert predicate(), "Console exited unexpectedly"
                    return
            raise AssertionError("Console action timed out: " + repr(bytes(self.output[-1000:])))
        def visible(self, text): self.wait(lambda: text.encode() in self.output)
        def key(self, data): os.write(self.master, data)
        def click(self, x, y): self.key(("\x1b[<0;%d;%dM\x1b[<0;%d;%dm" % (x,y,x,y)).encode())
        def resize(self, width, height):
            fcntl.ioctl(self.master, termios.TIOCSWINSZ, struct.pack("HHHH",height,width,0,0))
            os.kill(self.process.pid, signal.SIGWINCH)
        def reply(self, value):
            self.key(b"\x1b[200~" + value.encode() + b"\x1b[201~")
            time.sleep(0.15)
            self.key(b"\r")
        def finish(self, success):
            self.visible("Finished")
            self.key(b"\r")
            self.wait(lambda: self.process.poll() is not None)
            assert (self.process.returncode == 0) == success
        def close(self):
            if self.process.poll() is None:
                self.process.terminate()
                self.process.wait(timeout=5)
            os.close(self.master)

    session = Session(["install", "Install console demo", "--planner", str(planner), "--provider", "codex"])
    try:
        session.visible("MCPDECK / INSTALLATION CHAT")
        session.visible("Use the existing Python runtime")
        session.visible("Message your agent")
        session.resize(72,24)
        time.sleep(0.2)
        session.reply("Explain the runtime")
        session.resize(120,30)
        session.visible("No new Python installation")
        session.reply("Trigger failure")
        session.visible("previous recipe")
        assert target.read_bytes() == original and not (root / "installations").exists()
        session.key(b"\x13")  # Ctrl+S continues to command approval.
        session.visible("Approval required")
        session.visible("Command:")
        assert target.read_bytes() == original
        session.click(52,3)  # Explicit visible Approve action.
        session.visible("Private value")
        secret = "private$test-value"
        session.reply(secret)
        session.visible("Verified: 1 tools")
        session.finish(True)
        assert secret.encode() not in session.output
        assert secret not in (root / "prompts.txt").read_text()
        assert (root / "installations/console-demo/installed").read_text() == "ok"
        assert "console-demo" in target.read_text() and "keep" in target.read_text()
        print("PASS: PTY agent chat/revision/retry, explicit approval, masked secret, actual setup/probe/sync and exit")
    finally: session.close()

    session = Session(["repair", "console-demo", "Fix the fixture", "--planner", str(planner), "--provider", "codex"], width=60, height=22)
    try:
        session.visible("Message your agent")
        session.key(b"\x13")
        session.visible("Approval required")
        before = target.read_bytes()
        session.reply("")  # Empty confirmation must cancel without rewriting settings.
        session.finish(False)
        assert target.read_bytes() == before
        print("PASS: narrow repair console, shared review UI and empty approval cancellation")
    finally: session.close()

    session = Session(["install", "Slow request", "--planner", str(planner), "--provider", "codex"])
    try:
        session.visible("Preparing plan")
        session.key(b"\x03")
        session.finish(False)
        print("PASS: cancellation while planner is busy unblocks and preserves agent settings")
    finally: session.close()
