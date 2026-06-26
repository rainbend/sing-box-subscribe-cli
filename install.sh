#!/usr/bin/env sh
set -eu

repo="rainbend/sing-box-subscribe-cli"
release_binary="sing-box-sub"
binary_name="${BINARY_NAME:-sing-box-sub}"
install_dir="${INSTALL_DIR:-/usr/local/bin}"
version="${VERSION:-}"

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "error: $1 is required" >&2
    exit 1
  fi
}

need_cmd curl
need_cmd head
need_cmd install
need_cmd uname
need_cmd mktemp
need_cmd sed

os="$(uname -s)"
if [ "$os" != "Linux" ]; then
  echo "error: this installer only supports Linux" >&2
  exit 1
fi

case "$(uname -m)" in
  x86_64 | amd64)
    arch="amd64"
    ;;
  aarch64 | arm64)
    arch="arm64"
    ;;
  *)
    echo "error: unsupported architecture: $(uname -m)" >&2
    exit 1
    ;;
esac

if [ -z "$version" ]; then
  version="$(curl -fsSL "https://api.github.com/repos/${repo}/releases/latest" |
    sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' |
    head -n 1)"
fi

if [ -z "$version" ]; then
  echo "error: could not determine latest release version" >&2
  exit 1
fi

case "$version" in
  v*) ;;
  *) version="v${version}" ;;
esac

asset="${release_binary}_${version}_linux_${arch}"
url="https://github.com/${repo}/releases/download/${version}/${asset}"
tmp_dir="$(mktemp -d)"
tmp_file="${tmp_dir}/${asset}"

cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT INT TERM

echo "Downloading ${asset}..."
curl -fL "$url" -o "$tmp_file"
chmod 0755 "$tmp_file"

if [ ! -d "$install_dir" ]; then
  if ! mkdir -p "$install_dir" 2>/dev/null; then
    need_cmd sudo
    sudo mkdir -p "$install_dir"
  fi
fi

if [ -w "$install_dir" ]; then
  install -m 0755 "$tmp_file" "${install_dir}/${binary_name}"
else
  need_cmd sudo
  sudo install -m 0755 "$tmp_file" "${install_dir}/${binary_name}"
fi

echo "Installed ${binary_name} to ${install_dir}/${binary_name}"
"${install_dir}/${binary_name}" version
