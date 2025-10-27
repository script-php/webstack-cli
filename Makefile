# WebStack CLI - Makefile

# Variables
BINARY_NAME=webstack
VERSION?=v1.0.0
BUILD_DIR=build
LDFLAGS=-ldflags="-s -w -X webstack-cli/cmd.Version=$(VERSION) -X webstack-cli/cmd.BuildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ) -X webstack-cli/cmd.GitCommit=$(shell git rev-parse --short HEAD)"

# Default target
.PHONY: all
all: clean build

# Clean build directory
.PHONY: clean
clean:
	@echo "🧹 Cleaning build directory..."
	@rm -rf $(BUILD_DIR)
	@mkdir -p $(BUILD_DIR)

# Build for current platform
.PHONY: build
build:
	@echo "🔨 Building WebStack CLI..."
	@go build $(LDFLAGS) -o $(BINARY_NAME) .

# Build for all platforms
.PHONY: build-all
build-all: clean
	@echo "🔨 Building for all platforms..."
	
	# Linux
	@echo "Building for Linux AMD64..."
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	
	@echo "Building for Linux ARM64..."
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 .
	
	@echo "Building for Linux ARM..."
	@GOOS=linux GOARCH=arm go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm .
	
	# macOS
	@echo "Building for macOS AMD64..."
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	
	@echo "Building for macOS ARM64..."
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .
	
	# Windows
	@echo "Building for Windows AMD64..."
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	
	# Generate checksums
	@echo "📋 Generating checksums..."
	@cd $(BUILD_DIR) && sha256sum * > checksums.txt
	
	# Create templates archive
	@echo "📦 Creating templates archive..."
	@tar -czf $(BUILD_DIR)/$(BINARY_NAME)-templates.tar.gz templates/
	
	@echo "✅ Build completed! Artifacts in $(BUILD_DIR)/"

# Install locally
.PHONY: install
install: build
	@echo "📦 Installing WebStack CLI..."
	@sudo cp $(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@echo "✅ Installed to /usr/local/bin/$(BINARY_NAME)"

# Uninstall
.PHONY: uninstall
uninstall:
	@echo "🗑️ Uninstalling WebStack CLI..."
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "✅ Uninstalled from /usr/local/bin/$(BINARY_NAME)"

# Run tests
.PHONY: test
test:
	@echo "🧪 Running tests..."
	@go test -v ./...

# Format code
.PHONY: fmt
fmt:
	@echo "🎨 Formatting code..."
	@go fmt ./...

# Lint code
.PHONY: lint
lint:
	@echo "🔍 Linting code..."
	@golangci-lint run

# Download dependencies
.PHONY: deps
deps:
	@echo "📥 Downloading dependencies..."
	@go mod download
	@go mod tidy

# Create release
.PHONY: release
release: build-all
	@echo "🚀 Creating release $(VERSION)..."
	@git tag -a $(VERSION) -m "Release $(VERSION)"
	@git push origin $(VERSION)
	@echo "✅ Release $(VERSION) created!"

# Docker build
.PHONY: docker-build
docker-build:
	@echo "🐳 Building Docker image..."
	@docker build -t $(BINARY_NAME):$(VERSION) .
	@docker tag $(BINARY_NAME):$(VERSION) $(BINARY_NAME):latest
	@echo "✅ Docker image built: $(BINARY_NAME):$(VERSION)"

# Docker run
.PHONY: docker-run
docker-run:
	@echo "🐳 Running Docker container..."
	@docker run --rm -it $(BINARY_NAME):latest

# Development server (for testing)
.PHONY: dev
dev: build
	@echo "🔧 Starting development mode..."
	@./$(BINARY_NAME) --help

# Package for different distributions
.PHONY: package
package: build-all
	@echo "📦 Creating distribution packages..."
	
	# Create DEB package structure
	@mkdir -p $(BUILD_DIR)/deb/usr/local/bin
	@mkdir -p $(BUILD_DIR)/deb/etc/webstack/templates
	@mkdir -p $(BUILD_DIR)/deb/DEBIAN
	
	@cp $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(BUILD_DIR)/deb/usr/local/bin/$(BINARY_NAME)
	@cp -r templates/* $(BUILD_DIR)/deb/etc/webstack/templates/
	
	@echo "Package: $(BINARY_NAME)" > $(BUILD_DIR)/deb/DEBIAN/control
	@echo "Version: $(VERSION)" >> $(BUILD_DIR)/deb/DEBIAN/control
	@echo "Section: utils" >> $(BUILD_DIR)/deb/DEBIAN/control
	@echo "Priority: optional" >> $(BUILD_DIR)/deb/DEBIAN/control
	@echo "Architecture: amd64" >> $(BUILD_DIR)/deb/DEBIAN/control
	@echo "Maintainer: WebStack CLI Team" >> $(BUILD_DIR)/deb/DEBIAN/control
	@echo "Description: Complete web stack management CLI tool" >> $(BUILD_DIR)/deb/DEBIAN/control
	
	@dpkg-deb --build $(BUILD_DIR)/deb $(BUILD_DIR)/$(BINARY_NAME)_$(VERSION)_amd64.deb
	
	@echo "✅ Packages created in $(BUILD_DIR)/"

# Show help
.PHONY: help
help:
	@echo "WebStack CLI - Build System"
	@echo "=========================="
	@echo ""
	@echo "Available targets:"
	@echo "  build        Build for current platform"
	@echo "  build-all    Build for all platforms"
	@echo "  install      Install locally"
	@echo "  uninstall    Uninstall from system"
	@echo "  test         Run tests"
	@echo "  fmt          Format code"
	@echo "  lint         Lint code"
	@echo "  deps         Download dependencies"
	@echo "  release      Create and push release tag"
	@echo "  docker-build Build Docker image"
	@echo "  docker-run   Run Docker container"
	@echo "  package      Create distribution packages"
	@echo "  clean        Clean build directory"
	@echo "  dev          Development mode"
	@echo "  help         Show this help"
	@echo ""
	@echo "Examples:"
	@echo "  make build VERSION=v1.2.3"
	@echo "  make release VERSION=v1.2.3"
	@echo "  make install"