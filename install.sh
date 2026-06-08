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
# Зеркало на своём домене (GitHub release-CDN нестабилен в РФ); GitHub — фолбэк.
mirror="https://tokenburning.ru/dl/$tag"
gh="https://github.com/$REPO/releases/download/$tag"
fetch() { # $1=filename $2=outfile
  curl -fsSL "$mirror/$1" -o "$2" 2>/dev/null || curl -fsSL "$gh/$1" -o "$2"
}

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
echo "tokenburning: downloading $fname ($tag)"
fetch "$fname" "$tmp/$fname"
fetch "checksums.txt" "$tmp/checksums.txt"

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
inpath=0
case ":$PATH:" in *":$dest:"*) inpath=1 ;; esac
echo ""
echo "Installed ($tag)."
echo ""
echo "See your AI spend (local, nothing leaves your machine):"
echo "    tokenburning dashboard     # visual dashboard in your browser"
echo "    tokenburning scan          # quick numbers in the terminal"
echo ""
echo "Send your stats to a team dashboard (optional):"
echo "    1) open  https://tokenburning.ru/install   and click \"Generate token\""
echo "    2) run:  tokenburning connect --to https://tokenburning.ru --token <YOUR-TOKEN> --breadth"
if [ "$inpath" -eq 0 ]; then
  echo ""
  echo "Note: add it to your PATH for the 'tokenburning' command:"
  echo "    export PATH=\"$dest:\$PATH\""
fi
# Авто-открытие дашборда, если установка идёт в интерактивном терминале
if [ -t 1 ] && [ -z "$TOKENBURNING_NO_LAUNCH" ]; then
  echo ""
  echo "Opening your dashboard…"
  ( nohup "$dest/$BIN" dashboard >/dev/null 2>&1 & ) 2>/dev/null || true
fi
