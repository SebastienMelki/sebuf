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
	@echo "  build-rust  - Build all Rust plugins"
	@echo "  build-all   - Build both Go and Rust binaries"
	@echo "  clean       - Remove all built binaries"
	@echo "  clean-rust  - Clean Rust build artifacts"
	@echo "  test        - Run Go tests with coverage analysis"
	@echo "  test-fast   - Run Go tests without coverage (faster)"
	@echo "  test-rust   - Run comprehensive Rust tests"
	@echo "  test-rust-fast - Run Rust tests without coverage"
	@echo "  test-rust-update-golden - Update Rust golden test files"
	@echo "  test-all    - Run all tests (Go + Rust)"
	@echo "  test-all-fast - Run all tests fast (Go + Rust)"
	@echo "  install     - Install all required dependencies"
	@echo "  install-binaries - Install binaries to GOPATH/bin"
	@echo "  install-rust - Install Rust plugins to PATH"
	@echo "  proto       - Generate Go code from proto files"
	@echo "  publish     - Publish annotations to Buf Schema Registry"
	@echo "  fmt         - Format all Go code"
	@echo "  fmt-rust    - Format all Rust code"
	@echo "  lint        - Run golangci-lint to check code quality"
	@echo "  lint-fix    - Run golangci-lint with auto-fix"
	@echo "  lint-rust   - Run clippy to check Rust code quality"
	@echo ""
	@echo "CI/CD targets:"
	@echo "  ci          - Run CI pipeline locally with act"
	@echo "  ci-lint     - Run lint workflow locally"
	@echo "  ci-test     - Run test workflow locally"
	@echo "  ci-list     - List available GitHub Actions workflows"
	@echo "  ci-setup    - Install act for local CI testing"
	@echo ""
	@echo "  help        - Show this help message"
	@echo ""
	@echo "Current Go binaries to build: $(BINARIES)"
	@echo "Rust plugins: protoc-gen-rust-oneof-helper, protoc-gen-rust-http, protoc-gen-rust-openapiv3"

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
		--go_opt=Msebuf/http/annotations.proto=github.com/SebastienMelki/sebuf/http \
		--go_opt=Msebuf/http/headers.proto=github.com/SebastienMelki/sebuf/http \
		--go_opt=Msebuf/http/errors.proto=github.com/SebastienMelki/sebuf/http \
		--proto_path=. \
		proto/sebuf/http/annotations.proto \
		proto/sebuf/http/headers.proto \
		proto/sebuf/http/errors.proto

# Publish annotations to Buf Schema Registry
.PHONY: publish
publish:
	@echo "Publishing annotations to Buf Schema Registry..."
	@cd proto && buf push
	@echo "✅ Published to buf.build/sebmelki/sebuf"
	@echo "Other projects can now use: deps: [buf.build/sebmelki/sebuf]"

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

# Rust targets
.PHONY: build-rust
build-rust:
	@echo "Building Rust plugins..."
	@cargo build --release
	@mkdir -p $(BIN_DIR)
	@cp target/release/protoc-gen-rust-oneof-helper $(BIN_DIR)/
	@cp target/release/protoc-gen-rust-http $(BIN_DIR)/
	@cp target/release/protoc-gen-rust-openapiv3 $(BIN_DIR)/
	@echo "✅ Rust plugins built successfully"

# Build all (Go + Rust)
.PHONY: build-all
build-all: build build-rust

# Clean Rust artifacts
.PHONY: clean-rust
clean-rust:
	@echo "Cleaning Rust build artifacts..."
	@cargo clean
	@rm -f $(BIN_DIR)/protoc-gen-rust-*

# Test Rust code
.PHONY: test-rust
test-rust:
	@echo "Running comprehensive Rust tests..."
	@$(SCRIPTS_DIR)/test_rust.sh

# Test Rust code (fast mode)
.PHONY: test-rust-fast
test-rust-fast:
	@echo "Running Rust tests (fast mode)..."
	@$(SCRIPTS_DIR)/test_rust.sh --fast

# Update Rust golden files
.PHONY: test-rust-update-golden
test-rust-update-golden:
	@echo "Updating Rust golden files..."
	@$(SCRIPTS_DIR)/test_rust.sh --update-golden

# Run all tests (Go + Rust)
.PHONY: test-all
test-all: test test-rust

# Run all tests fast (Go + Rust)
.PHONY: test-all-fast
test-all-fast: test-fast test-rust-fast

# Format Rust code
.PHONY: fmt-rust
fmt-rust:
	@echo "Formatting Rust code..."
	@cargo fmt --all

# Lint Rust code
.PHONY: lint-rust
lint-rust:
	@echo "Running clippy..."
	@cargo clippy --all -- -D warnings

# Install Rust plugins
.PHONY: install-rust
install-rust: build-rust
	@echo "Installing Rust plugins..."
	@cargo install --path rust/protoc-gen-rust-oneof-helper
	@cargo install --path rust/protoc-gen-rust-http
	@cargo install --path rust/protoc-gen-rust-openapiv3
	@echo "✅ Rust plugins installed to cargo bin directory"

# Rebuild (clean + build)
.PHONY: rebuild
rebuild: clean build

# Rebuild all
.PHONY: rebuild-all
rebuild-all: clean clean-rust build-all

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

# CI/CD targets using nektos/act for local testing

# Install act for local GitHub Actions testing
.PHONY: ci-setup
ci-setup:
	@echo "Installing act for local CI testing..."
	@if ! command -v act >/dev/null 2>&1; then \
		if [ "$(shell uname)" = "Darwin" ]; then \
			echo "Installing act via Homebrew..."; \
			brew install act; \
		elif [ "$(shell uname)" = "Linux" ]; then \
			echo "Installing act via script..."; \
			curl -s https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash; \
		else \
			echo "Please install act manually from: https://github.com/nektos/act"; \
			exit 1; \
		fi; \
		echo "✅ act installed successfully"; \
	else \
		echo "✅ act is already installed"; \
	fi
	@echo ""
	@echo "Next steps:"
	@echo "1. Copy .env.act.example to .env.act and customize if needed"
	@echo "2. Copy .secrets.example to .secrets and add any required secrets"
	@echo "3. Run 'make ci' to test the CI pipeline locally"

# Run full CI pipeline locally
.PHONY: ci
ci:
	@echo "Running CI pipeline locally with act..."
	@if ! command -v act >/dev/null 2>&1; then \
		echo "act not found. Run 'make ci-setup' to install it."; \
		exit 1; \
	fi
	@act push --workflows .github/workflows/ci.yml

# Run only lint job locally
.PHONY: ci-lint
ci-lint:
	@echo "Running lint workflow locally..."
	@if ! command -v act >/dev/null 2>&1; then \
		echo "act not found. Run 'make ci-setup' to install it."; \
		exit 1; \
	fi
	@act push --workflows .github/workflows/ci.yml --job lint

# Run only test job locally
.PHONY: ci-test
ci-test:
	@echo "Running test workflow locally..."
	@if ! command -v act >/dev/null 2>&1; then \
		echo "act not found. Run 'make ci-setup' to install it."; \
		exit 1; \
	fi
	@act push --workflows .github/workflows/ci.yml --job test

# Run proto validation workflow locally
.PHONY: ci-proto
ci-proto:
	@echo "Running proto validation workflow locally..."
	@if ! command -v act >/dev/null 2>&1; then \
		echo "act not found. Run 'make ci-setup' to install it."; \
		exit 1; \
	fi
	@act push --workflows .github/workflows/proto.yml

# List available workflows and jobs
.PHONY: ci-list
ci-list:
	@echo "Available GitHub Actions workflows:"
	@if ! command -v act >/dev/null 2>&1; then \
		echo "act not found. Run 'make ci-setup' to install it."; \
		exit 1; \
	fi
	@act list --workflows .github/workflows/

# Run release workflow locally (dry run)
.PHONY: ci-release-dry
ci-release-dry:
	@echo "Running release workflow in dry-run mode..."
	@if ! command -v act >/dev/null 2>&1; then \
		echo "act not found. Run 'make ci-setup' to install it."; \
		exit 1; \
	fi
	@act push --workflows .github/workflows/release.yml --dryrun

# Clean act artifacts
.PHONY: ci-clean
ci-clean:
	@echo "Cleaning act artifacts..."
	@rm -rf .act
	@docker container prune -f
	@docker image prune -f

# Validate all workflows
.PHONY: ci-validate
ci-validate:
	@echo "Validating GitHub Actions workflows..."
	@for workflow in .github/workflows/*.yml; do \
		echo "Validating $$workflow..."; \
		if command -v actionlint >/dev/null 2>&1; then \
			actionlint $$workflow; \
		else \
			echo "actionlint not found. Install with: go install github.com/rhysd/actionlint/cmd/actionlint@latest"; \
		fi; \
	done