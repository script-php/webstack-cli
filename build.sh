#!/bin/bash
# WebStack CLI - Build Script for Multiple Platforms

set -e

VERSION=${1:-"v1.0.0"}
BINARY_NAME="webstack"
BUILD_DIR="build"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

log() { echo -e "${GREEN}[BUILD]${NC} $1"; }
info() { echo -e "${BLUE}[INFO]${NC} $1"; }

# Clean previous builds
clean() {
    log "Cleaning previous builds..."
    rm -rf "$BUILD_DIR"
    mkdir -p "$BUILD_DIR"
}

# Build for platform
build_platform() {
    local os=$1
    local arch=$2
    local extension=""
    
    if [[ "$os" == "windows" ]]; then
        extension=".exe"
    fi
    
    local output="${BUILD_DIR}/${BINARY_NAME}-${os}-${arch}${extension}"
    
    log "Building for ${os}/${arch}..."
    
    GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build \
        -ldflags "-s -w -X main.Version=${VERSION}" \
        -o "$output" \
        .
    
    # Compress with upx if available (optional)
    if command -v upx >/dev/null 2>&1 && [[ "$os" != "darwin" ]]; then
        log "Compressing ${os}/${arch} binary..."
        upx --best --lzma "$output" 2>/dev/null || true
    fi
}

# Generate checksums
generate_checksums() {
    log "Generating checksums..."
    cd "$BUILD_DIR"
    
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum * > checksums.txt
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 * > checksums.txt
    fi
    
    cd ..
}

# Create GitHub release assets
create_release_assets() {
    log "Creating release assets..."
    
    # Create templates archive
    tar -czf "${BUILD_DIR}/webstack-templates.tar.gz" templates/
    
    # Create example configs
    tar -czf "${BUILD_DIR}/webstack-examples.tar.gz" examples/ || true
    
    # Create installation script
    cp install.sh "${BUILD_DIR}/"
}

# Main build function
main() {
    info "WebStack CLI - Multi-platform Build"
    info "Version: $VERSION"
    info "===================================="
    
    clean
    
    # Build for different platforms
    build_platform "linux" "amd64"
    build_platform "linux" "arm64"
    build_platform "linux" "arm"
    build_platform "darwin" "amd64"
    build_platform "darwin" "arm64"
    build_platform "windows" "amd64"
    
    generate_checksums
    create_release_assets
    
    log "Build completed successfully!"
    log "Artifacts are in the '${BUILD_DIR}' directory"
    
    # Show build results
    info "Build Results:"
    ls -la "$BUILD_DIR"
}

main "$@"