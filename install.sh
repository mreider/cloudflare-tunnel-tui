#!/bin/sh
set -e

REPO="mreider/cloudflare-tunnel-tui"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *)       echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
  darwin|linux) ;;
  *)            echo "Unsupported OS: $OS"; exit 1 ;;
esac

SUFFIX="${OS}-${ARCH}"

echo "Detected: ${OS}/${ARCH}"

# Get latest release tag
TAG=$(curl -sI "https://github.com/${REPO}/releases/latest" | grep -i '^location:' | sed 's/.*tag\///' | tr -d '\r\n')
if [ -z "$TAG" ]; then
  echo "Failed to find latest release"
  exit 1
fi

echo "Latest release: ${TAG}"

BASE_URL="https://github.com/${REPO}/releases/download/${TAG}"

for BINARY in tunneltui mkbundle; do
  URL="${BASE_URL}/${BINARY}-${SUFFIX}"
  echo "Downloading ${BINARY}..."
  curl -sL "$URL" -o "/tmp/${BINARY}"
  chmod +x "/tmp/${BINARY}"

  if [ -w "$INSTALL_DIR" ]; then
    mv "/tmp/${BINARY}" "${INSTALL_DIR}/${BINARY}"
  else
    sudo mv "/tmp/${BINARY}" "${INSTALL_DIR}/${BINARY}"
  fi

  echo "  Installed ${INSTALL_DIR}/${BINARY}"
done

echo ""
echo "Done. Run 'tunneltui --help' to get started."
