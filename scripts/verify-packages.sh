#!/bin/sh
# The allowed signers file must come from an independently trusted checkout.
set -eu
if [ "$#" -ne 2 ]; then echo 'Usage: verify-packages.sh DIRECTORY ALLOWED_SIGNERS' >&2; exit 1; fi
signers=$(cd "$(dirname "$2")" && pwd)/$(basename "$2")
cd "$1"
ssh-keygen -Y verify -f "$signers" -I mcpdeck-release \
  -n mcpdeck-release -s SHA256SUMS.sig < SHA256SUMS
# Reject paths and unexpected/missing/duplicate entries before checking files.
awk '
  NF != 2 || $1 !~ /^[a-f0-9]+$/ || length($1) != 64 { exit 1 }
  $2 !~ /^mcpdeck-(darwin|linux)-(amd64|arm64)\.tar\.gz$/ && $2 !~ /^mcpdeck-windows-(amd64|arm64)\.zip$/ { exit 1 }
  seen[$2]++ { exit 1 }
  END {
    if (NR != 4 && NR != 6) exit 1
    if (!seen["mcpdeck-darwin-amd64.tar.gz"] || !seen["mcpdeck-darwin-arm64.tar.gz"] || !seen["mcpdeck-linux-amd64.tar.gz"] || !seen["mcpdeck-linux-arm64.tar.gz"]) exit 1
    if (NR == 6 && (!seen["mcpdeck-windows-amd64.zip"] || !seen["mcpdeck-windows-arm64.zip"])) exit 1
  }
' SHA256SUMS
for platform in darwin linux; do
  for arch in amd64 arm64; do
    test -f "mcpdeck-$platform-$arch.tar.gz"
    test ! -L "mcpdeck-$platform-$arch.tar.gz"
  done
done
if [ "$(wc -l < SHA256SUMS | tr -d ' ')" = 6 ]; then
  for arch in amd64 arm64; do
    test -f "mcpdeck-windows-$arch.zip"
    test ! -L "mcpdeck-windows-$arch.zip"
  done
fi
shasum -a 256 -c SHA256SUMS
