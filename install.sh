#!/usr/bin/env bash
set -euo pipefail

# install.sh: Cross-platform installer for tf-blast
# Usage:
#   curl -sSL https://raw.githubusercontent.com/smford/tf-blast/main/install.sh | bash
#   curl -sSL https://raw.githubusercontent.com/smford/tf-blast/main/install.sh | VERSION=v1.0.0 bash

REPO="smford/tf-blast"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
  linux*)  OS="linux" ;;
  darwin*) OS="darwin" ;;
  mingw*|msys*|cygwin*) OS="windows" ;;
  *)
    echo "Error: Unsupported operating system: $OS" >&2
    exit 1
    ;;
esac

# Detect Architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)
    echo "Error: Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

# Determine Version
if [ -z "${VERSION:-}" ]; then
  echo "==> Fetching latest release tag for ${REPO}..."
  LATEST_RELEASE_JSON=$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null || true)
  VERSION=$(echo "$LATEST_RELEASE_JSON" | grep '"tag_name":' | head -1 | cut -d '"' -f 4 || true)
  if [ -z "$VERSION" ]; then
    VERSION="v1.0.0"
    echo "    (Could not query GitHub API; defaulting to ${VERSION})"
  else
    echo "    Latest release is ${VERSION}"
  fi
fi

VERSION_NO_V="${VERSION#v}"
TARBALL="tf-blast_${VERSION_NO_V}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${TARBALL}"

echo "==> Downloading tf-blast ${VERSION} for ${OS}/${ARCH}..."
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

if curl -sSL -f "$DOWNLOAD_URL" -o "${TMP_DIR}/${TARBALL}"; then
  echo "    Extracting binary..."
  tar -xzf "${TMP_DIR}/${TARBALL}" -C "$TMP_DIR"
elif command -v go >/dev/null 2>&1; then
  echo "    Release binary not found. Falling back to local 'go install'..."
  go install "github.com/${REPO}/cmd/tf-blast@latest"
  echo "==> Successfully installed tf-blast via 'go install'!"
  exit 0
else
  echo "Error: Failed to download release asset from ${DOWNLOAD_URL}" >&2
  exit 1
fi

# Locate the extracted binary
if [ -f "${TMP_DIR}/tf-blast" ]; then
  BIN_PATH="${TMP_DIR}/tf-blast"
elif [ -f "${TMP_DIR}/tf-blast.exe" ]; then
  BIN_PATH="${TMP_DIR}/tf-blast.exe"
else
  BIN_PATH=$(find "$TMP_DIR" -name "tf-blast*" -type f -perm +111 2>/dev/null | head -1 || true)
fi

if [ -z "$BIN_PATH" ] || [ ! -f "$BIN_PATH" ]; then
  echo "Error: Extracted binary not found in release archive." >&2
  exit 1
fi

# Ensure destination directory is writable, fallback to ~/.local/bin
if [ ! -d "$INSTALL_DIR" ] || [ ! -w "$INSTALL_DIR" ]; then
  if command -v sudo >/dev/null 2>&1 && [ "$EUID" -ne 0 ]; then
    echo "==> Installing to ${INSTALL_DIR}/tf-blast (using sudo)..."
    sudo install -m 755 "$BIN_PATH" "${INSTALL_DIR}/tf-blast"
  else
    INSTALL_DIR="${HOME}/.local/bin"
    echo "==> ${INSTALL_DIR} selected for user installation..."
    mkdir -p "$INSTALL_DIR"
    install -m 755 "$BIN_PATH" "${INSTALL_DIR}/tf-blast"
    if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
      echo "Notice: Add '${INSTALL_DIR}' to your PATH to run tf-blast directly."
    fi
  fi
else
  echo "==> Installing to ${INSTALL_DIR}/tf-blast..."
  install -m 755 "$BIN_PATH" "${INSTALL_DIR}/tf-blast"
fi

echo "==> Verification:"
"${INSTALL_DIR}/tf-blast" --version || true
echo "==> Successfully installed tf-blast!"
