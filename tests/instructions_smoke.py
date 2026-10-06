"""Unix binary/PTY instruction lifecycle; temporary HOME, no provider account.

Usage: python3 tests/instructions_smoke.py /absolute/path/to/mcpdeck
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

binary = str(pathlib.Path(sys.argv[1]).resolve())
with tempfile.TemporaryDirectory(prefix="mcpdeck-instructions-") as temporary:
    root = pathlib.Path(temporary)
    env = dict(os.environ, HOME=str(root), CODEX_HOME=str(root / ".codex"),
               COPILOT_HOME=str(root / ".copilot"), TERM="xterm-256color", NO_COLOR="1")
    names = ("codex", "claude-code", "copilot-cli", "copilot", "gemini-cli",
             "antigravity", "qwen-code", "opencode", "cursor", "windsurf",
             "kiro", "cline", "cline-vscode-code", "claude", "zed", "trae")
    profiles = {}
    for name in names:
        config = root / "configs" / (name + ".json")
        config.parent.mkdir(exist_ok=True)
        config.write_text("{}")
        profiles[name] = {"target_path": str(config), "enabled_servers": []}
    deck = root / "deck.json"
    deck.write_text(json.dumps(dict(version=1, servers={}, profiles=profiles)))
    source = root / "instructions.md"
    native = root / ".codex/AGENTS.md"
    native.parent.mkdir()
    original = "# Personal rules\nKeep existing private guidance.\n"
    native.write_text(original)
    project = root / "project/AGENTS.md"
    project.parent.mkdir()
    project.write_text("Project instructions must stay unchanged.\n")
    modular = root / ".claude/rules/existing.md"
    modular.parent.mkdir(parents=True)
    modular.write_text("Existing modular rule.\n")

    def run(*args, text=None, success=True):
        result = subprocess.run([binary, "--config", str(deck), "instructions", *args],
                                cwd=project.parent, env=env, input=text,
                                capture_output=True, text=True, timeout=30)
        assert (result.returncode == 0) == success, "Instruction command unexpected status: " + args[0]
        return result.stdout

    files = run("files")
    targets = {pathlib.Path(line.split("\t")[1]) for line in files.splitlines()}
    assert str(modular) in files and str(project) not in files
    text = "Shared instruction: Türkçe, English, 日本語 🚀."
    output = run("set", "--file", "-", text=text)
    assert "claude: manual" in output and "zed: manual" in output
    assert len(targets - {modular}) == 11, "Unexpected shared global target inventory"
    for path in targets - {modular}:
        assert text in path.read_text(), "Missing shared instruction"
    assert native.read_text().startswith(original)
    assert native.with_name(native.name + ".mcpdeck-backup").read_text() == original
    assert run("show") == text
    tracked = targets | {source} | {pathlib.Path(str(p) + ".mcpdeck-backup") for p in targets | {source}}
    before = {p: p.read_bytes() for p in tracked if p.exists()}
    run("sync")
    assert all(p.read_bytes() == content for p, content in before.items())
    run("set", "Shared instruction version two.")
    assert all(text not in p.read_text() for p in targets - {modular})
    addition = "Keep the earlier shared instruction too."
    run("append", addition)
    run("append", addition)
    assert run("show").count(addition) == 1
    current = run("show", "--agent", "codex")
    run("set", "--agent", "codex", "--file", "-", text=current + "\nNative edit only.\n")
    assert "Native edit only." in native.read_text()
    assert all("Native edit only." not in p.read_text() for p in targets - {native, modular})
    run("set", "--agent", "codex", "--file", "-", text="Erase shared block", success=False)
    run("clear", success=False)
    for invalid in ("x" * (24 * 1024 + 1), "invalid\x00instruction"):
        unchanged = source.read_bytes()
        run("set", "--file", "-", text=invalid, success=False)
        assert source.read_bytes() == unchanged
    run("clear", "--yes")
    assert source.read_text() == ""
    assert all("mcpdeck:global-instructions:begin" not in p.read_text() for p in targets)
    assert original in native.read_text() and "Native edit only." in native.read_text()
    assert project.read_text() == "Project instructions must stay unchanged.\n"
    assert modular.read_text() == "Existing modular rule.\n"
    native_before = native.read_bytes()
    broken = "<!-- mcpdeck:global-instructions:begin -->\nBroken fixture"
    native.write_text(broken)
    run("set", "Retryable shared fixture.", success=False)
    assert source.read_text() == "Retryable shared fixture."
    assert native.read_text() == broken
    assert "Retryable shared fixture." in (root / ".qwen/QWEN.md").read_text()
    native.write_bytes(native_before)
    run("sync")
    assert "Retryable shared fixture." in native.read_text()
    run("clear", "--yes")
    print("PASS: 13 automatic profiles / 11 shared paths; CLI Unicode, existing files, native edit, update, append, retry, clear and validation")

    master, slave = pty.openpty()
    fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 30, 120, 0, 0))
    process = subprocess.Popen([binary, "--config", str(deck), "instructions"],
                               cwd=project.parent, env=env, stdin=slave,
                               stdout=slave, stderr=slave, start_new_session=True)
    os.close(slave)
    output = bytearray()

    def wait(predicate):
        deadline = time.monotonic() + 10
        while time.monotonic() < deadline:
            if select.select([master], [], [], 0.05)[0]:
                try:
                    output.extend(os.read(master, 65536))
                except OSError:
                    pass
            if predicate():
                return
            if process.poll() is not None:
                assert predicate(), "Instruction TUI exited unexpectedly"
                return
        raise AssertionError("Instruction TUI action timed out")

    def visible(text):
        wait(lambda: text.encode() in output)

    def click(x, y):
        os.write(master, ("\x1b[<0;%d;%dM\x1b[<0;%d;%dm" % (x, y, x, y)).encode())

    def paste(text):
        os.write(master, b"\x1b[200~" + text.encode() + b"\x1b[201~")

    try:
        visible("MCPDECK / PERSONAL INSTRUCTIONS")
        draft = "Terminal test: Türkçe 🚀."
        paste(draft)
        visible("Terminal test:")
        click(20, 3)
        visible("Save & distribute")
        assert source.read_text() == "", "Review unexpectedly wrote instructions"
        click(20, 3)
        wait(lambda: source.read_text() == draft)
        visible("Saved. Start new agent sessions")
        os.write(master, b"\x1b")  # Leave review.
        time.sleep(0.2)  # Let the terminal distinguish Esc from an Alt chord.
        os.write(master, b"\x0e")  # Ctrl+N adds one shared instruction.
        visible("MCPDECK / ADD ONE SHARED INSTRUCTION")
        paste("Added in terminal.")
        visible("Added in terminal.")
        output.clear()
        os.write(master, b"\x13")
        visible("Save & distribute")
        output.clear()
        os.write(master, b"\x13")
        wait(lambda: "Added in terminal." in source.read_text())
        visible("Saved. Start new agent sessions")
        assert draft in source.read_text()
        os.write(master, b"\x19")  # Ctrl+Y emits OSC 52 for terminal clipboard.
        wait(lambda: b"\x1b]52;" in output)
        os.write(master, b"\x1b")
        time.sleep(0.2)
        os.write(master, b"\x05")  # Ctrl+E opens existing files.
        visible("MCPDECK / EXISTING GLOBAL INSTRUCTIONS")
        documents = run("files").splitlines()
        codex_index = next(i for i, line in enumerate(documents) if line.startswith("codex\t"))
        click(10, codex_index + 6)
        visible("MCPDECK / EXISTING INSTRUCTIONS / codex")
        before_native_edit = {p: p.read_bytes() for p in targets - {native}}
        prefix = "Edited personal text in terminal.\n"
        paste(prefix)
        output.clear()
        click(20, 3)
        visible("Save this file")
        assert not native.read_text().startswith(prefix), "Native review unexpectedly wrote"
        click(20, 3)
        wait(lambda: native.read_text().startswith(prefix))
        visible("Saved. Start new agent sessions")
        assert all(p.read_bytes() == raw for p, raw in before_native_edit.items())
        shared_before_import = source.read_bytes()
        click(8, 5)  # Use for all loads a draft, without distributing it.
        visible("Loaded as shared instructions")
        assert source.read_bytes() == shared_before_import
        os.write(master, b"\x1b")
        wait(lambda: process.poll() is not None)
        assert process.returncode == 0
        assert project.read_text() == "Project instructions must stay unchanged.\n"
        print("PASS: real PTY mouse review/save, keyboard append, Unicode, existing native edit, use-for-all draft/cancel, OSC 52 emission and exit")
    finally:
        if process.poll() is None:
            process.terminate()
            process.wait(timeout=5)
        os.close(master)
