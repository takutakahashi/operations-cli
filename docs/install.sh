#!/bin/sh
set -e

# Script version
SCRIPT_VERSION="1.0.0"

# Default values
DEFAULT_INSTALL_DIR="/usr/local/bin"
OWNER="takutakahashi"
REPO="operation-mcp"
VERSION=""
BINARY_NAME="operations"
INSTALL_DIR=""
DRY_RUN=false
FORCE=false
HELP=false

print_help() {
    cat << EOF
Install script for operations CLI tool

USAGE:
    curl -fsSL https://takutakahashi.github.io/operation-mcp/install.sh | sh -s -- [OPTIONS]

OPTIONS:
    -v, --version VERSION    Specify version to install (default: latest)
    -d, --dir DIRECTORY      Specify installation directory (default: /usr/local/bin)
    -f, --force              Skip confirmation prompt
    --dry-run                Show what would be done without making changes
    -h, --help               Show this help message

EXAMPLES:
    # Install latest version
    curl -fsSL https://takutakahashi.github.io/operation-mcp/install.sh | sh

    # Install specific version
    curl -fsSL https://takutakahashi.github.io/operation-mcp/install.sh | sh -s -- -v v1.0.0

    # Install to custom directory
    curl -fsSL https://takutakahashi.github.io/operation-mcp/install.sh | sh -s -- -d ~/bin
EOF
}

# Parse arguments
while [ $# -gt 0 ]; do
    case "$1" in
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        -d|--dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        -f|--force)
            FORCE=true
            shift
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        -h|--help)
            HELP=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            print_help
            exit 1
            ;;
    esac
done

if [ "$HELP" = true ]; then
    print_help
    exit 0
fi

# Set install directory
if [ -z "$INSTALL_DIR" ]; then
    INSTALL_DIR="$DEFAULT_INSTALL_DIR"
fi

# Detect OS and architecture
detect_os_arch() {
    # Detect OS
    if [ "$(uname)" = "Darwin" ]; then
        OS="darwin"
    elif [ "$(uname)" = "Linux" ]; then
        OS="linux"
    else
        echo "Unsupported OS: $(uname)"
        exit 1
    fi

    # Detect architecture
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64)
            ARCH_NAME="x86_64"
            ;;
        aarch64|arm64)
            ARCH_NAME="aarch64"
            ;;
        *)
            echo "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac
}

# Fetch latest release information
fetch_latest_release() {
    if [ -z "$VERSION" ]; then
        echo "Fetching latest release..."
        RELEASE_URL="https://api.github.com/repos/$OWNER/$REPO/releases/latest"
    else
        echo "Fetching release $VERSION..."
        # Ensure version has v prefix
        if ! echo "$VERSION" | grep -q "^v"; then
            VERSION="v$VERSION"
        fi
        RELEASE_URL="https://api.github.com/repos/$OWNER/$REPO/releases/tags/$VERSION"
    fi

    # Use curl to fetch release information
    if ! RELEASE_INFO=$(curl -s "$RELEASE_URL"); then
        echo "Failed to fetch release information"
        exit 1
    fi

    # Check if release info contains error message
    if echo "$RELEASE_INFO" | grep -q "Not Found"; then
        echo "Release not found"
        exit 1
    fi

    # Extract version
    TAG_NAME=$(echo "$RELEASE_INFO" | grep -o '"tag_name":"[^"]*' | sed 's/"tag_name":"//g')
    if [ -z "$TAG_NAME" ]; then
        echo "Failed to extract tag name from release information"
        exit 1
    fi

    echo "Found release: $TAG_NAME"
}

# Find the correct asset URL for download
find_asset_url() {
    # Example asset name: operations_v0.1.0_linux_x86_64.tar.gz
    ASSET_PATTERN="$BINARY_NAME\_.*\_$OS\_$ARCH_NAME\.tar\.gz"
    ASSET_URL=$(echo "$RELEASE_INFO" | grep -o "\"browser_download_url\":\"[^\"]*$ASSET_PATTERN\"" | sed 's/"browser_download_url":"//g' | sed 's/"//g')

    if [ -z "$ASSET_URL" ]; then
        echo "No matching asset found for $OS $ARCH_NAME"
        exit 1
    fi

    echo "Found asset: $ASSET_URL"
}

# Download and install the binary
download_and_install() {
    # Create temporary directory
    TMP_DIR=$(mktemp -d)
    echo "Working in temporary directory: $TMP_DIR"

    # Download asset
    ARCHIVE_PATH="$TMP_DIR/$(basename "$ASSET_URL")"
    echo "Downloading to $ARCHIVE_PATH..."
    
    if [ "$DRY_RUN" = true ]; then
        echo "[DRY RUN] Would download $ASSET_URL to $ARCHIVE_PATH"
    else
        if ! curl -sL -o "$ARCHIVE_PATH" "$ASSET_URL"; then
            echo "Failed to download asset"
            rm -rf "$TMP_DIR"
            exit 1
        fi
    fi

    # Extract archive
    echo "Extracting archive..."
    if [ "$DRY_RUN" = true ]; then
        echo "[DRY RUN] Would extract $ARCHIVE_PATH to $TMP_DIR"
    else
        tar -xzf "$ARCHIVE_PATH" -C "$TMP_DIR"
    fi

    # Find the binary
    BINARY_PATH="$TMP_DIR/$BINARY_NAME"
    if [ ! -f "$BINARY_PATH" ] && [ "$DRY_RUN" = false ]; then
        echo "Binary not found in extracted archive"
        rm -rf "$TMP_DIR"
        exit 1
    fi

    # Create install directory if it doesn't exist
    if [ ! -d "$INSTALL_DIR" ] && [ "$DRY_RUN" = false ]; then
        echo "Creating install directory: $INSTALL_DIR"
        mkdir -p "$INSTALL_DIR"
    fi

    # Check if binary already exists
    DEST_PATH="$INSTALL_DIR/$BINARY_NAME"
    if [ -f "$DEST_PATH" ] && [ "$FORCE" = false ] && [ "$DRY_RUN" = false ]; then
        printf "Binary already exists at %s. Overwrite? [y/N] " "$DEST_PATH"
        read -r RESPONSE
        if [ "$RESPONSE" != "y" ] && [ "$RESPONSE" != "Y" ]; then
            echo "Installation aborted"
            rm -rf "$TMP_DIR"
            exit 1
        fi
    fi

    # Install binary
    if [ "$DRY_RUN" = true ]; then
        echo "[DRY RUN] Would install $BINARY_PATH to $DEST_PATH"
    else
        echo "Installing binary to $DEST_PATH..."
        if ! cp "$BINARY_PATH" "$DEST_PATH"; then
            echo "Failed to install binary. Do you have permission to write to $INSTALL_DIR?"
            echo "Try running with sudo or specifying a different directory with -d/--dir"
            rm -rf "$TMP_DIR"
            exit 1
        fi

        # Set execute permissions
        chmod +x "$DEST_PATH"
    fi

    # Clean up
    rm -rf "$TMP_DIR"

    echo "Installation complete!"
    echo "You can now run: $DEST_PATH"

    # Check if directory is in PATH
    if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
        echo "Warning: $INSTALL_DIR is not in your PATH"
        echo "Add it to your PATH to run $BINARY_NAME from anywhere"
        echo "For example, add this to your ~/.bashrc or ~/.zshrc:"
        echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
    fi
}

# Main flow
main() {
    echo "Operations CLI installer (script version: $SCRIPT_VERSION)"

    detect_os_arch
    echo "Detected OS: $OS, Architecture: $ARCH_NAME"

    fetch_latest_release
    find_asset_url
    download_and_install

    echo "Thank you for installing Operations CLI!"
}

main