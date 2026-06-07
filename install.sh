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
url="https://github.com/$REPO/releases/download/$tag/${BIN}_${ver}_${os}_${arch}.tar.gz"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
echo "tokenburning: downloading $url"
curl -fsSL "$url" | tar -xz -C "$tmp"

dest="${TOKENBURNING_BIN_DIR:-$HOME/.local/bin}"
mkdir -p "$dest"
install -m 0755 "$tmp/$BIN" "$dest/$BIN"

echo "tokenburning: installed to $dest/$BIN ($tag)"
case ":$PATH:" in
  *":$dest:"*) ;;
  *) echo "tokenburning: add it to your PATH:  export PATH=\"$dest:\$PATH\"" ;;
esac
echo "tokenburning: run  $BIN scan"
