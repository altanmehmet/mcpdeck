"""Verify a native archive's checksum, contents and installation into a spaced path."""
import hashlib
import os
import pathlib
import subprocess
import sys
import tarfile
import tempfile

archive = pathlib.Path(sys.argv[1]).resolve()
checksum = pathlib.Path(str(archive) + ".sha256").read_text().split()[0]
assert hashlib.sha256(archive.read_bytes()).hexdigest() == checksum, "Archive checksum mismatch"
with tempfile.TemporaryDirectory(prefix="mcpdeck-package-smoke-") as temporary:
    root = pathlib.Path(temporary)
    unpacked = root / "unpacked"
    unpacked.mkdir()
    with tarfile.open(archive) as bundle:
        required = {"mcpdeck", "install.sh", "README.txt", "AKILLI-KURULUM.md", "INSTRUCTIONS.md", "RECOVERY.md", "LICENSE", "THIRD_PARTY_NOTICES.txt"}
        assert set(bundle.getnames()) == required, "Missing or unexpected archive content"
        assert all(member.isfile() for member in bundle.getmembers()), "Archive contains non-regular entries"
        bundle.extractall(unpacked)
    assert "MIT License" in (unpacked / "LICENSE").read_text()
    notices = (unpacked / "THIRD_PARTY_NOTICES.txt").read_text()
    assert "Go runtime" in notices and "github.com/charmbracelet/bubbletea" in notices
    prefix = root / "installation with spaces"
    env = dict(os.environ, HOME=str(root))
    def run(args):
        result = subprocess.run([str(arg) for arg in args], env=env, cwd=root,
                                text=True, capture_output=True, timeout=30)
        assert result.returncode == 0, result.stderr
        return result.stdout
    expected = run([unpacked / "mcpdeck", "--version"])
    run(["sh", unpacked / "install.sh", "--prefix", prefix])
    installed = prefix / "bin/mcpdeck"
    assert run([installed, "--version"]) == expected, "Installed binary version differs"
    assert "Manage MCP servers" in run([installed, "--help"])
    assert installed.stat().st_mode & 0o777 == 0o755
print("PASS: archive checksum, license/guides, executable version and installation into spaced prefix")
