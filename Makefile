# log2json Makefile

# Build configuration
BINARY_NAME := log2json
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//' || echo "dev")
BUILD_DIR := build
GO := go

# Build flags
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: all build clean test coverage lint vet check release version run install help

# Default target
all: build

# Build the binary
build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
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

# Run linters
lint:
	@echo "Running linters..."
	golangci-lint run ./...

# Run go vet
vet:
	@echo "Running go vet..."
	$(GO) vet ./...

# Run all checks (lint + vet + test) — use before pushing
check: lint vet test
	@echo "All checks passed."

# Create a new release (usage: make release V=x.y.z)
release:
ifndef V
	$(error Usage: make release V=x.y.z)
endif
	@echo "Tagging release v$(V)..."
	git tag -a "v$(V)" -m "Release v$(V)"
	@echo "Push with: git push origin v$(V)"

# Show current version
version:
	@echo $(VERSION)

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
	@echo "  lint      - Run golangci-lint"
	@echo "  vet       - Run go vet"
	@echo "  check     - Run lint + vet + test (pre-push validation)"
	@echo "  release   - Create a release tag (usage: make release V=x.y.z)"
	@echo "  version   - Show current version"
	@echo "  run       - Build and run with sample input"
	@echo "  install   - Install to GOPATH/bin"
	@echo "  help      - Show this help"
