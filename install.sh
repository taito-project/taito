#!/bin/sh
# taito installer script
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/taito-project/taito/main/install.sh | sh
#
# Environment variables:
#   VERSION      - specific version to install (e.g., "0.34.1"), defaults to latest
#   INSTALL_DIR  - installation directory, defaults to /usr/local/bin

set -e

GITHUB_REPO="taito-project/taito"
BINARY_NAME="taito"

# --- helper functions ---

info() {
    printf '%s\n' "$@"
}

warn() {
    printf '%s\n' "$@" >&2
}

error() {
    printf '%s\n' "$@" >&2
    exit 1
}

# --- detect platform ---

detect_os() {
    os="$(uname -s)"
    case "$os" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        *)       error "Unsupported operating system: $os" ;;
    esac
}

detect_arch() {
    arch="$(uname -m)"
    case "$arch" in
        x86_64|amd64)  echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *)             error "Unsupported architecture: $arch" ;;
    esac
}

# --- download helpers ---

has_cmd() {
    command -v "$1" >/dev/null 2>&1
}

download() {
    url="$1"
    dest="$2"
    if has_cmd curl; then
        curl -fsSL -o "$dest" "$url"
    elif has_cmd wget; then
        wget -qO "$dest" "$url"
    else
        error "Neither curl nor wget found. Please install one of them."
    fi
}

# --- version resolution ---

get_latest_version() {
    if has_cmd curl; then
        curl -fsSL "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" 2>/dev/null
    elif has_cmd wget; then
        wget -qO- "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" 2>/dev/null
    else
        error "Neither curl nor wget found."
    fi | sed -n 's/.*"tag_name": *"v\{0,1\}\([^"]*\)".*/\1/p' | head -1
}

# --- main ---

main() {
    OS="$(detect_os)"
    ARCH="$(detect_arch)"

    info "taito installer"
    info "Detected platform: ${OS}/${ARCH}"

    if [ -z "$VERSION" ]; then
        info "Fetching latest version..."
        VERSION="$(get_latest_version)"
        if [ -z "$VERSION" ]; then
            error "Could not determine latest version. Set VERSION manually."
        fi
    fi

    # Strip leading 'v' if present
    VERSION="${VERSION#v}"

    INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
    ASSET_NAME="${BINARY_NAME}-${OS}-${ARCH}"
    DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/v${VERSION}/${ASSET_NAME}"
    CHECKSUMS_URL="https://github.com/${GITHUB_REPO}/releases/download/v${VERSION}/checksums.txt"

    info "Installing taito v${VERSION} (${OS}/${ARCH})"

    tmpdir="$(mktemp -d)"
    trap 'rm -rf "$tmpdir"' EXIT

    info "Downloading ${ASSET_NAME}..."
    download "$DOWNLOAD_URL" "${tmpdir}/${ASSET_NAME}"

    chmod +x "${tmpdir}/${ASSET_NAME}"

    # Install to target directory
    if [ -w "$INSTALL_DIR" ]; then
        mv "${tmpdir}/${ASSET_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    else
        info "Elevated permissions required to install to ${INSTALL_DIR}"
        if has_cmd sudo; then
            sudo mv "${tmpdir}/${ASSET_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
        elif has_cmd doas; then
            doas mv "${tmpdir}/${ASSET_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
        else
            error "Cannot write to ${INSTALL_DIR}. Run as root or set INSTALL_DIR to a writable path."
        fi
    fi

    info "taito v${VERSION} installed to ${INSTALL_DIR}/${BINARY_NAME}"
    printf '\n'
    printf '  Get started by running:\n'
    printf '\n'
    printf '    \033[1m$ taito setup\033[0m\n'
    printf '\n'
}

main
