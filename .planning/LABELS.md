# GitHub Labels System

Every issue, PR, and release MUST use this labeling system. Apply labels consistently.

## Type Labels

| Label | Color | Use When |
|-------|-------|----------|
| `bug` | red | Something isn't working |
| `enhancement` | cyan | New feature or improvement |
| `refactor` | yellow | Code restructuring without behavior change |
| `documentation` | blue | Docs, examples, README |
| `breaking-change` | red | Changes that break backward compatibility (CAPITAL SIN) |
| `consistency` | orange | Cross-generator consistency issue (CAPITAL SIN) |

## Component Labels (prefix: `gen/`)

| Label | Color | Component |
|-------|-------|-----------|
| `gen/go-http` | Go blue | protoc-gen-go-http (HTTP server) |
| `gen/go-client` | Go blue | protoc-gen-go-client (Go HTTP client) |
| `gen/ts-client` | TS blue | protoc-gen-ts-client (TypeScript HTTP client) |
| `gen/swift-client` | Swift orange | protoc-gen-swift-client (Swift HTTP client) |
| `gen/kt-client` | Kotlin purple | protoc-gen-kt-client (Kotlin HTTP client) |
| `gen/py-client` | Python blue | protoc-gen-py-client (Python HTTP client) |
| `gen/openapiv3` | green | protoc-gen-openapiv3 (OpenAPI spec) |
| `gen/annotations` | yellow | Shared annotation definitions (proto/sebuf/http/) |

## Category Labels

| Label | Color | Use When |
|-------|-------|----------|
| `json-mapping` | dark blue | JSON serialization mapping features |
| `foundation` | light blue | Infrastructure, refactoring, bug fixes |
| `polish` | lighter blue | Documentation, examples, test coverage |
| `language-client` | purple | New language client generator |

## Milestone Labels

| Label | Color | Use When |
|-------|-------|----------|
| `1.0-blocker` | red | Must be resolved before 1.0 release |
| `v1.0` | green | Targeted for v1.0 |
| `v2.0` | teal | Targeted for v2.0 |

## Labeling Rules

1. **Every issue/PR gets at least**: one type label + one component label + one milestone label
2. **JSON mapping issues**: always get `json-mapping` + `gen/annotations` + all affected `gen/*` labels
3. **Cross-generator issues**: always get `consistency` label
4. **PRs**: mirror the labels of the issue they fix
5. **Breaking changes**: ALWAYS get `breaking-change` — these require explicit approval before merge

## Commit Message Prefixes

| Prefix | When |
|--------|------|
| `feat(gen):` | New feature in a generator |
| `fix(gen):` | Bug fix in a generator |
| `refactor(gen):` | Refactoring a generator |
| `docs:` | Documentation only |
| `test:` | Test only |
| `chore:` | Build, CI, deps |

Examples:
- `feat(go-http): add nullable primitive support`
- `fix(go-client): conditional net/url import`
- `refactor(annotations): extract shared parsing package`
- `feat(swift-client): initial Swift HTTP client generator`
- `docs: add JSON mapping examples`
