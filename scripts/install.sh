#!/bin/sh
# Cortex 一键安装脚本（Linux / macOS）
#
# 用法：
#   curl -fsSL https://raw.githubusercontent.com/lh123aa/cortex/main/scripts/install.sh | sh
#   curl -fsSL ... | sh -s -- "/path/to/your/docs"          # 安装后自动索引文档
#   CORTEX_VERSION=v4.0.0 sh scripts/install.sh             # 安装指定版本
#   CORTEX_FROM_SOURCE=1 sh scripts/install.sh              # 从当前仓库源码编译
#
# 环境变量：
#   CORTEX_VERSION      版本 tag（默认 latest，走 GitHub Releases 最新版）
#   INSTALL_DIR         安装目录（默认：可写则 /usr/local/bin，否则 ~/.local/bin）
#   CORTEX_FROM_SOURCE  =1 时跳过下载，直接从源码编译（需要 Go 1.25+）
#
# 产物命名与 .github/workflows/build.yml 严格一致：
#   cortex-<os>-<arch>.tar.gz   (os: linux/darwin, arch: amd64/arm64)
#   SHA256SUMS                   (sha256sum 校验文件)
set -eu

REPO="lh123aa/cortex"
BINARY_NAME="cortex"
CONFIG_DIR="${HOME}/.cortex"
CORTEX_VERSION="${CORTEX_VERSION:-latest}"
CORTEX_FROM_SOURCE="${CORTEX_FROM_SOURCE:-0}"

TMP_DIR=""
cleanup() {
    if [ -n "${TMP_DIR}" ] && [ -d "${TMP_DIR}" ]; then
        rm -rf "${TMP_DIR}"
    fi
}
trap cleanup EXIT INT TERM

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { printf '%b[INFO]%b %s\n' "${GREEN}" "${NC}" "$1"; }
log_warn() { printf '%b[WARN]%b %s\n' "${YELLOW}" "${NC}" "$1"; }
log_error() { printf '%b[ERROR]%b %s\n' "${RED}" "${NC}" "$1" >&2; }

detect_os() {
    OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
    ARCH="$(uname -m)"

    case "${OS}" in
        linux) ;;
        darwin) ;;
        mingw* | msys* | cygwin*)
            log_error "检测到 Windows 环境，请改用 PowerShell 安装脚本："
            log_error "  iwr -useb https://raw.githubusercontent.com/${REPO}/main/scripts/install.ps1 | iex"
            exit 1
            ;;
        *)
            log_error "不支持的操作系统：${OS}（仅支持 Linux / macOS）"
            exit 1
            ;;
    esac

    case "${ARCH}" in
        x86_64 | amd64) ARCH="amd64" ;;
        arm64 | aarch64) ARCH="arm64" ;;
        *)
            log_error "不支持的 CPU 架构：${ARCH}（仅支持 amd64 / arm64）"
            exit 1
            ;;
    esac

    log_info "检测到环境：${OS}/${ARCH}"
}

resolve_urls() {
    ARCHIVE_NAME="cortex-${OS}-${ARCH}.tar.gz"

    if [ "${CORTEX_VERSION}" = "latest" ]; then
        BASE_URL="https://github.com/${REPO}/releases/latest/download"
    else
        # 允许 CORTEX_VERSION=4.0.0 这种省略 v 前缀的写法
        case "${CORTEX_VERSION}" in
            v*) ;;
            *) CORTEX_VERSION="v${CORTEX_VERSION}" ;;
        esac
        BASE_URL="https://github.com/${REPO}/releases/download/${CORTEX_VERSION}"
    fi

    ARCHIVE_URL="${BASE_URL}/${ARCHIVE_NAME}"
    CHECKSUMS_URL="${BASE_URL}/SHA256SUMS"
}

# fetch <url> <输出文件>
fetch() {
    if command -v curl > /dev/null 2>&1; then
        curl -fsSL --retry 3 --connect-timeout 15 -o "$2" "$1"
    elif command -v wget > /dev/null 2>&1; then
        wget -q -O "$2" "$1"
    else
        log_error "需要 curl 或 wget 才能下载安装包"
        return 1
    fi
}

sha256_of() {
    if command -v sha256sum > /dev/null 2>&1; then
        sha256sum "$1" | awk '{print $1}'
    elif command -v shasum > /dev/null 2>&1; then
        shasum -a 256 "$1" | awk '{print $1}'
    else
        return 1
    fi
}

verify_checksum() {
    archive="$1"
    if ! fetch "${CHECKSUMS_URL}" "${TMP_DIR}/SHA256SUMS" 2> /dev/null; then
        log_warn "未获取到 SHA256SUMS，跳过校验"
        return 0
    fi

    expected="$(grep -F " ${ARCHIVE_NAME}" "${TMP_DIR}/SHA256SUMS" | awk '{print $1}' | head -n 1)"
    if [ -z "${expected}" ]; then
        log_warn "SHA256SUMS 中没有 ${ARCHIVE_NAME} 的记录，跳过校验"
        return 0
    fi

    if ! actual="$(sha256_of "${archive}")"; then
        log_warn "系统缺少 sha256sum/shasum，跳过校验"
        return 0
    fi

    if [ "${expected}" != "${actual}" ]; then
        log_error "SHA256 校验失败！"
        log_error "  期望：${expected}"
        log_error "  实际：${actual}"
        exit 1
    fi
    log_info "SHA256 校验通过"
}

install_from_release() {
    log_info "下载 ${ARCHIVE_URL}"
    if ! fetch "${ARCHIVE_URL}" "${TMP_DIR}/${ARCHIVE_NAME}"; then
        log_error "下载失败：${ARCHIVE_URL}"
        return 1
    fi

    verify_checksum "${TMP_DIR}/${ARCHIVE_NAME}"

    mkdir -p "${TMP_DIR}/extract"
    tar -xzf "${TMP_DIR}/${ARCHIVE_NAME}" -C "${TMP_DIR}/extract"

    binary="${TMP_DIR}/extract/${BINARY_NAME}"
    if [ ! -f "${binary}" ]; then
        log_error "安装包内未找到 ${BINARY_NAME} 可执行文件"
        return 1
    fi
    chmod +x "${binary}"
    install_binary "${binary}"
}

install_binary() {
    src="$1"

    if [ -n "${INSTALL_DIR:-}" ]; then
        dir="${INSTALL_DIR}"
    elif [ -w "/usr/local/bin" ]; then
        dir="/usr/local/bin"
    else
        dir="${HOME}/.local/bin"
    fi

    mkdir -p "${dir}"
    if ! cp "${src}" "${dir}/${BINARY_NAME}" 2> /dev/null; then
        log_error "无法写入 ${dir}，请用 sudo 重新执行，或指定 INSTALL_DIR"
        exit 1
    fi
    chmod +x "${dir}/${BINARY_NAME}"
    INSTALLED_PATH="${dir}/${BINARY_NAME}"
    log_info "已安装到 ${INSTALLED_PATH}"

    case ":${PATH}:" in
        *":${dir}:"*) ;;
        *)
            log_warn "${dir} 不在 PATH 中，请将下面一行加入 shell 配置："
            log_warn "  export PATH=\"${dir}:\$PATH\""
            ;;
    esac
}

compile_from_source() {
    log_warn "改为从源码编译（需要 Go 1.25+）"
    if ! command -v go > /dev/null 2>&1; then
        log_error "未安装 Go。请从 https://go.dev/dl/ 安装，或使用 CORTEX_VERSION 指定已发布版本。"
        exit 1
    fi
    if [ ! -f "go.mod" ]; then
        log_error "当前目录不是 Cortex 源码仓库，无法从源码编译。"
        log_error "请先 git clone https://github.com/${REPO}.git 后执行 scripts/install.sh"
        exit 1
    fi

    go build -trimpath -ldflags="-s -w" -o "${TMP_DIR}/${BINARY_NAME}" ./cmd/cortex
    INSTALLED_PATH="${TMP_DIR}/${BINARY_NAME}"
    install_binary "${TMP_DIR}/${BINARY_NAME}"
}

verify_install() {
    "$1" --version || true
    log_info "✅ Cortex 安装完成：$1"
}

main() {
    DOC_DIR="${1:-}"

    printf '\n  ⚡ Installing Cortex...\n\n'

    detect_os
    resolve_urls

    TMP_DIR="$(mktemp -d 2> /dev/null || mktemp -d -t cortex)"
    mkdir -p "${CONFIG_DIR}"

    if [ "${CORTEX_FROM_SOURCE}" = "1" ]; then
        compile_from_source
    else
        if ! install_from_release; then
            log_warn "GitHub Releases 安装失败，尝试从源码编译"
            compile_from_source
        fi
    fi

    verify_install "${INSTALLED_PATH}"

    if [ -n "${DOC_DIR}" ]; then
        log_info "运行：cortex install ${DOC_DIR}"
        "${INSTALLED_PATH}" install "${DOC_DIR}" || log_warn "自动配置失败，可稍后手动执行 cortex install ${DOC_DIR}"
    else
        log_info "下一步：cortex install <你的文档目录>"
    fi

    printf '\n'
    log_info "🎉  Cortex is ready!"
    printf '\n'
}

main "$@"
