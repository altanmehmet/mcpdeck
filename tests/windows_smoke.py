"""Native Windows archive, installer, reviewed MCP install and config lifecycle."""
import hashlib
import json
import os
import pathlib
import re
import subprocess
import sys
import tempfile
import zipfile

assert sys.platform == "win32", "Run this test on native Windows"
archive = pathlib.Path(sys.argv[1]).resolve()
assert hashlib.sha256(archive.read_bytes()).hexdigest() == pathlib.Path(str(archive) + ".sha256").read_text().split()[0]
with tempfile.TemporaryDirectory(prefix="mcpdeck-windows-") as temporary:
    root = pathlib.Path(temporary)
    unpacked = root / "package"
    with zipfile.ZipFile(archive) as bundle:
        required = {"mcpdeck.exe", "install.ps1", "uninstall.ps1", "README.txt", "AKILLI-KURULUM.md", "INSTRUCTIONS.md", "RECOVERY.md", "LICENSE", "THIRD_PARTY_NOTICES.txt"}
        assert set(bundle.namelist()) == required
        bundle.extractall(unpacked)
    notices = (unpacked / 'THIRD_PARTY_NOTICES.txt').read_text(encoding='utf-8')
    assert 'github.com/mattn/go-localereader v0.0.1' in notices
    assert '## License\n\nMIT' in notices and 'Yasuhiro Matsumoto' in notices
    env = dict(os.environ, USERPROFILE=str(root), HOME=str(root),
               APPDATA=str(root / "AppData/Roaming"), LOCALAPPDATA=str(root / "AppData/Local"),
               CODEX_HOME=str(root / ".codex"), COPILOT_HOME=str(root / ".copilot"))

    def run(arguments, data=None, success=True):
        result = subprocess.run([str(arg) for arg in arguments], cwd=root, env=env,
                                input=data, capture_output=True, text=True, timeout=60)
        assert (result.returncode == 0) == success, result.stderr + result.stdout
        assert "fixture-secret" not in result.stdout + result.stderr
        return result.stdout

    prefix = root / "installation with spaces"
    installer = ["powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", unpacked / "install.ps1", "-Prefix", prefix, "-NoPathUpdate"]
    run(installer)
    run(installer)  # Matching version is reused without overwriting a running exe.
    binaries = list((prefix / "versions").glob("*/mcpdeck.exe"))
    assert len(binaries) == 1
    binary = binaries[0]
    assert "mcpdeck version" in run([binary, "--version"])
    assert "Manage MCP servers" in run([binary, "--help"])
    # Directory junctions must not redirect install/uninstall into another path.
    junction = root / "redirected installation"
    ps_quote = lambda value: "'" + str(value).replace("'", "''") + "'"
    run(["powershell", "-NoProfile", "-Command",
         "New-Item -ItemType Junction -Path " + ps_quote(junction) + " -Target " + ps_quote(prefix)])
    try:
        run(["powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File",
             unpacked / "install.ps1", "-Prefix", junction, "-NoPathUpdate"], success=False)
        run(["powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command",
             "& " + ps_quote(unpacked / "uninstall.ps1") + " -Prefix " + ps_quote(junction) + " -NoPathUpdate -Confirm:$false"], success=False)
        assert binary.exists()
    finally:
        os.rmdir(junction)  # Remove only the junction, never its target.
    # The CI runner is disposable. Exercise actual per-user PATH persistence,
    # restoring the registry and process values even if an assertion fails.
    quote = lambda value: "'" + str(value).replace("'", "''") + "'"
    path_check = root / "check-path.ps1"
    path_check.write_text("""$ErrorActionPreference = 'Stop'
$originalUser = [Environment]::GetEnvironmentVariable('Path', 'User')
$originalProcess = $env:Path
try {
    & INSTALLER -Prefix PREFIX
    $first = [Environment]::GetEnvironmentVariable('Path', 'User')
    # Windows PowerShell expands 8.3 paths used by Python's temp directory.
    if (($first -split ';')[0] -ne [IO.Path]::GetFullPath(DESTINATION)) { throw 'User PATH was not updated.' }
    & INSTALLER -Prefix PREFIX
    if ([Environment]::GetEnvironmentVariable('Path', 'User') -ne $first) { throw 'Repeated install duplicated PATH.' }
    if ((Get-Command mcpdeck -CommandType Application).Source -ne [IO.Path]::GetFullPath(BINARY)) { throw 'Current terminal command was not updated.' }
} finally {
    [Environment]::SetEnvironmentVariable('Path', $originalUser, 'User')
    $env:Path = $originalProcess
}
""".replace("INSTALLER", quote(unpacked / "install.ps1")).replace("PREFIX", quote(prefix))
       .replace("DESTINATION", quote(binary.parent)).replace("BINARY", quote(binary)))
    run(["powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", path_check])
    server = root / "fixture.py"
    server.write_text('''import json, sys
for line in sys.stdin:
    request = json.loads(line)
    if "id" not in request: continue
    method = request["method"]
    if method == "initialize": result = {"protocolVersion":request["params"]["protocolVersion"],"capabilities":{"tools":{}},"serverInfo":{"name":"fixture","version":"1"}}
    elif method == "tools/list": result = {"tools":[{"name":"health","inputSchema":{"type":"object"}}]}
    elif method == "tools/call": result = {"content":[{"type":"text","text":"native-windows-proof"}]}
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
        profiles[name] = {"target_path":str(target),"format":fmt,"remote_format":remote,"enabled_servers":[]}
    deck = root / "deck.json"
    deck.write_text(json.dumps({"version":1,"servers":{},"profiles":profiles}))
    plan = root / "plan.json"
    plan.write_text(json.dumps({"version":1,"name":"smoke","summary":"Windows fixture", "sources":[],"requirements":[sys.executable],"manual":[],
        "steps":[{"description":"Create local marker","command":sys.executable,
                  "args":["-c","import pathlib; pathlib.Path('installed').touch()"],"directory":"."}],
        "connection":json.dumps({"command":sys.executable,"args":[str(server)],"env":{"TOKEN":"${SMOKE_TOKEN}"}})}))
    def deck_run(*args, **kwargs): return run([binary,"--config",deck,*args], **kwargs)
    preview = deck_run("install","--plan",plan,"--plan-only")
    digest = re.search(r"Plan ID: ([a-f0-9]{64})",preview).group(1)
    output = deck_run("install","--plan",plan,"--approve",digest,"--values-stdin",data='{"SMOKE_TOKEN":"fixture-secret"}')
    assert "Verified: 1 tools. Updated 8 agent configurations." in output
    assert (root / "installations/smoke/installed").exists()
    for profile in profiles.values():
        target = pathlib.Path(profile["target_path"])
        assert "keep" in target.read_text() and "smoke" in target.read_text()
        assert pathlib.Path(str(target) + ".mcpdeck-backup").exists()
    deck_run("disable","smoke")
    for profile in profiles.values(): assert "smoke" not in pathlib.Path(profile["target_path"]).read_text()
    deck_run("enable","smoke")
    for profile in profiles.values(): assert "smoke" in pathlib.Path(profile["target_path"]).read_text()
    deck_run("remove","smoke","--yes")
    for profile in profiles.values():
        value = pathlib.Path(profile["target_path"]).read_text()
        assert "smoke" not in value and "keep" in value
    # Re-sync migrates owned bridge commands to the running version, with backup.
    deck_run("sync", "--profile", "cursor", "--mode", "bridge")
    cursor_path = pathlib.Path(profiles["cursor"]["target_path"])
    cursor_config = json.loads(cursor_path.read_text())
    cursor_config["mcpServers"]["mcpdeck"]["command"] = "C:\\old-version\\mcpdeck.exe"
    cursor_path.write_text(json.dumps(cursor_config))
    deck_run("sync", "--profile", "cursor")
    migrated = json.loads(cursor_path.read_text())
    assert pathlib.Path(migrated["mcpServers"]["mcpdeck"]["command"]).resolve() == binary.resolve()
    assert migrated["keep"] is True
    assert "old-version" in pathlib.Path(str(cursor_path) + ".mcpdeck-backup").read_text()
    # Uninstall refuses unknown files before deleting any owned file.
    foreign = binary.parent / "user-file.txt"
    foreign.write_text("preserve")
    uninstaller = ["powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command",
                   "& " + quote(unpacked / "uninstall.ps1") + " -Prefix " + quote(prefix) + " -NoPathUpdate -Confirm:$false"]
    run(uninstaller, success=False)
    assert binary.exists() and foreign.read_text() == "preserve"
    foreign.unlink()
    saved_deck = deck.read_bytes()
    uninstall_path_check = root / "check-uninstall-path.ps1"
    uninstall_path_check.write_text("""$ErrorActionPreference = 'Stop'
$originalUser = [Environment]::GetEnvironmentVariable('Path', 'User')
$originalProcess = $env:Path
try {
    & INSTALLER -Prefix PREFIX
    & UNINSTALLER -Prefix PREFIX -Confirm:$false
    # The established installer omits empty PATH segments. Compare the remaining
    # entries in order, including duplicates, without hiding extra version paths.
    $expectedUser = (@($originalUser -split ';' | Where-Object { $_ }) -join ';')
    $expectedProcess = (@($originalProcess -split ';' | Where-Object { $_ }) -join ';')
    $remainingUser = (@([Environment]::GetEnvironmentVariable('Path', 'User') -split ';' | Where-Object { $_ }) -join ';')
    $remainingProcess = (@($env:Path -split ';' | Where-Object { $_ }) -join ';')
    if ($remainingUser -ne $expectedUser) { throw 'Uninstall changed unrelated User PATH entries.' }
    if ($remainingProcess -ne $expectedProcess) { throw 'Uninstall changed unrelated process PATH entries.' }
} finally {
    [Environment]::SetEnvironmentVariable('Path', $originalUser, 'User')
    $env:Path = $originalProcess
}
""".replace("UNINSTALLER", quote(unpacked / "uninstall.ps1"))
       .replace("INSTALLER", quote(unpacked / "install.ps1")).replace("PREFIX", quote(prefix)))
    run(["powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", uninstall_path_check])
    assert not binary.exists() and deck.read_bytes() == saved_deck
print("PASS: native Windows ZIP/install with spaces, idempotent installer, MCP handshake and 8-profile lifecycle")
