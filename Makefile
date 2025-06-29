# MCP Manager CLI Makefile

# Variables
BINARY_NAME=mcp-manager
MAIN_PACKAGE=./main.go
BUILD_DIR=./build
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME?=$(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Go variables
GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)
GO_VERSION?=$(shell go version | cut -d' ' -f3)

# Build flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME) -s -w"

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[0;33m
BLUE=\033[0;34m
NC=\033[0m # No Color

.PHONY: help deps build test clean install install-system cross-compile docker-build fmt vet lint

# Default target
all: deps build

# Help target
help: ## Show this help message
	@echo "$(BLUE)MCP Manager CLI - Available Commands:$(NC)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "$(GREEN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)Build Information:$(NC)"
	@echo "  Version: $(VERSION)"
	@echo "  Go Version: $(GO_VERSION)"
	@echo "  OS/Arch: $(GOOS)/$(GOARCH)"

# Install dependencies
deps: ## Install Go dependencies
	@echo "$(BLUE)Installing dependencies...$(NC)"
	@go mod download
	@go mod tidy
	@echo "$(GREEN)Dependencies installed successfully!$(NC)"

# Build the binary
build: deps ## Build the binary
	@echo "$(BLUE)Building $(BINARY_NAME)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "$(GREEN)Build completed: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

# Build and install to current directory
install: build ## Install dependencies and build binary to current directory
	@echo "$(BLUE)Installing $(BINARY_NAME) to current directory...$(NC)"
	@cp $(BUILD_DIR)/$(BINARY_NAME) ./$(BINARY_NAME)
	@chmod +x ./$(BINARY_NAME)
	@echo "$(GREEN)Installation completed: ./$(BINARY_NAME)$(NC)"
	@echo "$(YELLOW)Run './$(BINARY_NAME) --help' to get started$(NC)"

# Install to system PATH
install-system: build ## Install binary to system PATH (/usr/local/bin)
	@echo "$(BLUE)Installing $(BINARY_NAME) to system PATH...$(NC)"
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@echo "$(GREEN)System installation completed!$(NC)"
	@echo "$(YELLOW)Run '$(BINARY_NAME) --help' from anywhere$(NC)"

# Run tests
test: ## Run tests
	@echo "$(BLUE)Running tests...$(NC)"
	@go test -v ./...
	@echo "$(GREEN)Tests completed!$(NC)"

# Run tests with coverage
test-coverage: ## Run tests with coverage report
	@echo "$(BLUE)Running tests with coverage...$(NC)"
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

# Format code
fmt: ## Format Go code
	@echo "$(BLUE)Formatting code...$(NC)"
	@go fmt ./...
	@echo "$(GREEN)Code formatted!$(NC)"

# Vet code
vet: ## Run go vet
	@echo "$(BLUE)Running go vet...$(NC)"
	@go vet ./...
	@echo "$(GREEN)Vet completed!$(NC)"

# Lint code (requires golangci-lint)
lint: ## Run golangci-lint
	@echo "$(BLUE)Running linter...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
		echo "$(GREEN)Linting completed!$(NC)"; \
	else \
		echo "$(YELLOW)golangci-lint not found. Install it with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest$(NC)"; \
	fi

# Clean build artifacts
clean: ## Clean build artifacts
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	@rm -rf $(BUILD_DIR)
	@rm -f ./$(BINARY_NAME)
	@rm -f coverage.out coverage.html
	@echo "$(GREEN)Clean completed!$(NC)"

# Cross-compile for multiple platforms
cross-compile: deps ## Build for multiple platforms
	@echo "$(BLUE)Cross-compiling for multiple platforms...$(NC)"
	@mkdir -p $(BUILD_DIR)
	
	@echo "Building for Linux AMD64..."
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PACKAGE)
	
	@echo "Building for Linux ARM64..."
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PACKAGE)
	
	@echo "Building for macOS AMD64..."
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PACKAGE)
	
	@echo "Building for macOS ARM64..."
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PACKAGE)
	
	@echo "Building for Windows AMD64..."
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PACKAGE)
	
	@echo "$(GREEN)Cross-compilation completed!$(NC)"
	@echo "$(YELLOW)Binaries available in $(BUILD_DIR)/$(NC)"
	@ls -la $(BUILD_DIR)/

# Create release archives
release: cross-compile ## Create release archives for all platforms
	@echo "$(BLUE)Creating release archives...$(NC)"
	@mkdir -p $(BUILD_DIR)/releases
	
	@cd $(BUILD_DIR) && tar -czf releases/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64
	@cd $(BUILD_DIR) && tar -czf releases/$(BINARY_NAME)-$(VERSION)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64
	@cd $(BUILD_DIR) && tar -czf releases/$(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64
	@cd $(BUILD_DIR) && tar -czf releases/$(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64
	@cd $(BUILD_DIR) && zip -q releases/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe
	
	@echo "$(GREEN)Release archives created in $(BUILD_DIR)/releases/$(NC)"
	@ls -la $(BUILD_DIR)/releases/

# Run the binary (for testing)
run: build ## Build and run the binary
	@echo "$(BLUE)Running $(BINARY_NAME)...$(NC)"
	@$(BUILD_DIR)/$(BINARY_NAME) --help

# Development setup
dev-setup: ## Setup development environment
	@echo "$(BLUE)Setting up development environment...$(NC)"
	@go mod download
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "$(GREEN)Development environment setup completed!$(NC)"

# Check if required tools are installed
check-tools: ## Check if required development tools are installed
	@echo "$(BLUE)Checking required tools...$(NC)"
	@command -v go >/dev/null 2>&1 || { echo "$(RED)Go is not installed$(NC)"; exit 1; }
	@command -v git >/dev/null 2>&1 || { echo "$(RED)Git is not installed$(NC)"; exit 1; }
	@echo "$(GREEN)All required tools are installed!$(NC)"

# Show version information
version: ## Show version information
	@echo "$(BLUE)Version Information:$(NC)"
	@echo "  Version: $(VERSION)"
	@echo "  Commit: $(COMMIT)"
	@echo "  Build Time: $(BUILD_TIME)"
	@echo "  Go Version: $(GO_VERSION)"
	@echo "  OS/Arch: $(GOOS)/$(GOARCH)"

# Quick development cycle
dev: fmt vet test build ## Run development cycle (format, vet, test, build)
	@echo "$(GREEN)Development cycle completed!$(NC)"

# CI/CD pipeline simulation
ci: check-tools deps fmt vet lint test build ## Run CI/CD pipeline (check, deps, format, vet, lint, test, build)
	@echo "$(GREEN)CI pipeline completed successfully!$(NC)"
