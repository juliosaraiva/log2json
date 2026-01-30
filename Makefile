# log2json Makefile

# Build configuration
BINARY_NAME := log2json
VERSION := 0.1.0
BUILD_DIR := build
GO := go

# Build flags
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: all build clean test coverage run install help

# Default target
all: build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/log2json

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

# Run tests
test:
	@echo "Running tests..."
	$(GO) test -v -race ./...

# Run tests with coverage report
coverage:
	@echo "Running tests with coverage..."
	$(GO) test -coverprofile=coverage.out -race ./...
	$(GO) tool cover -func=coverage.out
	@echo ""
	@echo "Generating HTML report: coverage.html"
	$(GO) tool cover -html=coverage.out -o coverage.html

# Run with sample input
run: build
	@echo "Running with sample syslog input..."
	@echo 'Jan 15 10:30:45 myhost sshd[1234]: Accepted password for user from 192.168.1.1' | ./$(BUILD_DIR)/$(BINARY_NAME)

# Install to GOPATH/bin
install:
	@echo "Installing..."
	$(GO) install $(LDFLAGS) ./cmd/log2json

# Show help
help:
	@echo "Available targets:"
	@echo "  build     - Build the binary"
	@echo "  clean     - Remove build artifacts"
	@echo "  test      - Run tests with race detection"
	@echo "  coverage  - Run tests with coverage report"
	@echo "  run       - Build and run with sample input"
	@echo "  install   - Install to GOPATH/bin"
	@echo "  help      - Show this help"
