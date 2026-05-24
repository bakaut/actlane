#!/bin/sh
set -eu

repo="${ACTLANE_REPO:-actlane/actlane}"
version="${ACTLANE_VERSION:-latest}"
install_dir="${ACTLANE_INSTALL_DIR:-/usr/local/bin}"
binary_name="${ACTLANE_BINARY_NAME:-actlane}"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"

case "$os" in
  darwin|linux) ;;
  *)
    echo "unsupported OS: $os" >&2
    exit 1
    ;;
esac

case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *)
    echo "unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

asset="actlane-${os}-${arch}"
base_url="https://github.com/${repo}/releases"
if [ "$version" = "latest" ]; then
  url="${base_url}/latest/download/${asset}"
else
  url="${base_url}/download/${version}/${asset}"
fi

tmp="${TMPDIR:-/tmp}/actlane-install.$$"
trap 'rm -f "$tmp"' EXIT INT TERM

echo "Downloading ${asset} from ${repo}..."
curl -fsSL "$url" -o "$tmp"
chmod +x "$tmp"

mkdir -p "$install_dir"
target="${install_dir}/${binary_name}"
if [ -w "$install_dir" ]; then
  mv "$tmp" "$target"
else
  sudo mv "$tmp" "$target"
fi

"$target" version
