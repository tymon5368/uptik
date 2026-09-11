#!/usr/bin/env bash
# UpTik - Unified One-Line Installer for Linux and macOS
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/tymon5368/uptik/main/scripts/install.sh | bash

set -e

REPO="tymon5368/uptik"
GITHUB_API="https://api.github.com/repos/${REPO}/releases/latest"
BOLD="$(tput bold 2>/dev/null || echo '')"
GREEN="$(tput setaf 2 2>/dev/null || echo '')"
CYAN="$(tput setaf 6 2>/dev/null || echo '')"
YELLOW="$(tput setaf 3 2>/dev/null || echo '')"
RED="$(tput setaf 1 2>/dev/null || echo '')"
RESET="$(tput sgr0 2>/dev/null || echo '')"

info() {
  printf "${CYAN}${BOLD}[UpTik]${RESET} %s\n" "$1"
}

success() {
  printf "${GREEN}${BOLD}[UpTik]${RESET} %s\n" "$1"
}

warn() {
  printf "${YELLOW}${BOLD}[UpTik] Warning:${RESET} %s\n" "$1"
}

error() {
  printf "${RED}${BOLD}[UpTik] Error:${RESET} %s\n" "$1" >&2
  exit 1
}

# 1. Check tools
command -v curl >/dev/null 2>&1 || error "curl is required to run this installer. Please install curl first."

# 2. Detect OS and Architecture
OS="$(uname -s)"
ARCH="$(uname -m)"

info "Detecting platform: OS=${OS}, Arch=${ARCH}"

case "${OS}" in
  Darwin)
    TARGET_OS="macos"
    ;;
  Linux)
    TARGET_OS="linux"
    ;;
  *)
    error "Unsupported operating system: ${OS}. UpTik currently supports Linux, macOS, and Windows."
    ;;
esac

# 3. Fetch latest release from GitHub API
info "Fetching latest release information from GitHub (${REPO})..."
RELEASE_JSON="$(curl -sSL -H "Accept: application/vnd.github.v3+json" "${GITHUB_API}")"

TAG_NAME="$(printf '%s' "${RELEASE_JSON}" | grep '"tag_name":' | head -n1 | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')"
if [ -z "${TAG_NAME}" ]; then
  error "Could not determine latest release tag from GitHub. Check your internet connection."
fi

VERSION="${TAG_NAME#v}"
info "Found latest release: ${BOLD}${TAG_NAME}${RESET}"

# 4. Create temporary working directory
TMP_DIR="$(mktemp -d 2>/dev/null || mktemp -d -t 'uptik-install')"
cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT INT TERM

# 5. Platform-specific download & installation
if [ "${TARGET_OS}" = "macos" ]; then
  PKG_NAME="uptik-${VERSION}-macos-universal.zip"
  DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${TAG_NAME}/${PKG_NAME}"
  
  info "Downloading macOS Universal package (${PKG_NAME})..."
  curl -fSL "${DOWNLOAD_URL}" -o "${TMP_DIR}/${PKG_NAME}" || error "Failed to download macOS package from ${DOWNLOAD_URL}"
  
  info "Extracting ${PKG_NAME}..."
  command -v unzip >/dev/null 2>&1 || error "unzip is required to extract package."
  unzip -q "${TMP_DIR}/${PKG_NAME}" -d "${TMP_DIR}/extracted"
  
  APP_BUNDLE=""
  if [ -d "${TMP_DIR}/extracted/UpTik.app" ]; then
    APP_BUNDLE="${TMP_DIR}/extracted/UpTik.app"
  elif [ -d "${TMP_DIR}/extracted/uptik.app" ]; then
    APP_BUNDLE="${TMP_DIR}/extracted/uptik.app"
  else
    APP_BUNDLE="$(find "${TMP_DIR}/extracted" -name "*.app" -type d -maxdepth 2 | head -n1)"
  fi
  
  if [ -z "${APP_BUNDLE}" ] || [ ! -d "${APP_BUNDLE}" ]; then
    error "Failed to locate .app bundle inside archive."
  fi
  
  DEST_DIR="/Applications"
  if [ ! -w "${DEST_DIR}" ]; then
    DEST_DIR="${HOME}/Applications"
    mkdir -p "${DEST_DIR}"
  fi
  
  info "Installing to ${DEST_DIR}/UpTik.app..."
  rm -rf "${DEST_DIR}/UpTik.app"
  cp -R "${APP_BUNDLE}" "${DEST_DIR}/UpTik.app"
  
  # Remove macOS quarantine bit if present
  xattr -cr "${DEST_DIR}/UpTik.app" 2>/dev/null || true
  
  success "UpTik ${TAG_NAME} successfully installed to ${DEST_DIR}/UpTik.app!"
  info "You can now launch UpTik from Launchpad, Spotlight, or by running:"
  info "  open \"${DEST_DIR}/UpTik.app\""

elif [ "${TARGET_OS}" = "linux" ]; then
  if [ "${ARCH}" != "x86_64" ] && [ "${ARCH}" != "amd64" ]; then
    error "Linux ${ARCH} is not yet supported. Prebuilt binaries are currently available for x86_64/amd64."
  fi

  PKG_NAME="uptik-${VERSION}-linux-amd64.tar.gz"
  DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${TAG_NAME}/${PKG_NAME}"
  
  info "Downloading Linux 64-bit package (${PKG_NAME})..."
  curl -fSL "${DOWNLOAD_URL}" -o "${TMP_DIR}/${PKG_NAME}" || error "Failed to download Linux package from ${DOWNLOAD_URL}"
  
  info "Extracting ${PKG_NAME}..."
  tar -xzf "${TMP_DIR}/${PKG_NAME}" -C "${TMP_DIR}"
  
  EXTRACT_DIR="${TMP_DIR}/uptik-${VERSION}-linux-amd64"
  if [ ! -d "${EXTRACT_DIR}" ]; then
    EXTRACT_DIR="$(find "${TMP_DIR}" -name "uptik" -type f -exec dirname {} \; | head -n1)"
  fi
  
  BIN_DIR="${HOME}/.local/bin"
  APPS_DIR="${HOME}/.local/share/applications"
  ICONS_DIR="${HOME}/.local/share/pixmaps"
  
  mkdir -p "${BIN_DIR}" "${APPS_DIR}" "${ICONS_DIR}"
  
  info "Installing binary to ${BIN_DIR}/uptik..."
  cp "${EXTRACT_DIR}/uptik" "${BIN_DIR}/uptik"
  chmod 755 "${BIN_DIR}/uptik"
  
  if [ -f "${EXTRACT_DIR}/uptik.png" ]; then
    cp "${EXTRACT_DIR}/uptik.png" "${ICONS_DIR}/uptik.png"
  fi
  
  if [ -f "${EXTRACT_DIR}/uptik.desktop" ]; then
    sed "s|Exec=.*|Exec=${BIN_DIR}/uptik|" "${EXTRACT_DIR}/uptik.desktop" > "${APPS_DIR}/uptik.desktop"
    chmod 644 "${APPS_DIR}/uptik.desktop"
    update-desktop-database "${APPS_DIR}" 2>/dev/null || true
  fi
  
  success "UpTik ${TAG_NAME} successfully installed to ${BIN_DIR}/uptik!"
  
  # Check if ~/.local/bin is in PATH
  case ":${PATH}:" in
    *:"${BIN_DIR}":*)
      ;;
    *)
      warn "${BIN_DIR} is not in your current PATH."
      warn "Add it by adding this line to your ~/.bashrc or ~/.zshrc:"
      warn "  export PATH=\"\$HOME/.local/bin:\$PATH\""
      ;;
  esac
  
  info "You can now launch UpTik from your desktop app menu or by typing: uptik"
fi
