#!/bin/sh
# Build a self-contained native macOS app; Node is needed only at build time.
set -eu
if [ "$(uname -s)" != Darwin ]; then
  echo 'This script packages macOS only. Use the Wails build workflow on other systems.' >&2
  exit 1
fi
desktop_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
node_bin=${MCPDECK_NODE:-node}
command -v "$node_bin" >/dev/null || { echo 'Node.js is required to build the frontend.' >&2; exit 1; }
command -v go >/dev/null || { echo 'Go is required to build the desktop backend.' >&2; exit 1; }
cd "$desktop_dir/frontend"
if [ ! -f node_modules/vite/bin/vite.js ]; then
  echo 'Install the locked frontend dependencies first: cd desktop/frontend && npm ci' >&2
  exit 1
fi
"$node_bin" --test tests/*.test.mjs
"$node_bin" node_modules/vite/bin/vite.js build
"$node_bin" scripts/prepare-sites-build.mjs
cd "$desktop_dir"
build_arch=${MCPDECK_ARCH:-$(go env GOARCH)}
case "$build_arch" in arm64|amd64) ;; *) echo "Unsupported architecture" >&2; exit 1;; esac
app_dir=${MCPDECK_APP_DIR:-"$desktop_dir/build/bin/MCPDeck.app"}
mkdir -p "$app_dir/Contents/MacOS" "$app_dir/Contents/Resources"
env GOARCH="$build_arch" go build -trimpath -tags production -ldflags "-s -w" -o "$app_dir/Contents/MacOS/MCPDeck" .
cat > "$app_dir/Contents/Info.plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>CFBundlePackageType</key><string>APPL</string>
<key>CFBundleName</key><string>MCPDeck</string>
<key>CFBundleExecutable</key><string>MCPDeck</string>
<key>CFBundleIdentifier</key><string>io.github.altanmehmet.mcpdeck</string>
<key>CFBundleVersion</key><string>0.1.0</string>
<key>CFBundleShortVersionString</key><string>0.1.0</string>
<key>CFBundleIconFile</key><string>iconfile</string>
<key>LSMinimumSystemVersion</key><string>13.0</string>
<key>NSHighResolutionCapable</key><true/>
<key>NSHumanReadableCopyright</key><string>MIT</string>
</dict></plist>
PLIST
icon_root=$(mktemp -d)
trap 'rm -rf "$icon_root"' EXIT HUP INT TERM
mkdir "$icon_root/App.iconset"
for size in 16 32 128 256 512; do
  sips -z "$size" "$size" build/appicon.png --out "$icon_root/App.iconset/icon_${size}x${size}.png" >/dev/null
  double=$((size * 2))
  sips -z "$double" "$double" build/appicon.png --out "$icon_root/App.iconset/icon_${size}x${size}@2x.png" >/dev/null
done
iconutil -c icns "$icon_root/App.iconset" -o "$app_dir/Contents/Resources/iconfile.icns"
plutil -lint "$app_dir/Contents/Info.plist"
codesign --force --deep --sign - "$app_dir"
codesign --verify --deep --strict "$app_dir"
printf 'Built %s\n' "$app_dir"
