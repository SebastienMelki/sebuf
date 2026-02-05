# External Integrations

**Analysis Date:** 2026-02-05

## APIs & External Services

**Protocol Buffer Registry:**
- Buf Schema Registry (buf.build) - Central dependency management for proto definitions
  - Dependency: `buf.build/bufbuild/protovalidate` - Validation rule definitions
  - Dependency: `buf.build/sebmelki/sebuf` - Custom sebuf annotations published to registry
  - Auth: GitHub token via BUF_TOKEN env var (for Buf CLI)
  - Configuration: `proto/buf.yaml` with deps declaration

**GitHub Integration:**
- GitHub API - Automated changelog generation and release management
  - SDK/Client: Native protobuf via github.com/google/protobuf
  - Auth: GITHUB_TOKEN (environment variable)
  - Used in: `.goreleaser.yml` for releases and changelog generation
  - Used in: GitHub Actions CI/CD workflows (.github/workflows/)

**OpenAPI Specification Parsing:**
- libopenapi (github.com/pb33f/libopenapi v0.33.0) - OpenAPI v3.1 document manipulation
  - No external API calls; pure in-process library
  - Used for building OpenAPI documents from protobuf definitions
  - Implementation: `internal/openapiv3/generator.go`

## Data Storage

**Databases:**
- Not detected - sebuf is a code generator; examples use in-memory maps (e.g., `examples/simple-api/main.go` uses map[string]*models.User)

**File Storage:**
- Local filesystem only
  - Generated files written to protoc output directories
  - Proto files stored in `proto/sebuf/http/` and `proto/sebuf/http/headers.proto`
  - Test data stored in `internal/*/testdata/` directories

**Caching:**
- None detected

## Authentication & Identity

**Auth Provider:**
- GitHub OAuth - For GitHub Actions and goreleaser release management
  - Configuration: GitHub secrets (`GITHUB_TOKEN`, `GPG_FINGERPRINT`, `HOMEBREW_TAP_GITHUB_TOKEN`)
  - Used in: `.github/workflows/` for CI/CD
  - Used in: `.goreleaser.yml` for Homebrew tap and release management

**Buf CLI Auth:**
- Buf token-based authentication for Schema Registry access
  - Environment: `BUF_TOKEN` (CI/CD secret)
  - Used for: `make publish` target to publish annotations

## Monitoring & Observability

**Error Tracking:**
- Not detected - Plugins operate in offline code generation mode

**Logs:**
- Structured logging via stdlib log package (not configured with slog)
- Examples show simple log.Printf() calls (e.g., `examples/simple-api/main.go`)
- No external logging service integration

## CI/CD & Deployment

**Hosting:**
- GitHub (primary repository: github.com/SebastienMelki/sebuf)
- GitHub Container Registry (ghcr.io) for Docker images
- GitHub Releases for binary distribution

**CI Pipeline:**
- GitHub Actions (`.github/workflows/`)
  - ci.yml - Lint, test, and code quality checks on push/PR
  - proto.yml - Proto validation and linting with Buf CLI
  - release.yml - Automated releases with goreleaser
  - GO_VERSION: 1.24
  - BUF_VERSION: latest
  - PROTOC_VERSION: 25.1

**Local CI Testing:**
- act (nektos/act) - Local GitHub Actions testing via Docker
  - Optional tool for `make ci`, `make ci-lint`, `make ci-test`
  - Installed via `make ci-setup`

## Environment Configuration

**Required env vars:**
- `GITHUB_TOKEN` - For GitHub Actions and release automation
- `BUF_TOKEN` - For Buf Schema Registry access (publishing protos)
- `GPG_FINGERPRINT` - For GPG signing of release checksums
- `HOMEBREW_TAP_GITHUB_TOKEN` - For Homebrew tap updates

**Secrets location:**
- GitHub Secrets (.github/secrets/) - Used by workflows
- `.env.act` - Local environment file for act CI testing (example provided)
- `.secrets` - Local secrets file for act CI testing (example provided)

## Webhooks & Callbacks

**Incoming:**
- GitHub Push/PR webhooks trigger CI workflows (`.github/workflows/ci.yml`)
  - Triggers on push to main/develop branches
  - Triggers on pull requests to main branch

**Outgoing:**
- goreleaser release notifications to GitHub Releases
- Homebrew tap commits (automated via goreleaser)
- Docker image pushes to ghcr.io (automated via goreleaser)

## Build & Release Pipeline

**Binary Distribution:**
- goreleaser (.goreleaser.yml) - Multi-platform build and release automation
  - Builds: protoc-gen-go-http, protoc-gen-go-client, protoc-gen-ts-client, protoc-gen-openapiv3
  - Targets: Linux (amd64, arm64, arm/v7), macOS (amd64, arm64), Windows (amd64)
  - Archives: .tar.gz for Unix, .zip for Windows
  - Checksums: SHA256 verification provided

**Package Distribution:**
- Homebrew tap (SebastienMelki/homebrew-sebuf) - macOS/Linux package manager
  - Formula: `sebuf` package
  - Dependencies: protobuf, buf (optional)
  - Maintained via goreleaser automation

- Linux packages (nfpms) - Native package managers
  - Formats: .deb (Debian/Ubuntu), .rpm (Fedora/RHEL), .apk (Alpine)
  - Installed to: /usr/bin/

- Docker images - Container distribution
  - Registry: ghcr.io
  - Images: protoc-gen-go-http, protoc-gen-openapiv3
  - Versions: latest and semver tags
  - Base: Alpine or distroless (standard protoc base)

**Version Management:**
- Semantic versioning (auto-detected from git tags)
- Release notes auto-generated from conventional commits
- Changelog grouped by type: Features, Bug Fixes, Performance, Refactoring, Documentation, Testing, Config, Build, CI/CD

## External Tool Dependencies

**Proto Ecosystem:**
- protoc (Protocol Buffer compiler) v25.1+ - Invokes sebuf plugins via standard plugin interface
- Buf CLI - Modern protobuf build system for dependency management and linting
- protovalidate - Validation framework (buf.build dependency)

**Development Tools:**
- golangci-lint - Code quality (automatic download via `make install`)
- go-test-coverage - Coverage badge generation (automatic download via `make install`)
- goreleaser - Release and packaging automation (invoked in CI)
- act - Local GitHub Actions testing (optional, installed via `make ci-setup`)

---

*Integration audit: 2026-02-05*
