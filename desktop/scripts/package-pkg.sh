#!/bin/sh
set -eu
desktop_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
arch=${MCPDECK_ARCH:-$(go env GOARCH)}
case "$arch" in arm64|amd64) ;; *) echo 'Unsupported Mac architecture' >&2; exit 1;; esac
app_dir=${MCPDECK_APP_DIR:-"$desktop_dir/build/bin/MCPDeck.app"}
codesign --verify --deep --strict "$app_dir"
stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT HUP INT TERM
mkdir -p "$stage/root/Applications" "$stage/resources"
ditto --norsrc --noextattr "$app_dir" "$stage/root/Applications/MCPDeck.app"
archive="$desktop_dir/build/bin/MCPDeck-macos-$arch.zip"
test -f "$archive" || { echo 'Create the matching desktop ZIP before the installer.' >&2; exit 1; }
unzip -p "$archive" "MCPDeck-macos-$arch/package.json" > "$stage/package.json"
expected=$(plutil -extract sha256 raw -o - "$stage/package.json")
actual=$(shasum -a 256 "$app_dir/Contents/MacOS/MCPDeck" | cut -d ' ' -f1)
test "$actual" = "$expected" || { echo 'App and architecture package do not match; rebuild the ZIP first.' >&2; exit 1; }
unzip -p "$archive" "MCPDeck-macos-$arch/LICENSE" > "$stage/resources/LICENSE"
unzip -p "$archive" "MCPDeck-macos-$arch/THIRD_PARTY_NOTICES.txt" > "$stage/resources/THIRD_PARTY_NOTICES.txt"
pkgbuild --analyze --root "$stage/root" "$stage/components.plist"
# Install only at the declared location; do not search for and update other copies.
plutil -replace 0.BundleIsRelocatable -bool false "$stage/components.plist"
pkgbuild --root "$stage/root" --component-plist "$stage/components.plist" --identifier io.github.altanmehmet.mcpdeck.desktop --version 0.1.0 --ownership recommended "$stage/component.pkg"
cat > "$stage/distribution.xml" <<'XML'
<?xml version="1.0" encoding="utf-8"?>
<installer-gui-script minSpecVersion="1">
 <title>MCPDeck</title>
 <options customize="never" require-scripts="false"/>
 <choices-outline><line choice="desktop"/></choices-outline>
 <choice id="desktop" visible="false"><pkg-ref id="io.github.altanmehmet.mcpdeck.desktop"/></choice>
 <pkg-ref id="io.github.altanmehmet.mcpdeck.desktop" version="0.1.0">component.pkg</pkg-ref>
</installer-gui-script>
XML
output="$desktop_dir/build/bin/MCPDeck-Setup-macos-$arch.pkg"
productbuild --distribution "$stage/distribution.xml" --resources "$stage/resources" --package-path "$stage" "$output"
pkgutil --payload-files "$output" > "$stage/payload-files.txt"
grep -q 'Applications/MCPDeck.app/Contents/MacOS/MCPDeck' "$stage/payload-files.txt"
size=$(stat -f %z "$output")
if [ "$size" -gt 20971520 ];then rm -f "$output";echo 'PKG exceeds 20 MiB download budget' >&2;exit 1;fi
shasum -a 256 "$output" > "$output.sha256"
printf 'Created %s\n' "$output"
