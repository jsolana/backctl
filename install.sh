#!/bin/bash
set -eufo pipefail

REPO="github.com/jsolana/backctl"
REPO_URL="https://${REPO}.git"
BINARY_NAME="backctl"
INSTALL_DIR="/usr/local/bin"

# Parse arguments
VERSION=""
REPO_PATH=""
while [[ $# -gt 0 ]]; do
    case $1 in
        -p|--path)
            REPO_PATH="${2:?Error: --path requires a value}"
            shift 2
            ;;
        -v|--version)
            VERSION="${2:?Error: --version requires a value}"
            shift 2
            ;;
        -d|--dir)
            INSTALL_DIR="${2:?Error: --dir requires a value}"
            shift 2
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  -v, --version TAG    Install a specific version/tag (default: latest)"
            echo "  -p, --path DIR       Build from a local checkout instead of cloning"
            echo "  -d, --dir DIR        Installation directory (default: /usr/local/bin)"
            echo "  -h, --help           Show this help"
            echo ""
            echo "Examples:"
            echo "  $0                          # install latest from main"
            echo "  $0 -v v0.2.0               # install specific tag"
            echo "  $0 -p ~/dev/backctl        # build from local path"
            echo "  $0 -d ~/.local/bin         # install to custom dir"
            exit 0
            ;;
        *)
            echo "Unknown option: $1. Use --help for usage."
            exit 1
            ;;
    esac
done

if [ -n "$REPO_PATH" ] && [ -n "$VERSION" ]; then
    echo "Error: --path and --version cannot be used together"
    exit 1
fi

# Check Go is available
if ! command -v go >/dev/null 2>&1; then
    echo "Error: Go is not installed."
    echo ""
    echo "Install Go from https://go.dev/dl/ or use:"
    echo "  go install ${REPO}/cmd/backctl@latest"
    echo ""
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}')
echo "Using $GO_VERSION"

# Build from source
TMP_DIR=$(mktemp -d)
cleanup() {
    rm -rf "$TMP_DIR"
}
trap cleanup EXIT

SRC_DIR=""
if [ -n "$REPO_PATH" ]; then
    SRC_DIR="$(cd "$REPO_PATH" && pwd)"
    echo "Building from local path: $SRC_DIR"
else
    BRANCH="${VERSION:-main}"

    echo -n "Cloning ${REPO} (${BRANCH}) ... "
    if ! git clone --depth=1 --quiet -b "$BRANCH" "$REPO_URL" "$TMP_DIR/backctl" 2>/dev/null; then
        echo "Failed!"
        echo "Error: Could not clone branch/tag '$BRANCH'. Check it exists."
        exit 1
    fi
    echo "Done"
    SRC_DIR="$TMP_DIR/backctl"
fi

# Determine version info for ldflags
cd "$SRC_DIR"
BUILD_VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}"
BUILD_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE=$(date -u '+%Y-%m-%dT%H:%M:%SZ')

LDFLAGS="-X main.version=${BUILD_VERSION} -X main.commit=${BUILD_COMMIT} -X main.date=${BUILD_DATE}"

echo -n "Building backctl ${BUILD_VERSION} (${BUILD_COMMIT}) ... "
if ! go build -ldflags "$LDFLAGS" -o "$TMP_DIR/backctl-bin" ./cmd/backctl 2>/dev/null; then
    go build -ldflags "$LDFLAGS" -o "$TMP_DIR/backctl-bin" ./cmd/backctl
    exit 1
fi
echo "Done"

# Install binary
echo -n "Installing to ${INSTALL_DIR}/${BINARY_NAME} ... "
mkdir -p "$INSTALL_DIR"
if [ -w "$INSTALL_DIR" ]; then
    mv "$TMP_DIR/backctl-bin" "${INSTALL_DIR}/${BINARY_NAME}"
else
    sudo mv "$TMP_DIR/backctl-bin" "${INSTALL_DIR}/${BINARY_NAME}"
fi
chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
echo "Done"

echo ""
echo "backctl ${BUILD_VERSION} installed successfully!"
echo ""
echo "Verify with:"
echo "  backctl version"
echo ""
echo "Configuration:"
echo "  export BACKSTAGE_URL=https://backstage.example.com"
echo "  export BACKSTAGE_TOKEN=<your-token>"
