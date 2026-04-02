#!/bin/bash
set -euo pipefail

# CloudMock installer — detects platform and installs the latest release.
# Usage: curl -fsSL https://cloudmock.io/install.sh | bash

REPO="Viridian-Inc/cloudmock"
INSTALL_DIR="/usr/local/bin"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

case "$OS" in
  darwin|linux) ;;
  *)
    echo "Unsupported OS: $OS"
    exit 1
    ;;
esac

VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
BINARY="cloudmock-${OS}-${ARCH}"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY}"

echo "Installing CloudMock ${VERSION} (${OS}/${ARCH})..."
echo "  ${URL}"

TMP=$(mktemp)
curl -fsSL -o "$TMP" "$URL"
chmod +x "$TMP"

if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP" "${INSTALL_DIR}/cloudmock"
else
  sudo mv "$TMP" "${INSTALL_DIR}/cloudmock"
fi

echo "Installed cloudmock to ${INSTALL_DIR}/cloudmock"
cloudmock --version 2>/dev/null || echo "Done."
