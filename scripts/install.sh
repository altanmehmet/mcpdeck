#!/bin/sh
set -eu
base=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
prefix="${HOME}/.local"
if [ "${1:-}" = --prefix ]; then
  test "$#" = 2 || { echo 'Usage: install.sh [--prefix DIRECTORY]' >&2; exit 1; }
  prefix=$2
elif [ "$#" != 0 ]; then
  echo 'Usage: install.sh [--prefix DIRECTORY]' >&2; exit 1
fi
source_bin="$base/mcpdeck"
if [ ! -f "$source_bin" ]; then source_bin="$base/../mcpdeck"; fi
test -x "$source_bin" || { echo 'Compiled mcpdeck binary not found beside installer or in project root.' >&2; exit 1; }
"$source_bin" --version
mkdir -p "$prefix/bin"
temporary=$(mktemp "$prefix/bin/.mcpdeck-install.XXXXXX")
trap 'rm -f "$temporary"' EXIT HUP INT TERM
cp "$source_bin" "$temporary"
chmod 755 "$temporary"
mv -f "$temporary" "$prefix/bin/mcpdeck"
printf '\nInstalled: %s/bin/mcpdeck\nRun: %s/bin/mcpdeck\n' "$prefix" "$prefix"
case ":${PATH:-}:" in
  *":$prefix/bin:"*) printf 'You can also run: mcpdeck\n' ;;
  *) printf 'Add %s/bin to PATH to launch with just mcpdeck.\n' "$prefix" ;;
esac
