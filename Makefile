.PHONY: all build clean dev staging prod linux-amd64 linux-arm64 darwin-amd64 darwin-arm64 windows-amd64 help release

# Variables
VERSION ?= 1.3.1
BINARY_NAME = kmap
OUTPUT_DIR = bin
BUILD_TIME = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS = -s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)

# Colors
BLUE = \033[0;34m
GREEN = \033[0;32m
YELLOW = \033[1;33m
NC = \033[0m # No Color

# Default target
all: clean build

# Help target
help:
	@echo "$(BLUE)kmap - Kafka Cluster Inspector$(NC)"
	@echo ""
	@echo "$(GREEN)Usage:$(NC)"
	@echo "  make [target]"
	@echo ""
	@echo "$(GREEN)Targets:$(NC)"
	@echo "  $(YELLOW)all$(NC)              - Clean and build for all platforms"
	@echo "  $(YELLOW)build$(NC)            - Build for all platforms"
	@echo "  $(YELLOW)clean$(NC)            - Remove all binaries from bin/"
	@echo "  $(YELLOW)linux-amd64$(NC)      - Build for Linux AMD64"
	@echo "  $(YELLOW)linux-arm64$(NC)      - Build for Linux ARM64"
	@echo "  $(YELLOW)darwin-amd64$(NC)     - Build for macOS Intel"
	@echo "  $(YELLOW)darwin-arm64$(NC)     - Build for macOS Apple Silicon"
	@echo "  $(YELLOW)windows-amd64$(NC)    - Build for Windows AMD64"
	@echo "  $(YELLOW)test$(NC)             - Run tests"
	@echo "  $(YELLOW)deps$(NC)             - Download dependencies"
	@echo "  $(YELLOW)local$(NC)            - Quick build for current platform"
	@echo "  $(YELLOW)release$(NC)          - Create a new release (build + tag)"
	@echo "  $(YELLOW)help$(NC)             - Show this help message"
	@echo ""
	@echo "$(GREEN)Examples:$(NC)"
	@echo "  make build                   # Build binaries for all platforms"
	@echo "  make linux-amd64             # Build Linux AMD64 binary"
	@echo "  make VERSION=2.0.0 build     # Build with custom version"
	@echo ""

# Clean build directory
clean:
	@echo "$(BLUE)Cleaning bin directory...$(NC)"
	@rm -rf $(OUTPUT_DIR)
	@mkdir -p $(OUTPUT_DIR)
	@echo "$(GREEN)✓ Cleaned$(NC)"

# Download dependencies
deps:
	@echo "$(BLUE)Downloading dependencies...$(NC)"
	@go mod download
	@echo "$(GREEN)✓ Dependencies downloaded$(NC)"

# Run tests
test:
	@echo "$(BLUE)Running tests...$(NC)"
	@go test -v ./...
	@echo "$(GREEN)✓ Tests passed$(NC)"

# Build all platforms
build: deps
	@echo "$(BLUE)Building all platforms...$(NC)"
	@chmod +x build.sh
	@./build.sh

# Build for specific platforms
linux-amd64: deps
	@echo "$(BLUE)Building for Linux AMD64...$(NC)"
	@chmod +x build.sh
	@./build.sh --platform linux-amd64

linux-arm64: deps
	@echo "$(BLUE)Building for Linux ARM64...$(NC)"
	@chmod +x build.sh
	@./build.sh --platform linux-arm64

darwin-amd64: deps
	@echo "$(BLUE)Building for macOS Intel...$(NC)"
	@chmod +x build.sh
	@./build.sh --platform darwin-amd64

darwin-arm64: deps
	@echo "$(BLUE)Building for macOS Apple Silicon...$(NC)"
	@chmod +x build.sh
	@./build.sh --platform darwin-arm64

windows-amd64: deps
	@echo "$(BLUE)Building for Windows AMD64...$(NC)"
	@chmod +x build.sh
	@./build.sh --platform windows-amd64

# Quick local build (current platform only)
local:
	@echo "$(BLUE)Building for local platform...$(NC)"
	@go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME)
	@echo "$(GREEN)✓ Built $(BINARY_NAME)$(NC)"
# Release - build all platforms and create git tag
release: clean build
	@echo "$(BLUE)Creating release v$(VERSION)...$(NC)"
	@echo "$(YELLOW)Binaries built in $(OUTPUT_DIR)/$(NC)"
	@ls -lh $(OUTPUT_DIR)/
	@echo ""
	@echo "$(GREEN)Release v$(VERSION) ready!$(NC)"
	@echo ""
	@echo "$(YELLOW)Next steps:$(NC)"
	@echo "  1. Review the binaries in $(OUTPUT_DIR)/"
	@echo "  2. Test the binaries"
	@echo "  3. Commit release notes: git add release-notes-v$(VERSION).md CHANGELOG.md Makefile"
	@echo "  4. Commit: git commit -m 'Release v$(VERSION)'"
	@echo "  5. Tag: git tag -a v$(VERSION) -m 'Release v$(VERSION)'"
	@echo "  6. Push: git push && git push --tags"
	@echo ""