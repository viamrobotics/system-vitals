#!/usr/bin/env bash
# Build-environment setup for Viam cloud build (meta.json build.setup) and `viam module build local`.
#
# The Go toolchain is already present on Viam's builders. This script adds what the custom
# control UI needs: Node 22 and pnpm. On Linux the builder is a Debian layer without Node, so the
# official Node tarball is installed under /usr/local. On macOS, Homebrew provides node and pnpm
# through build.darwin_deps in meta.json, so this script only fills in pnpm if it is missing.
set -euo pipefail

NODE_VERSION="${NODE_VERSION:-22.23.1}"
PNPM_VERSION="${PNPM_VERSION:-10}"
MIN_NODE_MAJOR=20

os="${VIAM_BUILD_OS:-$(uname -s | tr '[:upper:]' '[:lower:]')}"
arch="${VIAM_BUILD_ARCH:-$(uname -m)}"

# Prefix privileged commands with sudo only when we are not root and sudo exists.
as_root() {
  if [ "$(id -u)" -eq 0 ]; then "$@"; elif command -v sudo >/dev/null 2>&1; then sudo "$@"; else "$@"; fi
}

node_ok() {
  command -v node >/dev/null 2>&1 || return 1
  local major
  major="$(node -p 'process.versions.node.split(".")[0]')"
  [ "$major" -ge "$MIN_NODE_MAJOR" ]
}

install_node_linux() {
  local node_arch
  case "$arch" in
    amd64 | x86_64) node_arch=x64 ;;
    arm64 | aarch64) node_arch=arm64 ;;
    *) echo "setup.sh: unsupported linux arch '$arch'" >&2; exit 1 ;;
  esac
  if ! command -v curl >/dev/null 2>&1; then
    as_root apt-get update
    as_root apt-get install -y --no-install-recommends curl ca-certificates
  fi
  local url="https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-linux-${node_arch}.tar.gz"
  echo "setup.sh: installing Node ${NODE_VERSION} (${node_arch}) from ${url}"
  curl -fsSL "$url" | as_root tar -xz -C /usr/local --strip-components=1 --no-same-owner
}

case "$os" in
  linux)
    if node_ok; then
      echo "setup.sh: node $(node --version) already present"
    else
      install_node_linux
    fi
    ;;
  darwin)
    if ! node_ok; then
      if command -v brew >/dev/null 2>&1; then
        brew install node
      else
        echo "setup.sh: Node >= ${MIN_NODE_MAJOR} is required; install it from https://nodejs.org or Homebrew" >&2
        exit 1
      fi
    fi
    ;;
  *)
    echo "setup.sh: unsupported OS '$os'" >&2
    exit 1
    ;;
esac

if ! command -v pnpm >/dev/null 2>&1; then
  echo "setup.sh: installing pnpm ${PNPM_VERSION}"
  as_root npm install -g "pnpm@${PNPM_VERSION}"
fi

echo "setup.sh: node $(node --version), pnpm $(pnpm --version)"
go mod tidy
