#!/bin/bash

# Kafka Analyzer Builder
# Builds binaries for multiple architectures and environments

set -e

VERSION=${VERSION:-"1.0.0"}
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

BINARY_NAME="kmap"
OUTPUT_DIR="./bin"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Build flags
LDFLAGS="-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}"

print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Create output directory
mkdir -p "${OUTPUT_DIR}"

build_binary() {
    local os=$1
    local arch=$2
    local output_name="${BINARY_NAME}-${os}-${arch}"
    
    if [ "$os" = "windows" ]; then
        output_name="${output_name}.exe"
    fi
    
    local output_path="${OUTPUT_DIR}/${output_name}"
    
    print_info "Building ${output_name}..."
    
    env GOOS=${os} GOARCH=${arch} go build \
        -ldflags "${LDFLAGS}" \
        -o "${output_path}" \
        . 2>&1
    
    if [ $? -eq 0 ]; then
        local size=$(du -h "${output_path}" | cut -f1)
        print_success "Built ${output_name} (${size})"
    else
        print_error "Failed to build ${output_name}"
        return 1
    fi
}

show_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Build kafka-analyzer for multiple platforms

OPTIONS:
    -p, --platform PLAT Platform (linux-amd64|linux-arm64|darwin-amd64|darwin-arm64|windows-amd64|all) [default: all]
    -v, --version VER   Version string [default: ${VERSION}]
    -c, --clean         Clean bin directory before building
    -h, --help          Show this help message

EXAMPLES:
    # Build for all platforms
    $0

    # Build for Linux AMD64 only
    $0 --platform linux-amd64

    # Build for Linux ARM64
    $0 --platform linux-arm64

    # Clean and build everything
    $0 --clean

    # Build with custom version
    $0 --version 2.0.0

EOF
}

# Parse command line arguments
PLATFORM="all"
CLEAN=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -p|--platform)
            PLATFORM="$2"
            shift 2
            ;;
        -v|--version)
            VERSION="$2"
            LDFLAGS="-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}"
            shift 2
            ;;
        -c|--clean)
            CLEAN=true
            shift
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Clean if requested
if [ "$CLEAN" = true ]; then
    print_info "Cleaning ${OUTPUT_DIR}..."
    rm -rf "${OUTPUT_DIR}"
    mkdir -p "${OUTPUT_DIR}"
    print_success "Cleaned"
fi

echo ""
print_info "Kafka Analyzer Builder"
print_info "Version: ${VERSION}"
print_info "Build Time: ${BUILD_TIME}"
print_info "Git Commit: ${GIT_COMMIT}"
echo ""

# Determine platforms to build
PLATFORMS=()
if [ "$PLATFORM" = "all" ]; then
    PLATFORMS=(
        "linux:amd64"
        "linux:arm64"
        "darwin:amd64"
        "darwin:arm64"
        "windows:amd64"
    )
else
    # Parse platform string (e.g., "linux-amd64" -> "linux:amd64")
    IFS='-' read -r os arch <<< "$PLATFORM"
    PLATFORMS=("${os}:${arch}")
fi

# Build platforms
total_builds=${#PLATFORMS[@]}
current_build=0

for platform in "${PLATFORMS[@]}"; do
    IFS=':' read -r os arch <<< "$platform"
    current_build=$((current_build + 1))
    
    echo ""
    print_info "Building ${current_build}/${total_builds}: ${os}/${arch}"
    
    if ! build_binary "$os" "$arch"; then
        print_error "Build failed, stopping..."
        exit 1
    fi
done

echo ""
print_success "All builds completed successfully!"
echo ""
print_info "Binaries are in: ${OUTPUT_DIR}/"
ls -lh "${OUTPUT_DIR}" | tail -n +2 | awk '{print "  " $9 " (" $5 ")"}'
echo ""
