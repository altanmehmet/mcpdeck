#!/bin/sh
# Include licenses for modules linked on each supported operating system.
set -eu
go_bin=${GO_BIN:-go}
list=$(mktemp "${TMPDIR:-/tmp}/mcpdeck-licenses.XXXXXX")
trap 'rm -f "$list" "$list.sorted"' EXIT HUP INT TERM
for platform in darwin linux; do
  for arch in arm64 amd64; do
    GOOS="$platform" GOARCH="$arch" "$go_bin" list -deps \
      -f '{{if .Module}}{{if not .Module.Main}}{{.Module.Path}}|{{.Module.Version}}|{{.Module.Dir}}{{end}}{{end}}' . >> "$list"
  done
done
sort -u "$list" | sed '/^$/d' > "$list.sorted"
printf 'MCPDeck third-party notices\n\nGo runtime and standard library\n\n'
go_root=$("$go_bin" env GOROOT)
runtime_license="$go_root/LICENSE"
# Homebrew stores the Go license beside libexec rather than inside GOROOT.
if [ ! -f "$runtime_license" ]; then runtime_license="$go_root/../LICENSE"; fi
cat "$runtime_license"
while IFS='|' read -r module version directory; do
  printf '\n\n=== %s %s ===\n\n' "$module" "$version"
  found=false
  for name in LICENSE LICENSE.txt LICENSE.md LICENSE-MIT LICENCE COPYING; do
    if [ -f "$directory/$name" ]; then
      cat "$directory/$name"
      found=true
    fi
  done
  if [ "$found" = false ]; then
    printf 'Missing license for %s; package creation stopped.\n' "$module" >&2
    exit 1
  fi
  if [ -f "$directory/NOTICE" ]; then cat "$directory/NOTICE"; fi
done < "$list.sorted"
