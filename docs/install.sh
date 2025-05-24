#!/bin/sh
set -e

# Script version
SCRIPT_VERSION="1.0.0"

# Default values
DEFAULT_INSTALL_DIR="${HOME}/.local/bin"
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
    -d, --dir DIRECTORY      Specify installation directory (default: ${DEFAULT_INSTALL_DIR})
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

check_current_version() {
    CURRENT_VERSION=""
    DEST_PATH="$INSTALL_DIR/$BINARY_NAME"
    
    if [ -f "$DEST_PATH" ] && [ -x "$DEST_PATH" ]; then
        if CURRENT_VERSION_OUTPUT=$("$DEST_PATH" --version 2>/dev/null); then
            # Extract version from output like "operations v0.6.31 (commit: abc123, built: 2023-01-01)"
            CURRENT_VERSION=$(echo "$CURRENT_VERSION_OUTPUT" | sed -n 's/^operations \(v[0-9][^[:space:]]*\).*/\1/p')
        fi
    fi
}

# Fetch latest release information
fetch_latest_release() {
    if [ -z "$VERSION" ]; then
        echo "Fetching latest release..."
        RELEASE_URL="https://github.com/$OWNER/$REPO/releases/latest"
        
        if ! REDIRECT_URL=$(curl -sI "$RELEASE_URL" | grep -i "^location:" | sed 's/^location: *//i' | tr -d '\r'); then
            echo "Failed to fetch latest release information"
            exit 1
        fi
        
        # Extract tag from redirect URL like https://github.com/owner/repo/releases/tag/v0.6.31
        TAG_NAME=$(echo "$REDIRECT_URL" | sed -n 's|.*/releases/tag/\(.*\)|\1|p')
    else
        echo "Fetching release $VERSION..."
        # Ensure version has v prefix
        if ! echo "$VERSION" | grep -q "^v"; then
            VERSION="v$VERSION"
        fi
        TAG_NAME="$VERSION"
        
        RELEASE_PAGE_URL="https://github.com/$OWNER/$REPO/releases/tag/$TAG_NAME"
        if ! curl -sI "$RELEASE_PAGE_URL" | grep -q "200 OK"; then
            echo "Release $TAG_NAME not found"
            exit 1
        fi
    fi

    if [ -z "$TAG_NAME" ]; then
        echo "Failed to determine release version"
        exit 1
    fi

    echo "Found release: $TAG_NAME"
}

# Find the correct asset URL for download
find_asset_url() {
    VERSION_NUM=${TAG_NAME#v}
    
    ASSET_URL="https://github.com/$OWNER/$REPO/releases/download/$TAG_NAME/operation-mcp_${VERSION_NUM}_${OS}_${ARCH_NAME}.tar.gz"
    
    HTTP_STATUS=$(curl -sI "$ASSET_URL" | head -n 1)
    if ! echo "$HTTP_STATUS" | grep -q "302\|200"; then
        echo "No matching asset found for $OS $ARCH_NAME"
        echo "URL: $ASSET_URL"
        echo "Status: $HTTP_STATUS"
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

    check_current_version
    
    fetch_latest_release
    
    # Check if we already have the latest version
    if [ -n "$CURRENT_VERSION" ] && [ "$CURRENT_VERSION" = "$TAG_NAME" ] && [ "$FORCE" = false ]; then
        echo "Latest version $TAG_NAME is already installed. Use --force to reinstall."
        exit 0
    fi
    
    if [ -n "$CURRENT_VERSION" ]; then
        echo "Current version: $CURRENT_VERSION"
        echo "Target version: $TAG_NAME"
    fi
    
    find_asset_url
    download_and_install

    echo "Thank you for installing Operations CLI!"
}

main
