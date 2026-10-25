#!/usr/bin/env bash
# ==============================================================================
# Delvo Standalone Installer & Uninstaller
# Safe, cross-platform installer for Delvo (dvo)
# Supports: Android (Termux arm64), Linux (amd64/arm64), macOS (amd64/arm64)
# ==============================================================================

set -euo pipefail

REPO="AgentNex/delvo"
DEFAULT_VERSION="v0.1.0"
BINARY_NAME="dvo"

# Colors for terminal output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
NC='\033[0m' # No Color

info() {
  printf "${BLUE}[INFO]${NC} %s\n" "$1"
}

success() {
  printf "${GREEN}[SUCCESS]${NC} %s\n" "$1"
}

warn() {
  printf "${YELLOW}[WARN]${NC} %s\n" "$1"
}

error() {
  printf "${RED}[ERROR]${NC} %s\n" "$1" >&2
}

fatal() {
  error "$1"
  exit 1
}

print_help() {
  cat <<EOF
Delvo Installer

Usage:
  curl -fsSL https://raw.githubusercontent.com/${REPO}/main/install.sh | bash
  ./install.sh [OPTIONS]

Options:
  -v, --version <version>    Install a specific version (e.g. v0.1.0 or 0.1.0)
  -d, --install-dir <path>   Specify custom installation directory
  -u, --uninstall            Uninstall Delvo and remove binary
  -h, --help                 Show this help message

Environment Variables:
  VERSION                    Target version (default: latest or ${DEFAULT_VERSION})
  INSTALL_DIR                Target directory for binary
  UNINSTALL                  Set to 1 or true to trigger uninstallation
EOF
}

# Parse command-line flags
TARGET_VERSION="${VERSION:-}"
CUSTOM_DIR="${INSTALL_DIR:-}"
DO_UNINSTALL="${UNINSTALL:-false}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    -v|--version)
      TARGET_VERSION="$2"
      shift 2
      ;;
    -d|--install-dir)
      CUSTOM_DIR="$2"
      shift 2
      ;;
    -u|--uninstall)
      DO_UNINSTALL="true"
      shift
      ;;
    -h|--help)
      print_help
      exit 0
      ;;
    *)
      warn "Unknown option: $1"
      shift
      ;;
  esac
done

# Detect OS
detect_os() {
  if [ -n "${PREFIX:-}" ] && [[ "${PREFIX}" == *termux* ]] || [ -d "/data/data/com.termux" ]; then
    echo "android"
    return
  fi

  local os_type
  os_type="$(uname -s 2>/dev/null || true)"
  case "${os_type}" in
    Linux)  echo "linux" ;;
    Darwin) echo "darwin" ;;
    *)      echo "unsupported" ;;
  esac
}

# Detect Architecture
detect_arch() {
  local arch_type
  arch_type="$(uname -m 2>/dev/null || true)"
  case "${arch_type}" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *) echo "unsupported" ;;
  esac
}

# Determine default installation directory
resolve_install_dir() {
  if [ -n "${CUSTOM_DIR}" ]; then
    echo "${CUSTOM_DIR}"
    return
  fi

  # If running in Termux, $PREFIX/bin is standard, user-writable, and in PATH
  if [ -n "${PREFIX:-}" ] && [ -d "${PREFIX}/bin" ]; then
    echo "${PREFIX}/bin"
    return
  fi

  # If root or /usr/local/bin is writable
  if [ -w "/usr/local/bin" ]; then
    echo "/usr/local/bin"
    return
  fi

  # Default to user local bin (~/.local/bin)
  echo "${HOME}/.local/bin"
}

# Perform Uninstallation
do_uninstall() {
  local target_dir
  target_dir="$(resolve_install_dir)"
  local bin_path="${target_dir}/${BINARY_NAME}"
  local removed=0

  info "Searching for Delvo installations to remove..."

  if [ -f "${bin_path}" ]; then
    info "Removing ${bin_path}..."
    rm -f "${bin_path}"
    removed=1
  fi

  # Also check if another binary is in PATH
  local path_bin
  path_bin="$(command -v "${BINARY_NAME}" 2>/dev/null || true)"
  if [ -n "${path_bin}" ] && [ -f "${path_bin}" ] && [ "${path_bin}" != "${bin_path}" ]; then
    info "Found additional binary at ${path_bin}..."
    if [ -w "$(dirname "${path_bin}")" ]; then
      rm -f "${path_bin}"
      removed=1
    else
      warn "Cannot remove ${path_bin} without elevated permissions. Please run: rm -f ${path_bin}"
    fi
  fi

  if [ "${removed}" -eq 1 ]; then
    success "Delvo (${BINARY_NAME}) uninstalled successfully."
  else
    info "No Delvo installation found in ${target_dir} or PATH."
  fi
  exit 0
}

if [ "${DO_UNINSTALL}" = "true" ] || [ "${DO_UNINSTALL}" = "1" ]; then
  do_uninstall
fi

# Detect platform
OS="$(detect_os)"
ARCH="$(detect_arch)"

info "Detected platform: OS='${OS}', Arch='${ARCH}'"

# Validate supported combinations
case "${OS}-${ARCH}" in
  android-arm64|linux-amd64|linux-arm64|darwin-amd64|darwin-arm64)
    ;;
  *)
    error "Platform '${OS}-${ARCH}' is not officially supported by precompiled binaries."
    echo ""
    echo "Fallback: You can build Delvo directly from source with Go (v1.22+):"
    echo "  go install github.com/${REPO}/cmd/dvo@latest"
    echo ""
    exit 1
    ;;
esac

# Resolve Version
if [ -z "${TARGET_VERSION}" ] || [ "${TARGET_VERSION}" = "latest" ]; then
  info "Fetching latest release information from GitHub..."
  LATEST_TAG=""
  if command -v curl >/dev/null 2>&1; then
    LATEST_TAG="$(curl -sSL -H "Accept: application/vnd.github.v3+json" "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep -o '"tag_name": *"[^"]*"' | head -n 1 | cut -d '"' -f 4 || true)"
  fi
  if [ -z "${LATEST_TAG}" ]; then
    info "Could not query GitHub API (rate-limited or offline), defaulting to ${DEFAULT_VERSION}"
    TARGET_VERSION="${DEFAULT_VERSION}"
  else
    TARGET_VERSION="${LATEST_TAG}"
  fi
fi

# Normalize tag name (vX.Y.Z)
if [[ "${TARGET_VERSION}" != v* ]]; then
  TAG="v${TARGET_VERSION}"
else
  TAG="${TARGET_VERSION}"
fi

ARCHIVE_NAME="delvo-${OS}-${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${TAG}/${ARCHIVE_NAME}"
CHECKSUMS_URL="https://github.com/${REPO}/releases/download/${TAG}/checksums.txt"

INSTALL_DIR="$(resolve_install_dir)"
mkdir -p "${INSTALL_DIR}"

info "Installing Delvo ${TAG} into ${INSTALL_DIR}..."

# Create secure temporary directory
TEMP_DIR="$(mktemp -d 2>/dev/null || mktemp -d -t 'dvo_install_XXXXXX')"
cleanup() {
  rm -rf "${TEMP_DIR}"
}
trap cleanup EXIT

# Download archive
info "Downloading ${DOWNLOAD_URL}..."
if command -v curl >/dev/null 2>&1; then
  curl -fsSL "${DOWNLOAD_URL}" -o "${TEMP_DIR}/${ARCHIVE_NAME}"
elif command -v wget >/dev/null 2>&1; then
  wget -q "${DOWNLOAD_URL}" -O "${TEMP_DIR}/${ARCHIVE_NAME}"
else
  fatal "Neither curl nor wget is available. Please install curl or wget."
fi

# Download checksums
info "Downloading checksums from ${CHECKSUMS_URL}..."
if command -v curl >/dev/null 2>&1; then
  curl -fsSL "${CHECKSUMS_URL}" -o "${TEMP_DIR}/checksums.txt" || warn "Could not fetch checksums.txt"
elif command -v wget >/dev/null 2>&1; then
  wget -q "${CHECKSUMS_URL}" -O "${TEMP_DIR}/checksums.txt" || warn "Could not fetch checksums.txt"
fi

# Verify SHA-256 Checksum
if [ -f "${TEMP_DIR}/checksums.txt" ]; then
  info "Verifying SHA-256 checksum..."
  EXPECTED_SHA="$(grep "${ARCHIVE_NAME}" "${TEMP_DIR}/checksums.txt" | awk '{print $1}' || true)"
  if [ -n "${EXPECTED_SHA}" ]; then
    ACTUAL_SHA=""
    if command -v sha256sum >/dev/null 2>&1; then
      ACTUAL_SHA="$(sha256sum "${TEMP_DIR}/${ARCHIVE_NAME}" | awk '{print $1}')"
    elif command -v shasum >/dev/null 2>&1; then
      ACTUAL_SHA="$(shasum -a 256 "${TEMP_DIR}/${ARCHIVE_NAME}" | awk '{print $1}')"
    elif command -v openssl >/dev/null 2>&1; then
      ACTUAL_SHA="$(openssl dgst -sha256 "${TEMP_DIR}/${ARCHIVE_NAME}" | awk '{print $NF}')"
    fi

    if [ -n "${ACTUAL_SHA}" ]; then
      if [ "${ACTUAL_SHA}" != "${EXPECTED_SHA}" ]; then
        fatal "Checksum verification failed!\nExpected: ${EXPECTED_SHA}\nActual:   ${ACTUAL_SHA}\nAborting installation due to possible tampering or corrupted download."
      fi
      info "Checksum verified successfully (${ACTUAL_SHA})."
    else
      warn "No SHA-256 utility found. Skipping checksum verification."
    fi
  else
    warn "Archive ${ARCHIVE_NAME} not found in checksums.txt. Skipping checksum verification."
  fi
fi

# Extract archive
info "Extracting binary..."
tar -xzf "${TEMP_DIR}/${ARCHIVE_NAME}" -C "${TEMP_DIR}"

if [ ! -f "${TEMP_DIR}/${BINARY_NAME}" ]; then
  fatal "Binary '${BINARY_NAME}' not found inside archive."
fi

chmod +x "${TEMP_DIR}/${BINARY_NAME}"
mv -f "${TEMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"

# Verify execution
if [ -x "${INSTALL_DIR}/${BINARY_NAME}" ]; then
  INSTALLED_VERSION="$("${INSTALL_DIR}/${BINARY_NAME}" --version 2>/dev/null || true)"
  success "Successfully installed ${INSTALLED_VERSION} to ${INSTALL_DIR}/${BINARY_NAME}"
else
  fatal "Installed file is not executable at ${INSTALL_DIR}/${BINARY_NAME}"
fi

# PATH check
case ":$PATH:" in
  *":${INSTALL_DIR}:"*)
    info "${INSTALL_DIR} is already in your \$PATH."
    ;;
  *)
    warn "${INSTALL_DIR} is NOT in your \$PATH."
    echo "To run '${BINARY_NAME}' from anywhere, add the following to your ~/.bashrc or ~/.zshrc:"
    echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
    ;;
esac

echo ""
echo "Run '${BINARY_NAME} --help' or simply '${BINARY_NAME}' to start browsing!"
