# Technology Stack

**Analysis Date:** 2026-02-05

## Languages

**Primary:**
- Go 1.24.7 - Core plugin development language for all protoc generators

**Secondary:**
- TypeScript - TypeScript HTTP client code generation target
- Protocol Buffers (Proto3) - IDL for service and message definitions
- YAML/JSON - Configuration and output formats for OpenAPI specifications

## Runtime

**Environment:**
- Go runtime 1.24.7 (distributed as compiled binaries)
- Node.js (optional) - For TypeScript client examples and demos

**Package Manager:**
- Go Modules (go.mod/go.sum)
- Lockfile: Present (go.sum with 44 dependencies tracked)

## Frameworks

**Core:**
- Protocol Buffers (google.golang.org/protobuf) v1.36.11 - Foundation for message serialization and code generation
- protogen (google.golang.org/protobuf/compiler/protogen) - Official Go protoc plugin framework used by all generators
- protovalidate (buf.build/gen/go/bufbuild/protovalidate) - Validation framework for request body and header validation

**Code Generation:**
- protoc-gen-go-http - Custom HTTP handler generator built on protogen
- protoc-gen-go-client - Custom Go HTTP client generator
- protoc-gen-ts-client - Custom TypeScript HTTP client generator
- protoc-gen-openapiv3 - Custom OpenAPI v3.1 specification generator

**Testing:**
- Go testing (built-in testing package) - Unit and integration tests
- testify v1.11.1 - Assertion and mocking library for tests

**Build/Dev:**
- Buf CLI (buf.build) - Modern protobuf build system for dependency management and validation
- golangci-lint - Comprehensive Go code quality and linting
- goreleaser - Multi-platform binary building and release automation
- make - Build orchestration (Makefile)

**OpenAPI & Documentation:**
- libopenapi v0.33.0 (github.com/pb33f/libopenapi) - OpenAPI v3.1 document parsing and generation
- ordered-map/v2 v2.3.0 - Preserves field ordering in OpenAPI specifications

## Key Dependencies

**Critical:**
- google.golang.org/protobuf v1.36.11 - Protocol Buffer core library and compiler framework (nonnegotiable)
- github.com/pb33f/libopenapi v0.33.0 - OpenAPI document rendering and schema generation
- buf.build/gen/go/bufbuild/protovalidate v1.36.11 - Validation rule definitions for body/header validation

**Infrastructure:**
- go.yaml.in/yaml/v4 v4.0.0-rc.4 - YAML parsing and generation for OpenAPI output
- go.yaml.in/yaml/v2 v2.4.2 - Indirect YAML dependency
- sigs.k8s.io/yaml v1.6.0 - Kubernetes-style YAML conversion (JSON/YAML interop)
- golang.org/x/sync v0.19.0 - Synchronization primitives for concurrent operations
- github.com/pb33f/jsonpath v0.7.1 - JSON path traversal (libopenapi dependency)

**Utilities:**
- github.com/buger/jsonparser v1.1.1 - Fast JSON parsing (transitive)
- github.com/bahlo/generic-list-go v0.2.0 - Generic list utilities (transitive)

**Development:**
- github.com/stretchr/testify v1.11.1 - Testing assertions and mocks

## Configuration

**Environment:**
- Build configuration via Makefile (`/Users/sebastienmelki/Documents/documents_sebastiens_mac_mini/Workspace/kompani/sebuf/Makefile`)
- Proto dependencies managed via `buf.yaml` (`/Users/sebastienmelki/Documents/documents_sebastiens_mac_mini/Workspace/kompani/sebuf/proto/buf.yaml`)
- Proto code generation via `buf.gen.yaml` (`/Users/sebastienmelki/Documents/documents_sebastiens_mac_mini/Workspace/kompani/sebuf/proto/buf.gen.yaml`)

**Key Configs Required:**
- Go 1.24+ installed
- protoc (Protocol Buffer compiler) v25.1+
- Buf CLI (latest) for proto dependency management
- golangci-lint for code quality checks
- goreleaser for release builds

**Build:**
- `Makefile` - Main build orchestration with targets: build, test, lint, fmt, publish
- `.golangci.yml` - golangci-lint configuration (strict rules, line length 120)
- `.testcoverage.yml` - Test coverage thresholds (file: 70%, package: 80%, total: 85%)
- `.goreleaser.yml` - Multi-platform binary building, Docker images, Homebrew taps, Linux packages

## Platform Requirements

**Development:**
- Go 1.24.7+
- protoc (Protocol Buffer compiler) 25.1+
- Buf CLI (latest)
- golangci-lint (latest)
- macOS, Linux, or Windows with standard build tools

**Production/Distribution:**
- No runtime dependencies (compiled Go binaries)
- Distributed as:
  - Pre-built binaries (Linux, macOS, Windows; amd64, arm64, arm)
  - Docker images (ghcr.io/sebastienmelki/protoc-gen-go-http, protoc-gen-openapiv3)
  - Homebrew tap (macOS/Linux)
  - Linux packages (deb, rpm, apk)

---

*Stack analysis: 2026-02-05*
