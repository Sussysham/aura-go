#!/usr/bin/env bash

set -euo pipefail

# Aura Cross-Platform Stdio Installer (Linux & macOS)

GITHUB_REPO="Sussysham/aura-go"
INSTALL_DIR="/usr/local/bin"
APP_NAME="aura"

echo -e "\033[1;36m📖 AURA NATIVE TUI INSTALLATION ENGINE\033[0m"
echo "--------------------------------------------------------"

# 1. Probe Operating System and Architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "${ARCH}" in
    x86_64)  ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *)
        echo -e "\033[1;31m❌ [ERROR] Unsupported architecture: ${ARCH}\033[0m"
        exit 1
        ;;
esac

case "${OS}" in
    linux|darwin) ;;
    *)
        echo -e "\033[1;31m❌ [ERROR] Unsupported operating system: ${OS}\033[0m"
        exit 1
        ;;
esac

echo -e "✔ [DETECTED] Operating System: \033[1;32m${OS}\033[0m | Architecture: \033[1;32m${ARCH}\033[0m"

# 2. Fetch Latest Release Details from GitHub API
echo "🔍 Querying latest GitHub releases..."
LATEST_TAG=$(curl -s https://api.github.com/repos/${GITHUB_REPO}/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "${LATEST_TAG}" ]; then
    LATEST_TAG="v1.0.0"
fi

echo -e "✔ [FOUND] Latest release tag: \033[1;32m${LATEST_TAG}\033[0m"

# Normalize OS names to standard release names
RELEASE_OS="${OS}"
if [ "${RELEASE_OS}" = "darwin" ]; then
    RELEASE_OS="macOS"
fi

TARBALL="aura_${LATEST_TAG#v}_${RELEASE_OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/${LATEST_TAG}/${TARBALL}"

TEMP_DIR=$(mktemp -d)
defer_cleanup() {
    rm -rf "${TEMP_DIR}"
}
trap defer_cleanup EXIT

echo "📥 Downloading asset tarball: ${TARBALL}..."
if ! curl -sL -o "${TEMP_DIR}/${TARBALL}" "${DOWNLOAD_URL}"; then
    echo -e "\033[1;31m❌ [ERROR] Download failed. Make sure release assets are compiled for this platform.\033[0m"
    exit 1
fi

echo "📦 Extracting release contents..."
tar -xzf "${TEMP_DIR}/${TARBALL}" -C "${TEMP_DIR}"

# 3. Installing binary executable
echo "🚀 Copying compiled binary to ${INSTALL_DIR}/${APP_NAME}..."
if [ -w "${INSTALL_DIR}" ]; then
    cp "${TEMP_DIR}/${APP_NAME}" "${INSTALL_DIR}/${APP_NAME}"
    chmod +x "${INSTALL_DIR}/${APP_NAME}"
else
    echo "🔑 Root permissions required to copy to ${INSTALL_DIR}."
    sudo cp "${TEMP_DIR}/${APP_NAME}" "${INSTALL_DIR}/${APP_NAME}"
    sudo chmod +x "${INSTALL_DIR}/${APP_NAME}"
fi

echo "--------------------------------------------------------"
echo -e "\033[1;32m✔ [SUCCESS] Aura TUI successfully installed to ${INSTALL_DIR}/${APP_NAME}!\033[0m"
echo -e "💡 Get started by simply typing: \033[1;36maura\033[0m"
