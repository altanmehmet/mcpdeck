"""Verify publisher signatures reject tampered manifests, archives and wrong keys."""
import pathlib
import subprocess
import tempfile

project = pathlib.Path(__file__).resolve().parents[1]

def run(*args, success=True):
    result = subprocess.run(args, capture_output=True, text=True, timeout=20)
    assert (result.returncode == 0) == success, result.stderr

with tempfile.TemporaryDirectory(prefix="mcpdeck-signatures-") as temporary:
    root = pathlib.Path(temporary)
    key = root / "key"
    wrong = root / "wrong"
    for target in (key, wrong):
        run("ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", str(target))
    signers = root / "allowed_signers"
    signers.write_text("mcpdeck-release " + key.with_suffix(".pub").read_text())
    archives = []
    for platform in ("darwin", "linux"):
        for arch in ("amd64", "arm64"):
            archive = root / f"mcpdeck-{platform}-{arch}.tar.gz"
            archive.write_bytes(b"test package")
            archives.append(archive)
    sign = str(project / "scripts/sign-packages.sh")
    verify = str(project / "scripts/verify-packages.sh")
    run("sh", sign, str(root), str(key))
    run("sh", verify, str(root), str(signers))
    for arch in ("amd64", "arm64"):
        (root / f"mcpdeck-windows-{arch}.zip").write_bytes(b"Windows test package")
    run("sh", sign, str(root), str(key), "--windows")
    run("sh", verify, str(root), str(signers))
    (root / "mcpdeck-windows-arm64.zip").write_bytes(b"tampered")
    run("sh", verify, str(root), str(signers), success=False)
    (root / "mcpdeck-windows-arm64.zip").write_bytes(b"Windows test package")
    archives[0].write_bytes(b"modified package")
    run("sh", verify, str(root), str(signers), success=False)
    archives[0].write_bytes(b"test package")
    manifest = root / "SHA256SUMS"
    original = manifest.read_bytes()
    manifest.write_bytes(original + b"\n")
    run("sh", verify, str(root), str(signers), success=False)
    manifest.write_bytes(original)
    signers.write_text("mcpdeck-release " + wrong.with_suffix(".pub").read_text())
    run("sh", verify, str(root), str(signers), success=False)
    signers.write_text("mcpdeck-release " + key.with_suffix(".pub").read_text())
    manifest.write_text("0" * 64 + "  ../outside.tar.gz\n")
    (root / "SHA256SUMS.sig").unlink()
    run("ssh-keygen", "-Y", "sign", "-f", str(key), "-n", "mcpdeck-release", str(manifest))
    run("sh", verify, str(root), str(signers), success=False)
print("PASS: valid signature; tampered archive/manifest, wrong key and unsafe path rejected")
