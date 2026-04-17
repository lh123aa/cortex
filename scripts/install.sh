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

# 占位直接编译方式 (因当前尚无 release 制品)
compile_from_source() {
    log_info "Compiling Cortex from source..."
    if ! command -v go &> /dev/null; then
        log_error "Go 1.21+ is required for source compilation."
        exit 1
    fi
    
    go mod tidy
    go build -o $BINARY_NAME cmd/cortex/main.go
}

install() {
    log_info "Installing Cortex..."
    mkdir -p ${CONFIG_DIR}
    sudo mv ${BINARY_NAME} ${INSTALL_DIR}/${BINARY_NAME}
    log_info "Cortex installed to ${INSTALL_DIR}/${BINARY_NAME}"
}

check_ollama() {
    if ! command -v ollama &> /dev/null; then
        log_warn "Ollama not found. Please install it from https://ollama.ai for local embeddings."
    else
        log_info "Ollama found!"
    fi
}

verify() {
    if command -v cortex &> /dev/null; then
        log_info "Cortex installed successfully!"
        log_info "Run 'cortex index ./docs' to start indexing."
    else
        log_error "Installation verification failed"
        exit 1
    fi
}

main() {
    log_info "Starting Cortex installation..."
    detect_os
    compile_from_source
    install
    check_ollama
    verify
}

main "$@"
