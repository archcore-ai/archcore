#!/usr/bin/env bash

if [ -z "${BASH_VERSION:-}" ]; then
    echo "Error: this installer requires bash. Re-run with:" >&2
    echo "  curl -fsSL https://archcore.ai/install.sh | bash" >&2
    exit 1
fi

set -euo pipefail

# ── Constants ────────────────────────────────────────────────────────────────
GITHUB_REPO="archcore-ai/cli"
BINARY_NAME="archcore"
DEFAULT_INSTALL_DIR="$HOME/.local/bin"

# ── Env var overrides ────────────────────────────────────────────────────────
INSTALL_DIR="${ARCHCORE_INSTALL_DIR:-$DEFAULT_INSTALL_DIR}"
PINNED_VERSION="${ARCHCORE_VERSION:-}"

# ── Color / formatting (TTY-aware) ──────────────────────────────────────────
if [[ -t 1 ]]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[0;33m'
    BLUE='\033[0;34m'
    BOLD='\033[1m'
    NC='\033[0m'
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    BOLD=''
    NC=''
fi

# ── Logging helpers ─────────────────────────────────────────────────────────
info() {
    printf '%b%s%b\n' "${BLUE}==>${NC} ${BOLD}" "$1" "${NC}"
}

success() {
    printf '%b%s%b\n' "${GREEN}==>${NC} ${BOLD}" "$1" "${NC}"
}

warn() {
    printf '%b %s\n' "${YELLOW}Warning:${NC}" "$1"
}

error_exit() {
    printf '%b %s\n' "${RED}Error:${NC}" "$1" >&2
    exit 1
}

# ── Prerequisite check ──────────────────────────────────────────────────────
need_cmd() {
    if ! command -v "$1" &>/dev/null; then
        error_exit "'$1' is required but not found. Please install it and try again."
    fi
}

# ── Platform detection ──────────────────────────────────────────────────────
detect_os() {
    local os
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    case "$os" in
        darwin) printf '%s' "darwin" ;;
        linux)  printf '%s' "linux" ;;
        *)      error_exit "Unsupported operating system: $os" ;;
    esac
}

detect_arch() {
    local arch
    arch="$(uname -m)"
    case "$arch" in
        x86_64|amd64)   printf '%s' "amd64" ;;
        arm64|aarch64)  printf '%s' "arm64" ;;
        *)              error_exit "Unsupported architecture: $arch" ;;
    esac
}

# ── Version resolution ──────────────────────────────────────────────────────
# Deliberately resolved through the github.com web redirect rather than
# api.github.com/repos/.../releases/latest. The REST API allows only 60
# unauthenticated requests per hour *per IP*, so every user behind a shared
# egress address — corporate NAT, CGNAT, CI runners — draws from one tiny
# shared budget and installs start failing with a 403. This redirect is plain
# github.com: it carries no x-ratelimit-* budget, needs no token, and needs no
# JSON parser.
get_latest_version() {
    local url="https://github.com/${GITHUB_REPO}/releases/latest"
    local redirect_url

    # HEAD without -L: curl stops at the 302 and reports the Location header.
    # stderr is discarded so a retried transport error prints curl's diagnostic
    # once per attempt into the user's terminal ahead of our own message.
    redirect_url=$(curl -fsS -I -o /dev/null -w '%{redirect_url}' \
        --retry 3 --retry-delay 2 "$url" 2>/dev/null) || \
        error_exit "Could not reach ${url}. Check your internet connection or proxy settings, or pin a version to skip this lookup: ARCHCORE_VERSION=x.y.z curl -fsSL https://archcore.ai/install.sh | bash"

    # Expected shape: https://github.com/OWNER/REPO/releases/tag/vX.Y.Z
    # A repo with no published release still answers 302, but points at the
    # bare /releases page (verified against github/gitignore and golang/go,
    # both of which have an empty releases list), so match on the tag segment
    # rather than on emptiness.
    case "$redirect_url" in
        */releases/tag/*) ;;
        *) error_exit "Could not resolve the latest version from ${url} (unexpected response). Pin a version instead: ARCHCORE_VERSION=x.y.z curl -fsSL https://archcore.ai/install.sh | bash" ;;
    esac

    printf '%s' "${redirect_url##*/tag/}"
}

# ── Download helper ─────────────────────────────────────────────────────────
download_file() {
    local url="$1"
    local output="$2"
    local curl_opts=(-fsSL --retry 3 --retry-delay 2)

    if [[ -n "${GITHUB_TOKEN:-}" ]]; then
        curl_opts+=(-H "Authorization: Bearer ${GITHUB_TOKEN}")
    fi

    curl "${curl_opts[@]}" "$url" -o "$output"
}

# ── Checksum verification ───────────────────────────────────────────────────
verify_checksum() {
    local file="$1"
    local checksums_file="$2"
    local archive_name="$3"
    local expected_checksum
    local actual_checksum

    # Fixed-string match to avoid regex injection
    expected_checksum=$(grep -F "$archive_name" "$checksums_file" | awk '{print $1}' || true)

    if [[ -z "$expected_checksum" ]]; then
        error_exit "Checksum for ${archive_name} not found in checksums.txt"
    fi

    if command -v sha256sum &>/dev/null; then
        actual_checksum=$(sha256sum "$file" | awk '{print $1}')
    elif command -v shasum &>/dev/null; then
        actual_checksum=$(shasum -a 256 "$file" | awk '{print $1}')
    else
        warn "No checksum tool found (sha256sum or shasum). Skipping verification."
        return 0
    fi

    if [[ "$actual_checksum" != "$expected_checksum" ]]; then
        error_exit "Checksum verification failed! Expected: ${expected_checksum}, actual: ${actual_checksum}"
    fi
}

# ── Atomic install ──────────────────────────────────────────────────────────
install_binary() {
    local src="$1"
    local dest_dir="$2"
    local dest="${dest_dir}/${BINARY_NAME}"

    mkdir -p "$dest_dir"

    if [[ ! -w "$dest_dir" ]]; then
        error_exit "Cannot write to ${dest_dir}. Check permissions or set ARCHCORE_INSTALL_DIR."
    fi

    # Copy to destination filesystem, chmod, then atomic rename
    local tmp_dest="${dest}.tmp.$$"
    cp "$src" "$tmp_dest"
    chmod +x "$tmp_dest"
    mv "$tmp_dest" "$dest"
}

# ── PATH check + shell guidance ─────────────────────────────────────────────
check_path() {
    local install_dir="$1"
    local install_path="${install_dir}/${BINARY_NAME}"

    local path_binary
    path_binary=$(command -v "$BINARY_NAME" 2>/dev/null || true)

    if [[ -n "$path_binary" && "$path_binary" != "$install_path" ]]; then
        printf '\n'
        warn "PATH conflict detected"
        printf '%b  Installed to: %s\n' "${YELLOW}!${NC}" "$install_path"
        printf '%b  But '\''%s'\'' resolves to: %s\n' "${YELLOW}!${NC}" "$BINARY_NAME" "$path_binary"
        printf '%b\n' "${YELLOW}!${NC}"
        printf '%b  To fix:\n' "${YELLOW}!${NC}"
        printf '%b    1. Remove the old binary: rm %s\n' "${YELLOW}!${NC}" "$path_binary"
        printf '%b    or\n' "${YELLOW}!${NC}"
        printf '%b    2. Adjust your PATH to prioritize %s\n' "${YELLOW}!${NC}" "$install_dir"
        printf '\n'
        return 0
    fi

    if [[ -z "$path_binary" ]]; then
        local shell_name shell_config
        shell_name="$(basename "${SHELL:-}")"
        case "$shell_name" in
            zsh)
                # shellcheck disable=SC2088
                shell_config="~/.zshrc" ;;
            bash)
                if [[ -f "$HOME/.bash_profile" ]]; then
                    # shellcheck disable=SC2088
                    shell_config="~/.bash_profile"
                else
                    # shellcheck disable=SC2088
                    shell_config="~/.bashrc"
                fi
                ;;
            fish)
                # shellcheck disable=SC2088
                shell_config="~/.config/fish/config.fish" ;;
            *)
                shell_config="" ;;
        esac

        printf '\n'
        printf '  Add %b%s%b to your PATH:\n' "${BOLD}" "$BINARY_NAME" "${NC}"
        printf '\n'
        if [[ "$shell_name" == "fish" ]]; then
            printf '    %bmkdir -p ~/.config/fish%b\n' "${BOLD}" "${NC}"
            printf '    %becho '\''fish_add_path %s'\'' >> $HOME/.config/fish/config.fish%b\n' "${BOLD}" "$install_dir" "${NC}"
        elif [[ -n "$shell_config" ]]; then
            printf '    %becho '\''export PATH="%s:$PATH"'\'' >> %s%b\n' "${BOLD}" "$install_dir" "$shell_config" "${NC}"
        else
            printf '  Add this to your shell config:\n'
            printf '\n'
            printf '    %bexport PATH="%s:$PATH"%b\n' "${BOLD}" "$install_dir" "${NC}"
        fi
        printf '\n'
        printf '  Restart your terminal, then run %b%s%b to get started.\n' "${BOLD}" "$BINARY_NAME" "${NC}"
    fi
}

# ── Main ────────────────────────────────────────────────────────────────────
main() {
    # Guard: $HOME must be set
    if [[ -z "${HOME:-}" ]]; then
        error_exit "\$HOME is not set. Cannot determine install directory."
    fi

    # Prerequisites
    need_cmd curl
    need_cmd tar
    need_cmd uname

    info "Installing Archcore CLI..."

    # Platform
    local os arch
    os=$(detect_os) || exit 1
    arch=$(detect_arch) || exit 1
    info "Detected platform: ${os}/${arch}"

    # Version
    local version
    if [[ -n "$PINNED_VERSION" ]]; then
        version="${PINNED_VERSION#v}"
        info "Using pinned version: ${version}"
    else
        info "Fetching latest version..."
        version=$(get_latest_version) || exit 1
        version="${version#v}"
        info "Latest version: ${version}"
    fi

    # Construct URLs
    local archive_name="${BINARY_NAME}_${os}_${arch}.tar.gz"
    local download_url="https://github.com/${GITHUB_REPO}/releases/download/v${version}/${archive_name}"
    local checksums_url="https://github.com/${GITHUB_REPO}/releases/download/v${version}/checksums.txt"

    # Temp directory with cleanup trap
    tmp_dir=$(mktemp -d)
    trap 'rm -rf "$tmp_dir"' EXIT

    # Download archive
    local archive_path="${tmp_dir}/${archive_name}"
    info "Downloading ${archive_name}..."
    if ! download_file "$download_url" "$archive_path"; then
        error_exit "Failed to download from ${download_url}. Please check that version ${version} exists."
    fi

    # Download checksums
    info "Verifying checksum..."
    local checksums_path="${tmp_dir}/checksums.txt"
    if ! download_file "$checksums_url" "$checksums_path"; then
        error_exit "Failed to download checksums from ${checksums_url}"
    fi

    # Verify
    verify_checksum "$archive_path" "$checksums_path" "$archive_name"
    success "Checksum verified"

    # Extract — try named binary first (tar slip mitigation), fallback to full
    info "Extracting..."
    if ! tar -xzf "$archive_path" -C "$tmp_dir" "$BINARY_NAME" 2>/dev/null; then
        tar -xzf "$archive_path" -C "$tmp_dir"
    fi

    local extracted_binary="${tmp_dir}/${BINARY_NAME}"
    # GoReleaser may name the binary after the repo (e.g. "cli") rather than
    # BINARY_NAME ("archcore"). If so, rename it so the rest of the script works.
    if [[ ! -f "$extracted_binary" ]]; then
        local repo_binary_name
        repo_binary_name="$(basename "$GITHUB_REPO")"
        local repo_binary="${tmp_dir}/${repo_binary_name}"
        if [[ -f "$repo_binary" ]]; then
            mv "$repo_binary" "$extracted_binary"
        else
            error_exit "Binary '${BINARY_NAME}' not found in archive."
        fi
    fi

    # Install
    info "Installing to ${INSTALL_DIR}..."
    install_binary "$extracted_binary" "$INSTALL_DIR"
    local install_path="${INSTALL_DIR}/${BINARY_NAME}"

    # Post-install verification
    if "$install_path" --help &>/dev/null; then
        success "Binary executes OK"
    else
        warn "Binary installed but --help did not exit cleanly. It may still work."
    fi

    # PATH check (informational only — install already succeeded)
    check_path "$INSTALL_DIR"

    success "Archcore CLI v${version} installed to ${install_path}"
}

main "$@"
