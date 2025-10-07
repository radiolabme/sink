#!/usr/bin/env bash
set -e

# setup.sh - Development Environment Setup
#
# Checks for Go and Make availability. If both are found in your system PATH,
# displays 'make help' and exits. If either is missing, downloads and installs
# them locally to the .tools/ directory.
#
# Usage:
#   ./setup.sh
#
# After installation, if local tools were installed, you must run:
#   source .envrc
# This adds the local tools to your PATH for the current terminal session.
#
# Supported platforms: Linux and macOS (Darwin)
# Supported architectures: AMD64 and ARM64

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TOOLS_DIR="${SCRIPT_DIR}/.tools"
GO_VERSION="1.23.4"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    
    case "${OS}" in
        darwin) OS="darwin" ;;
        linux) OS="linux" ;;
        *)
            log_error "Unsupported operating system: ${OS}"
            exit 1
            ;;
    esac
    
    case "${ARCH}" in
        x86_64) ARCH="amd64" ;;
        amd64) ARCH="amd64" ;;
        arm64) ARCH="arm64" ;;
        aarch64) ARCH="arm64" ;;
        *)
            log_error "Unsupported architecture: ${ARCH}"
            exit 1
            ;;
    esac
    
    log_info "Detected platform: ${OS}/${ARCH}"
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check if Go is available (system or local)
check_go() {
    if command_exists go; then
        GO_BIN=$(command -v go)
        GO_VER=$(go version | awk '{print $3}' | sed 's/go//')
        log_info "Go found: ${GO_BIN} (${GO_VER})"
        return 0
    elif [ -f "${TOOLS_DIR}/go/bin/go" ]; then
        GO_BIN="${TOOLS_DIR}/go/bin/go"
        GO_VER=$(${GO_BIN} version | awk '{print $3}' | sed 's/go//')
        log_info "Local Go found: ${GO_BIN} (${GO_VER})"
        export PATH="${TOOLS_DIR}/go/bin:${PATH}"
        return 0
    fi
    return 1
}

# Check if Make is available (system or local)
check_make() {
    if command_exists make; then
        MAKE_BIN=$(command -v make)
        log_info "Make found: ${MAKE_BIN}"
        return 0
    elif [ -f "${TOOLS_DIR}/bin/make" ]; then
        MAKE_BIN="${TOOLS_DIR}/bin/make"
        log_info "Local Make found: ${MAKE_BIN}"
        export PATH="${TOOLS_DIR}/bin:${PATH}"
        return 0
    fi
    return 1
}

# Install Go locally
install_go() {
    log_info "Installing Go ${GO_VERSION} to ${TOOLS_DIR}/go..."
    
    mkdir -p "${TOOLS_DIR}"
    cd "${TOOLS_DIR}"
    
    GO_ARCHIVE="go${GO_VERSION}.${OS}-${ARCH}.tar.gz"
    GO_URL="https://go.dev/dl/${GO_ARCHIVE}"
    
    log_info "Downloading: ${GO_URL}"
    if command_exists curl; then
        curl -fsSL -O "${GO_URL}"
    elif command_exists wget; then
        wget -q "${GO_URL}"
    else
        log_error "Neither curl nor wget found. Cannot download Go."
        exit 1
    fi
    
    log_info "Extracting Go..."
    tar -xzf "${GO_ARCHIVE}"
    rm "${GO_ARCHIVE}"
    
    export PATH="${TOOLS_DIR}/go/bin:${PATH}"
    log_info "Go installed: $(${TOOLS_DIR}/go/bin/go version)"
}

# Install Make locally (for macOS where it might be missing)
install_make() {
    log_info "Installing Make to ${TOOLS_DIR}/bin..."
    
    if [ "${OS}" = "darwin" ]; then
        # On macOS, check for Xcode Command Line Tools
        if ! xcode-select -p >/dev/null 2>&1; then
            log_warn "Xcode Command Line Tools not found"
            log_info "Installing Xcode Command Line Tools (this may prompt for permission)..."
            xcode-select --install
            echo ""
            echo "⏸️  Please complete the Xcode Command Line Tools installation,"
            echo "   then run this script again."
            exit 0
        fi
        
        # Make should now be available via Xcode tools
        if command_exists make; then
            log_info "Make is now available via Xcode Command Line Tools"
            return 0
        else
            log_error "Make not found even after Xcode tools check"
            exit 1
        fi
    else
        # On Linux, suggest package manager installation
        log_error "Make not found. Please install it using your package manager:"
        echo ""
        echo "  # Debian/Ubuntu:"
        echo "  sudo apt-get install build-essential"
        echo ""
        echo "  # RHEL/CentOS/Fedora:"
        echo "  sudo yum groupinstall 'Development Tools'"
        echo ""
        echo "  # Alpine:"
        echo "  sudo apk add make gcc musl-dev"
        echo ""
        exit 1
    fi
}

# Create environment setup helper
create_env_helper() {
    cat > "${SCRIPT_DIR}/.envrc" << 'EOF'
# Source this file to add local tools to PATH
# Usage: source .envrc

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export PATH="${SCRIPT_DIR}/.tools/go/bin:${SCRIPT_DIR}/.tools/bin:${PATH}"
export GOPATH="${SCRIPT_DIR}/.go"

echo "✅ Local tools added to PATH"
echo "   Go: $(go version 2>/dev/null || echo 'not found')"
echo "   Make: $(make --version 2>/dev/null | head -1 || echo 'not found')"
EOF
    log_info "Created .envrc helper (source it to use local tools)"
}

# Main setup flow
main() {
    log_info "Starting Sink development environment setup..."
    echo ""
    
    detect_platform
    echo ""
    
    # Check and install Go if needed
    if ! check_go; then
        log_warn "Go not found"
        install_go
    fi
    echo ""
    
    # Check and install Make if needed  
    if ! check_make; then
        log_warn "Make not found"
        install_make
    fi
    echo ""
    
    # Create environment helper
    create_env_helper
    echo ""
    
    # Success - show make help
    log_info "Setup complete! All tools ready."
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    
    # If using local tools, remind user to update PATH
    if [ -d "${TOOLS_DIR}" ]; then
        echo "📌 Local tools installed. To use them, run this command in your terminal:"
        echo ""
        echo "   source .envrc"
        echo ""
        echo "   This adds the local Go and Make to your PATH for this session."
        echo "   You'll need to run this command each time you open a new terminal,"
        echo "   or add the PATH export to your ~/.zshrc or ~/.bashrc file."
        echo ""
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo ""
    fi
    
    # Show make help
    cd "${SCRIPT_DIR}"
    make help
}

main "$@"
