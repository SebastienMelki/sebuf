# Makefile for sebuf

# Variables
BIN_DIR := ./bin
CMD_DIR := ./cmd
SCRIPTS_DIR := ./scripts

# Get all cmd directories
CMD_DIRS := $(wildcard $(CMD_DIR)/*)
# Extract binary names from cmd directories
BINARIES := $(notdir $(CMD_DIRS))
# Create full binary paths
BINARY_PATHS := $(addprefix $(BIN_DIR)/, $(BINARIES))

# Default target
.PHONY: all
all: help

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build       - Build all binaries in cmd/* to ./bin/"
	@echo "  clean       - Remove all built binaries"
	@echo "  test        - Run all tests with coverage analysis"
	@echo "  test-fast   - Run all tests without coverage (faster)"
	@echo "  install     - Install all required dependencies"
	@echo "  install-binaries - Install binaries to GOPATH/bin"
	@echo "  proto       - Generate Go code from proto files"
	@echo "  fmt         - Format all Go code"
	@echo "  lint        - Run golangci-lint to check code quality"
	@echo "  lint-fix    - Run golangci-lint with auto-fix"
	@echo "  help        - Show this help message"
	@echo ""
	@echo "Current binaries to build: $(BINARIES)"

# Build all binaries
.PHONY: build
build: $(BINARY_PATHS)

# Pattern rule to build each binary
$(BIN_DIR)/%: $(CMD_DIR)/%/*.go | $(BIN_DIR)
	@echo "Building $*..."
	@go build -o $@ ./$(CMD_DIR)/$*

# Create bin directory
$(BIN_DIR):
	@mkdir -p $(BIN_DIR)

# Clean built binaries
.PHONY: clean
clean:
	@echo "Cleaning built binaries..."
	@rm -rf $(BIN_DIR)

# Run tests with coverage
.PHONY: test
test:
	@echo "Running tests with coverage analysis..."
	@$(SCRIPTS_DIR)/run_tests.sh

# Run tests without coverage (fast)
.PHONY: test-fast
test-fast:
	@echo "Running tests in fast mode..."
	@$(SCRIPTS_DIR)/run_tests.sh --fast

# Install required dependencies
.PHONY: install
install:
	@echo "Installing required dependencies..."
	@echo "Installing golangci-lint..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		echo "✅ golangci-lint installed"; \
	else \
		echo "✅ golangci-lint already installed"; \
	fi
	@echo "Installing go-test-coverage (for coverage badges)..."
	@if ! command -v go-test-coverage >/dev/null 2>&1; then \
		go install github.com/vladopajic/go-test-coverage/v2@latest; \
		echo "✅ go-test-coverage installed"; \
	else \
		echo "✅ go-test-coverage already installed"; \
	fi
	@echo "All dependencies installed!"

# Install binaries to GOPATH/bin
.PHONY: install-binaries
install-binaries:
	@echo "Installing binaries to GOPATH/bin..."
	@for binary in $(BINARIES); do \
		echo "Installing $$binary..."; \
		go install ./$(CMD_DIR)/$$binary; \
	done

# Generate proto files
.PHONY: proto
proto:
	@echo "Generating Go code from proto files..."
	@protoc --go_out=. --go_opt=module=github.com/SebastienMelki/sebuf \
		--go_opt=Msebuf/http/annotations.proto=github.com/SebastienMelki/sebuf/internal/httpgen \
		--proto_path=. \
		sebuf/http/annotations.proto

# Format Go code
.PHONY: fmt
fmt:
	@echo "Formatting Go code..."
	@go fmt ./...

# Run linter
.PHONY: lint
lint:
	@echo "Running golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

# Run linter with auto-fix
.PHONY: lint-fix
lint-fix:
	@echo "Running golangci-lint with auto-fix..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --fix; \
	else \
		echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

# Rebuild (clean + build)
.PHONY: rebuild
rebuild: clean build

# Show current binary targets
.PHONY: list-binaries
list-binaries:
	@echo "Binaries that will be built:"
	@for binary in $(BINARIES); do \
		echo "  $(BIN_DIR)/$$binary"; \
	done

# Check if scripts are executable
.PHONY: check-scripts
check-scripts:
	@if [ ! -x "$(SCRIPTS_DIR)/run_tests.sh" ]; then \
		echo "Making run_tests.sh executable..."; \
		chmod +x $(SCRIPTS_DIR)/run_tests.sh; \
	fi

# Make run_tests.sh executable and run tests
.PHONY: test-setup
test-setup: check-scripts test

.PHONY: test-fast-setup  
test-fast-setup: check-scripts test-fast