#!/bin/sh
# go-chrome-ai installer
# Usage: curl -fsSL https://raw.githubusercontent.com/itamaker/go-chrome-ai/main/scripts/install.sh | sh

set -eu

REPO="${REPO:-itamaker/go-chrome-ai}"
BINARY_NAME="go-chrome-ai"
VERSION="${VERSION:-latest}"
SKIP_PATH_SETUP="${SKIP_PATH_SETUP:-0}"
VERSION_TAG=""
VERSION_NUMBER=""
LEGACY_ARCHIVE_NAME=""
LEGACY_DOWNLOAD_URL=""
LEGACY_CHECKSUMS_URL=""

if [ "$(id -u)" = "0" ]; then
    INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
else
    INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
fi

if [ -t 1 ]; then
    RED="$(printf '\033[0;31m')"
    GREEN="$(printf '\033[38;2;23;128;68m')"
    YELLOW="$(printf '\033[1;33m')"
    CYAN="$(printf '\033[0;36m')"
    BLUE="$(printf '\033[0;34m')"
    BOLD="$(printf '\033[1m')"
    NC="$(printf '\033[0m')"
else
    RED=''
    GREEN=''
    YELLOW=''
    CYAN=''
    BLUE=''
    BOLD=''
    NC=''
fi

say() {
    printf '%b\n' "$1"
}

info() {
    say "${BLUE}==>${NC} $1"
}

success() {
    say "${GREEN}OK${NC}  $1"
}

warn() {
    say "${YELLOW}WARN${NC} $1"
}

error() {
    say "${RED}ERR${NC}  $1" >&2
    exit 1
}

print_banner() {
    printf '%b' "${GREEN}${BOLD}"
    cat <<'EOF'

  ██████╗  ██████╗
 ██╔════╝ ██╔═══██╗
 ██║  ███╗██║   ██║
 ██║   ██║██║   ██║
 ╚██████╔╝╚██████╔╝
  ╚═════╝  ╚═════╝

  ██████╗██╗  ██╗██████╗  ██████╗ ███╗   ███╗███████╗      █████╗ ██╗
 ██╔════╝██║  ██║██╔══██╗██╔═══██╗████╗ ████║██╔════╝     ██╔══██╗██║
 ██║     ███████║██████╔╝██║   ██║██╔████╔██║█████╗       ███████║██║
 ██║     ██╔══██║██╔══██╗██║   ██║██║╚██╔╝██║██╔══╝       ██╔══██║██║
 ╚██████╗██║  ██║██║  ██║╚██████╔╝██║ ╚═╝ ██║███████╗     ██║  ██║██║
  ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝ ╚═╝     ╚═╝╚══════╝     ╚═╝  ╚═╝╚═╝

EOF
    printf '%b\n' "${NC}"
}

need_cmd() {
    command -v "$1" >/dev/null 2>&1 || error "Missing required command: $1"
}

github_api_get() {
    if [ -n "${GITHUB_TOKEN:-}" ]; then
        curl -fsSL \
            -H "Accept: application/vnd.github+json" \
            -H "Authorization: Bearer $GITHUB_TOKEN" \
            "$1"
    else
        curl -fsSL -H "Accept: application/vnd.github+json" "$1"
    fi
}

detect_platform() {
    OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
    case "$OS" in
        darwin) OS="darwin" ;;
        linux) OS="linux" ;;
        mingw*|msys*|cygwin*) error "Windows release archives are published as .zip files. Download them directly from GitHub Releases." ;;
        *) error "Unsupported OS: $OS" ;;
    esac

    ARCH="$(uname -m)"
    case "$ARCH" in
        x86_64|amd64) ARCH="amd64" ;;
        arm64|aarch64) ARCH="arm64" ;;
        *) error "Unsupported architecture: $ARCH" ;;
    esac
}

resolve_version() {
    if [ "$VERSION" = "latest" ]; then
        latest_json="$(github_api_get "https://api.github.com/repos/$REPO/releases/latest")" || \
            error "Failed to query the latest GitHub release"
        VERSION_TAG="$(printf '%s\n' "$latest_json" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"
        [ -n "$VERSION_TAG" ] || error "Failed to parse the latest release version"
    else
        case "$VERSION" in
            v*)
                VERSION_TAG="$VERSION"
                ;;
            *)
                VERSION_TAG="v$VERSION"
                ;;
        esac
    fi

    VERSION_NUMBER="${VERSION_TAG#v}"
    ARCHIVE_NAME="${BINARY_NAME}_${VERSION_NUMBER}_${OS}_${ARCH}.tar.gz"
    DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION_TAG/$ARCHIVE_NAME"
    CHECKSUMS_URL="https://github.com/$REPO/releases/download/$VERSION_TAG/checksums.txt"
    LEGACY_ARCHIVE_NAME="${BINARY_NAME}-${OS}-${ARCH}.tar.gz"
    LEGACY_DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION_TAG/$LEGACY_ARCHIVE_NAME"
    LEGACY_CHECKSUMS_URL="https://github.com/$REPO/releases/download/$VERSION_TAG/SHA256SUMS"

    info "Version: ${BOLD}$VERSION_TAG${NC}"
}

create_install_dir() {
    [ -d "$INSTALL_DIR" ] && return 0

    info "Creating install directory: ${BOLD}$INSTALL_DIR${NC}"
    if mkdir -p "$INSTALL_DIR" 2>/dev/null; then
        return 0
    fi

    if command -v sudo >/dev/null 2>&1; then
        sudo mkdir -p "$INSTALL_DIR"
        return 0
    fi

    error "Cannot create $INSTALL_DIR. Set INSTALL_DIR to a writable directory."
}

verify_checksum() {
    archive_path="$1"
    checksums_path="$2"

    if ! command -v shasum >/dev/null 2>&1 && ! command -v sha256sum >/dev/null 2>&1; then
        warn "Skipping checksum verification: shasum/sha256sum not found"
        return 0
    fi

    if ! curl -fsSL "$CHECKSUMS_URL" -o "$checksums_path"; then
        warn "Skipping checksum verification: failed to download checksums.txt"
        return 0
    fi

    expected="$(grep " ${ARCHIVE_NAME}\$" "$checksums_path" | awk '{print $1}' | head -n 1)"
    [ -n "$expected" ] || {
        warn "Skipping checksum verification: checksum entry not found for $ARCHIVE_NAME"
        return 0
    }

    if command -v shasum >/dev/null 2>&1; then
        actual="$(shasum -a 256 "$archive_path" | awk '{print $1}')"
    else
        actual="$(sha256sum "$archive_path" | awk '{print $1}')"
    fi

    [ "$actual" = "$expected" ] || error "Checksum verification failed for $ARCHIVE_NAME"
    success "Checksum verified"
}

install_binary() {
    tmp_dir="$(mktemp -d 2>/dev/null || mktemp -d -t go-chrome-ai-install)"
    trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

    archive_path="$tmp_dir/$ARCHIVE_NAME"
    checksums_path="$tmp_dir/checksums.txt"

    info "Downloading ${BOLD}$ARCHIVE_NAME${NC}"
    if ! curl -fsSL "$DOWNLOAD_URL" -o "$archive_path"; then
        warn "Preferred archive name not found, trying legacy release naming"
        ARCHIVE_NAME="$LEGACY_ARCHIVE_NAME"
        DOWNLOAD_URL="$LEGACY_DOWNLOAD_URL"
        CHECKSUMS_URL="$LEGACY_CHECKSUMS_URL"
        curl -fsSL "$DOWNLOAD_URL" -o "$archive_path" || error "Download failed: $DOWNLOAD_URL"
    fi

    verify_checksum "$archive_path" "$checksums_path"

    info "Extracting archive"
    tar -xzf "$archive_path" -C "$tmp_dir" || error "Failed to extract archive"
    [ -f "$tmp_dir/$BINARY_NAME" ] || error "Archive did not contain $BINARY_NAME"

    create_install_dir

    info "Installing to ${BOLD}$INSTALL_DIR${NC}"
    if [ -w "$INSTALL_DIR" ]; then
        install -m 0755 "$tmp_dir/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
    elif command -v sudo >/dev/null 2>&1; then
        sudo install -m 0755 "$tmp_dir/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
    else
        error "Install directory is not writable and sudo is unavailable. Set INSTALL_DIR to a writable directory."
    fi

    INSTALL_PATH="$INSTALL_DIR/$BINARY_NAME"
    success "Installed ${BOLD}$BINARY_NAME${NC}"
}

setup_path() {
    if [ "$SKIP_PATH_SETUP" = "1" ]; then
        info "Skipping shell PATH changes"
        return 0
    fi

    case ":$PATH:" in
        *":$INSTALL_DIR:"*) return 0 ;;
    esac

    shell_name="$(basename "${SHELL:-sh}")"
    case "$shell_name" in
        bash)
            shell_rc="$HOME/.bashrc"
            path_line="export PATH=\"\$PATH:$INSTALL_DIR\""
            ;;
        zsh)
            shell_rc="$HOME/.zshrc"
            path_line="export PATH=\"\$PATH:$INSTALL_DIR\""
            ;;
        fish)
            shell_rc="$HOME/.config/fish/config.fish"
            path_line="fish_add_path \"$INSTALL_DIR\""
            ;;
        *)
            shell_rc="$HOME/.profile"
            path_line="export PATH=\"\$PATH:$INSTALL_DIR\""
            ;;
    esac

    if [ -f "$shell_rc" ] && grep -F "$INSTALL_DIR" "$shell_rc" >/dev/null 2>&1; then
        return 0
    fi

    mkdir -p "$(dirname "$shell_rc")"
    printf '\n%s\n' "$path_line" >> "$shell_rc"
    warn "Added $INSTALL_DIR to PATH in $shell_rc. Restart your terminal or reload that file."
}

print_next_steps() {
    say ""
    say "${GREEN}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    say "${GREEN}${BOLD}  QUICK START${NC}"
    say "${GREEN}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    say ""
    say "  ${BOLD}Run the CLI:${NC}"
    say "    ${CYAN}$BINARY_NAME${NC}"
    say ""
    say "  ${BOLD}Preview changes without writing:${NC}"
    say "    ${CYAN}$BINARY_NAME -dry-run${NC}"
    say ""
    if [ "$OS" = "darwin" ]; then
        say "  ${BOLD}Launch the GUI:${NC}"
        say "    ${CYAN}$BINARY_NAME gui${NC}"
        say ""
        say "  ${BOLD}If macOS blocks the first launch:${NC}"
        say "    ${CYAN}xattr -d com.apple.quarantine \"$INSTALL_PATH\"${NC}"
        say ""
    else
        say "  ${BOLD}GUI mode:${NC}"
        say "    Build from source on Linux if you want the Fyne desktop app."
        say ""
    fi
    say "${GREEN}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

main() {
    print_banner

    need_cmd curl
    need_cmd tar
    need_cmd install

    detect_platform
    info "Platform: ${BOLD}$OS-$ARCH${NC}"
    resolve_version
    install_binary
    setup_path
    print_next_steps
}

main "$@"
