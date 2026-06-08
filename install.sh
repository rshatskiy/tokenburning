#!/bin/sh
# tokenburning installer — downloads the latest release binary for your OS/arch.
set -e

REPO="rshatskiy/tokenburning"
BIN="tokenburning"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in
  x86_64|amd64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) echo "tokenburning: unsupported architecture: $arch" >&2; exit 1 ;;
esac
case "$os" in
  darwin|linux) ;;
  *) echo "tokenburning: unsupported OS: $os (on Windows use install.ps1)" >&2; exit 1 ;;
esac

tag=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
  | grep '"tag_name"' | head -1 | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
if [ -z "$tag" ]; then
  echo "tokenburning: could not find the latest release" >&2; exit 1
fi
ver=${tag#v}
fname="${BIN}_${ver}_${os}_${arch}.tar.gz"
base="https://github.com/$REPO/releases/download/$tag"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
echo "tokenburning: downloading $fname ($tag)"
curl -fsSL "$base/$fname" -o "$tmp/$fname"
curl -fsSL "$base/checksums.txt" -o "$tmp/checksums.txt"

# verify SHA-256 against the signed checksums.txt before installing
want=$(grep " ${fname}\$" "$tmp/checksums.txt" | awk '{print $1}')
if [ -z "$want" ]; then
  echo "tokenburning: no checksum for $fname in checksums.txt — refusing to install" >&2; exit 1
fi
if command -v sha256sum >/dev/null 2>&1; then
  got=$(sha256sum "$tmp/$fname" | awk '{print $1}')
elif command -v shasum >/dev/null 2>&1; then
  got=$(shasum -a 256 "$tmp/$fname" | awk '{print $1}')
else
  echo "tokenburning: no sha256 tool (sha256sum/shasum) available to verify download" >&2; exit 1
fi
if [ "$want" != "$got" ]; then
  echo "tokenburning: CHECKSUM MISMATCH — refusing to install" >&2
  echo "  expected: $want" >&2
  echo "  got:      $got" >&2
  exit 1
fi

tar -xzf "$tmp/$fname" -C "$tmp"

dest="${TOKENBURNING_BIN_DIR:-$HOME/.local/bin}"
mkdir -p "$dest"
install -m 0755 "$tmp/$BIN" "$dest/$BIN"

echo "tokenburning: installed to $dest/$BIN ($tag, checksum verified)"
case ":$PATH:" in
  *":$dest:"*) ;;
  *) echo "tokenburning: add it to your PATH:  export PATH=\"$dest:\$PATH\"" ;;
esac
echo ""
echo "Installed. Run:"
echo "    $BIN scan"
