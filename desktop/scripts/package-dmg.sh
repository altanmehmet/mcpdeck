#!/bin/sh
set -eu
desktop_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
arch=${MCPDECK_ARCH:-$(go env GOARCH)}
case "$arch" in arm64|amd64) ;; *) echo 'Unsupported Mac architecture' >&2; exit 1;; esac
app_dir=${MCPDECK_APP_DIR:-"$desktop_dir/build/bin/MCPDeck.app"}
codesign --verify --deep --strict "$app_dir"
stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT HUP INT TERM
ditto --norsrc --noextattr "$app_dir" "$stage/MCPDeck.app"
ln -s /Applications "$stage/Applications"
archive="$desktop_dir/build/bin/MCPDeck-macos-$arch.zip"
test -f "$archive" || { echo "Create the matching desktop ZIP before the DMG." >&2; exit 1; }
unzip -p "$archive" "MCPDeck-macos-$arch/package.json" > "$stage/package.json"
expected=$(plutil -extract sha256 raw -o - "$stage/package.json")
actual=$(shasum -a 256 "$app_dir/Contents/MacOS/MCPDeck" | cut -d ' ' -f1)
test "$actual" = "$expected" || { echo 'App and architecture package do not match; rebuild the ZIP first.' >&2; exit 1; }
unzip -p "$archive" "MCPDeck-macos-$arch/LICENSE" > "$stage/LICENSE"
unzip -p "$archive" "MCPDeck-macos-$arch/THIRD_PARTY_NOTICES.txt" > "$stage/THIRD_PARTY_NOTICES.txt"
cat > "$stage/Read me.txt" <<'README'
Drag MCPDeck.app onto Applications, then open MCPDeck from Applications.
For installation without system folder access, use your ~/Applications folder.
This package does not change your MCP settings during installation.
After moving an existing app, Sync affected Bridge profiles to update its path.
README
output="$desktop_dir/build/bin/MCPDeck-macos-$arch.dmg"
hdiutil create -volname MCPDeck -srcfolder "$stage" -format UDZO -imagekey zlib-level=9 -ov "$output"
hdiutil verify "$output"
size=$(stat -f %z "$output")
if [ "$size" -gt 20971520 ];then rm -f "$output";echo 'DMG exceeds 20 MiB download budget' >&2;exit 1;fi
shasum -a 256 "$output" > "$output.sha256"
printf 'Created %s\n' "$output"
