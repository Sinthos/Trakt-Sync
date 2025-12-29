.PHONY: build build-linux build-linux-arm test lint clean install help

# Variables
BINARY_NAME=trakt-sync
VERSION?=dev
BUILD_DIR=bin
GO=go
GOFLAGS=-ldflags="-X main.Version=$(VERSION)"

# Default target
all: build

# Build for current platform
build:
	@echo "Building $(BINARY_NAME) for current platform..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/trakt-sync

# Build for Linux AMD64
build-linux:
	@echo "Building $(BINARY_NAME) for linux/amd64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/trakt-sync

# Build for Linux ARM64
build-linux-arm:
	@echo "Building $(BINARY_NAME) for linux/arm64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/trakt-sync

# Build for all platforms
build-all: build-linux build-linux-arm
	@echo "Built for all platforms"

# Run tests
test:
	@echo "Running tests..."
	$(GO) test -v -race -coverprofile=coverage.out ./...

# Run linter
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out

# Install to /usr/local/bin
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@echo "Installed successfully!"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy

# Run the binary
run: build
	@$(BUILD_DIR)/$(BINARY_NAME)

# Show help
help:
	@echo "Available targets:"
	@echo "  build           - Build for current platform"
	@echo "  build-linux     - Build for linux/amd64"
	@echo "  build-linux-arm - Build for linux/arm64"
	@echo "  build-all       - Build for all platforms"
	@echo "  test            - Run tests"
	@echo "  lint            - Run linter"
	@echo "  clean           - Clean build artifacts"
	@echo "  install         - Install to /usr/local/bin"
	@echo "  deps            - Download and tidy dependencies"
	@echo "  run             - Build and run"
	@echo "  help            - Show this help"
