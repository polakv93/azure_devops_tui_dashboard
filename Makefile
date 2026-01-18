.PHONY: build run clean test fmt lint install

# Build variables
BINARY_NAME=azdo-tui
BUILD_DIR=./bin
CMD_DIR=./cmd/azdo-tui
VERSION?=dev
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

# Default target
all: build

# Build the application
build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)

# Build for multiple platforms
build-all: build-linux build-darwin build-windows

build-linux:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)

build-darwin:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)

build-windows:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)

# Run the application
run: build
	$(BUILD_DIR)/$(BINARY_NAME) --config configs/config.yaml

# Run with example config
run-example: build
	$(BUILD_DIR)/$(BINARY_NAME) --config configs/config.example.yaml

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	go clean

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Format code
fmt:
	go fmt ./...

# Lint code (requires golangci-lint)
lint:
	golangci-lint run

# Install to GOPATH/bin
install:
	go install $(LDFLAGS) $(CMD_DIR)

# Update dependencies
deps:
	go mod tidy
	go mod download

# Show help
help:
	@echo "Azure DevOps TUI Dashboard - Makefile targets:"
	@echo ""
	@echo "  build         Build the application"
	@echo "  build-all     Build for Linux, macOS, and Windows"
	@echo "  run           Build and run with configs/config.yaml"
	@echo "  run-example   Build and run with example config"
	@echo "  clean         Remove build artifacts"
	@echo "  test          Run tests"
	@echo "  test-coverage Run tests with coverage report"
	@echo "  fmt           Format code"
	@echo "  lint          Run linter"
	@echo "  install       Install to GOPATH/bin"
	@echo "  deps          Update dependencies"
	@echo "  help          Show this help message"
