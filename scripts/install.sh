#!/bin/bash
set -e

VERSION="v1.0.0"
BINARY_NAME="cortex"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="$HOME/.cortex"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

detect_os() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    case $ARCH in
        x86_64) ARCH="amd64" ;;
        arm64|aarch64) ARCH="arm64" ;;
        *) log_error "Unsupported architecture: $ARCH"; exit 1 ;;
    esac
    log_info "Detected OS: $OS, Architecture: $ARCH"
}

compile_from_source() {
    log_info "Compiling Cortex from source..."
    if ! command -v go &> /dev/null; then
        log_error "Go 1.21+ is required. Install from https://go.dev/dl/"
        exit 1
    fi

    go mod tidy
    go build -ldflags="-s -w" -o $BINARY_NAME ./cmd/cortex
}

install() {
    log_info "Installing Cortex..."
    mkdir -p ${CONFIG_DIR}
    sudo mv ${BINARY_NAME} ${INSTALL_DIR}/${BINARY_NAME}
    log_info "Cortex installed to ${INSTALL_DIR}/${BINARY_NAME}"
}

verify() {
    if command -v cortex &> /dev/null; then
        log_info "✅ Cortex installed!"
        if [ -n "$1" ]; then
            log_info "Running: cortex install $1"
            cortex install "$1"
        else
            log_info "Run 'cortex install <your-docs-dir>' to auto-configure and index."
        fi
    else
        log_error "Installation verification failed"
        exit 1
    fi
}

main() {
    DOC_DIR="${1:-}"

    echo ""
    echo "  ⚡ Installing Cortex..."
    echo ""

    detect_os
    compile_from_source
    install
    verify "$DOC_DIR"

    echo ""
    log_info "🎉  Cortex is ready!"
    echo ""
}

main "$@"
