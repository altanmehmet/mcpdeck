#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
go_bin=${GO_BIN:-go}
platform=$("$go_bin" env GOOS)
architecture=$("$go_bin" env GOARCH)
version=${MCPDECK_VERSION:-0.1.0}
case "$version" in ''|*[!A-Za-z0-9.+_-]*) echo 'Invalid package version.' >&2; exit 1 ;; esac
case "$platform" in darwin|linux) ;; *) echo 'This installer package supports macOS and Linux.' >&2; exit 1 ;; esac
stage=$(mktemp -d "${TMPDIR:-/tmp}/mcpdeck-cli.XXXXXX")
trap 'rm -rf "$stage"' EXIT HUP INT TERM
"$go_bin" build -trimpath -ldflags "-X github.com/altanmehmet/mcpdeck/cmd.version=$version" -o "$stage/mcpdeck" .
cp scripts/install.sh "$stage/install.sh"
cp docs/CLI-BASLANGIC.md "$stage/README.txt"
cp docs/AKILLI-KURULUM.md "$stage/AKILLI-KURULUM.md"
cp docs/INSTRUCTIONS.md "$stage/INSTRUCTIONS.md"
cp docs/RECOVERY.md "$stage/RECOVERY.md"
cp LICENSE "$stage/LICENSE"
sh scripts/package-notices.sh > "$stage/THIRD_PARTY_NOTICES.txt"
mkdir -p dist
archive="mcpdeck-$platform-$architecture.tar.gz"
COPYFILE_DISABLE=1 tar -czf "dist/$archive" -C "$stage" mcpdeck install.sh README.txt AKILLI-KURULUM.md INSTRUCTIONS.md RECOVERY.md LICENSE THIRD_PARTY_NOTICES.txt
(cd dist && shasum -a 256 "$archive" > "$archive.sha256")
printf 'CLI package: dist/%s\n' "$archive"
