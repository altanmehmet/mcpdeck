#!/bin/sh
# Sign a manifest covering all supported package archives.
set -eu
if [ "$#" -ne 2 ] && [ "$#" -ne 3 ]; then echo 'Usage: sign-packages.sh DIRECTORY PRIVATE_KEY [--windows]' >&2; exit 1; fi
if [ "${3:-}" != '' ] && [ "$3" != --windows ]; then exit 1; fi
key=$(cd "$(dirname "$2")" && pwd)/$(basename "$2")
cd "$1"
for platform in darwin linux; do
  for arch in amd64 arm64; do
    test -f "mcpdeck-$platform-$arch.tar.gz"
    test ! -L "mcpdeck-$platform-$arch.tar.gz"
  done
done
shasum -a 256 mcpdeck-darwin-amd64.tar.gz mcpdeck-darwin-arm64.tar.gz \
  mcpdeck-linux-amd64.tar.gz mcpdeck-linux-arm64.tar.gz > SHA256SUMS
if [ "${3:-}" = --windows ]; then
  for arch in amd64 arm64; do
    test -f "mcpdeck-windows-$arch.zip"
    test ! -L "mcpdeck-windows-$arch.zip"
  done
  shasum -a 256 mcpdeck-windows-amd64.zip mcpdeck-windows-arm64.zip >> SHA256SUMS
fi
rm -f SHA256SUMS.sig
ssh-keygen -Y sign -f "$key" -n mcpdeck-release SHA256SUMS
