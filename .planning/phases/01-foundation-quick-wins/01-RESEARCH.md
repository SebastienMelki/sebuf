# Phase 01: Foundation - Quick Wins - Research

**Researched:** 2026-02-05
**Domain:** Protoc plugin codegen bug fixes and issue housekeeping
**Confidence:** HIGH

## Summary

This phase is a housekeeping phase with four concrete items: landing an existing PR, fixing one bug, and closing two issues that are already resolved by existing features. All four items have been thoroughly investigated against the actual codebase.

The core technical work is (1) fixing the unconditional `net/url` import in `protoc-gen-go-client` and (2) landing PR #98 which adds cross-file unwrap resolution. The remaining two items (#91 and #94) require only GitHub issue management -- documenting existing solutions and closing.

**Primary recommendation:** Fix the `net/url` import by adding a `fileNeedsURLImport` check (mirroring the existing `fileNeedsRequestBody` pattern), then land PR #98 after fixing its lint failure, and close #91/#94 with documentation comments.

## Standard Stack

Not applicable -- this phase uses only existing project infrastructure and tooling.

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| protogen | Go protobuf v1.36.x | Plugin framework for all generators | Already used, foundational |
| Go testing | stdlib | Golden file tests, unit tests | Already used |
| protoc | latest | Proto compilation for test execution | Already used |

### Supporting
No new libraries needed. All work is within existing codebase.

## Architecture Patterns

### Pattern 1: Conditional Import Generation
**What:** The `writeImports` function in `internal/clientgen/generator.go` must conditionally include `net/url` based on whether any method in the file uses URL-related features (path params or query params).
**When to use:** Any import in generated code that is not universally needed.
**Confidence:** HIGH (verified against codebase)

The existing codebase already has a precedent for this pattern: `fileNeedsRequestBody()` (line 70-84 of `internal/clientgen/generator.go`) conditionally includes `bytes` based on whether any method has a body (POST/PUT/PATCH).

The fix must follow the same pattern:

```go
// In generator.go, a new function:
func (g *Generator) fileNeedsURLImport(file *protogen.File) bool {
    for _, service := range file.Services {
        for _, method := range service.Methods {
            httpConfig := getMethodHTTPConfig(method)
            // Check for path params
            if httpConfig != nil && len(httpConfig.PathParams) > 0 {
                return true
            }
            // Check for query params on GET/DELETE
            httpMethod := httpMethodPOST
            if httpConfig != nil && httpConfig.Method != "" {
                httpMethod = httpConfig.Method
            }
            if httpMethod == httpMethodGET || httpMethod == httpMethodDELETE {
                queryParams := getQueryParams(method.Input)
                if len(queryParams) > 0 {
                    return true
                }
            }
        }
    }
    return false
}
```

Then in `writeImports` (line 94), pass `needsURL bool` and conditionally emit the `"net/url"` line.

**Key detail:** `net/url` is used in TWO places:
1. `url.PathEscape()` for path parameters (line 551)
2. `url.Values{}` for query parameters (line 564)

Both must be accounted for in the check.

### Pattern 2: Two-Pass Unwrap Collection (PR #98)
**What:** PR #98 introduces a `GlobalUnwrapInfo` struct that collects unwrap annotations from ALL proto files before code generation begins, enabling cross-file resolution within the same Go package.
**When to use:** When a wrapper message (e.g., `BarList` with `unwrap=true`) is in a different `.proto` file than the response message that references it as a map value.
**Confidence:** HIGH (verified in PR diff)

The key change is in `internal/httpgen/generator.go`:
```go
func (g *Generator) Generate() error {
    // Phase 1: Collect global unwrap info from ALL files first
    g.globalUnwrap = CollectGlobalUnwrapInfo(g.plugin.Files)

    // Phase 2: Generate code for each file
    for _, file := range g.plugin.Files { ... }
}
```

And in `unwrap.go`, `collectUnwrapContext` now uses `g.globalUnwrap.UnwrapFields` instead of only scanning messages within the current file.

### Anti-Patterns to Avoid
- **Breaking golden files without updating them:** Any change to `writeImports` will break the `backward_compat_client.pb.go` golden file (which currently includes `net/url` unconditionally). Golden files MUST be updated with `UPDATE_GOLDEN=1`.
- **Forgetting to check both url.PathEscape and url.Values:** A partial fix that only checks query params but not path params (or vice versa) would still produce unused imports in some cases.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Conditional imports | Custom import tracking | Follow `fileNeedsRequestBody` pattern | Proven pattern already in codebase, simple boolean check |
| Golden file updates | Manual file editing | `UPDATE_GOLDEN=1 go test -run TestClientGenGoldenFiles` | Automated, avoids manual drift |

## Common Pitfalls

### Pitfall 1: Forgetting to Update All Golden Files
**What goes wrong:** The `net/url` fix changes generated output, breaking golden file comparisons.
**Why it happens:** Three golden files exist in `internal/clientgen/testdata/golden/`:
- `backward_compat_client.pb.go` -- POST-only service, should NOT have `net/url` after fix
- `query_params_client.pb.go` -- has query params, SHOULD have `net/url`
- `http_verbs_comprehensive_client.pb.go` -- has path params and query params, SHOULD have `net/url`
**How to avoid:** Run `UPDATE_GOLDEN=1 go test -run TestClientGenGoldenFiles` after the fix, then verify the diff manually to confirm only `backward_compat_client.pb.go` lost the `net/url` import.
**Warning signs:** Golden file tests failing after the fix.

### Pitfall 2: PR #98 Lint Failure
**What goes wrong:** PR #98's CI shows a "Lint & Code Quality" failure on the "Run Go fmt check" step.
**Why it happens:** Some file in the PR has formatting issues (likely in the generated example code or new test files).
**How to avoid:** Run `go fmt ./...` on the PR branch before merging. The test jobs (Go 1.24 on ubuntu and macOS) both pass, so the code is functionally correct.
**Warning signs:** The codecov checks also fail (likely because the new example directory has no test coverage, which is expected for examples).

### Pitfall 3: Issue #94 Is Already Closed
**What goes wrong:** Attempting to close an already-closed issue creates confusion.
**Why it happens:** Issue #94 was already closed with a comment explaining that proto3's `json_name` is the existing solution. The issue state is `CLOSED`.
**How to avoid:** Check issue state before acting. For #94, verify the existing closing comment is sufficient documentation. If it is, no action needed.

### Pitfall 4: Merging PR #98 with Stale Branch
**What goes wrong:** PR #98 branch (`unwrap_bug`) may be behind `main`, causing merge conflicts.
**Why it happens:** 5 commits have been made to `main` since the PR was opened (recent docs commits).
**How to avoid:** Rebase or merge `main` into the PR branch before landing.

## Code Examples

### Fix: Conditional net/url Import

The current code in `internal/clientgen/generator.go` (lines 94-112):
```go
func (g *Generator) writeImports(gf *protogen.GeneratedFile, needsBytes bool) {
    gf.P("import (")
    if needsBytes {
        gf.P(`"bytes"`)
    }
    gf.P(`"context"`)
    gf.P(`"encoding/json"`)
    gf.P(`"fmt"`)
    gf.P(`"io"`)
    gf.P(`"net/http"`)
    gf.P(`"net/url"`)  // <-- BUG: unconditional
    gf.P(`"strings"`)
    // ...
```

After fix:
```go
func (g *Generator) writeImports(gf *protogen.GeneratedFile, needsBytes, needsURL bool) {
    gf.P("import (")
    if needsBytes {
        gf.P(`"bytes"`)
    }
    gf.P(`"context"`)
    gf.P(`"encoding/json"`)
    gf.P(`"fmt"`)
    gf.P(`"io"`)
    gf.P(`"net/http"`)
    if needsURL {
        gf.P(`"net/url"`)
    }
    gf.P(`"strings"`)
    // ...
```

### Verification: Issue #91 Root-Level Arrays

The existing `(sebuf.http.unwrap)` annotation already supports root-level arrays. Proof from `internal/httpgen/testdata/proto/unwrap.proto` (lines 91-95):
```protobuf
// RootRepeatedResponse tests root-level repeated unwrap.
// JSON: [{...}, {...}] instead of {"items": [{...}, {...}]}
message RootRepeatedResponse {
  repeated OptionBar items = 1 [(sebuf.http.unwrap) = true];
}
```

The implementation exists in `internal/httpgen/unwrap.go`:
- `generateRootRepeatedUnwrapMarshalJSON()` (line 832)
- `generateRootRepeatedUnwrapUnmarshalJSON()` (line 870)

Test coverage exists in `internal/httpgen/unwrap_test.go`:
- `TestRootUnwrapFileGeneration` (line 202) with subtests for `RootRepeatedResponse`

### Documentation: Issue #94 Field Name Casing

Issue #94 is already closed. The closing comment documents the solution:
```
proto3's built-in json_name field option already provides per-field name override,
and protojson.MarshalOptions{UseProtoNames: true} provides global snake_case.

Use json_name for per-field override:
  message User {
    string first_name = 1 [json_name = "firstName"];
  }
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Single-file unwrap scan | Two-pass global unwrap collection (PR #98) | PR #98 (pending) | Enables cross-file unwrap resolution |
| Unconditional imports | Conditional imports based on usage analysis | This phase (to be done) | Fixes Go compilation error for POST-only services |

## Open Questions

1. **PR #98 Merge Strategy**
   - What we know: The PR has 6 commits, tests pass on both platforms, but lint fails on `go fmt` check. Codecov also fails (expected for example dirs).
   - What's unclear: Whether the lint issue is a trivial formatting fix or requires code changes. The specific file with formatting issues is not visible in the job logs (empty output from `gh run view --log-failed`).
   - Recommendation: Check out the branch locally, run `go fmt ./...`, identify and fix any formatting issues, then rebase onto current `main`.

2. **Issue #91 Closing Comment Content**
   - What we know: The root-level array feature already works via `(sebuf.http.unwrap) = true` on repeated fields. Test coverage exists.
   - What's unclear: Whether a simple "this works already" comment is sufficient, or if the issue closer should reference specific code/docs.
   - Recommendation: Close with a comment that includes (a) example proto usage, (b) link to the test case in `unwrap.proto`, and (c) link to the CLAUDE.md section documenting root-level unwrap.

## Sources

### Primary (HIGH confidence)
- `internal/clientgen/generator.go` -- Direct code inspection of `writeImports` (line 94-112), `generateURLBuilding` (line 534-572), `fileNeedsRequestBody` (line 70-84)
- `internal/httpgen/unwrap.go` -- Direct code inspection of unwrap context collection and root unwrap generation
- `internal/httpgen/testdata/proto/unwrap.proto` -- Test fixtures confirming root-level repeated unwrap exists (lines 91-95)
- `proto/sebuf/http/annotations.proto` -- Field extension definition for `unwrap` (line 73)
- GitHub PR #98 -- Full diff inspected via `gh pr diff 98`
- GitHub Issues #91, #94, #105 -- Full body and comments inspected via `gh issue view`
- `internal/clientgen/golden_test.go` -- Golden file test infrastructure
- `internal/clientgen/testdata/golden/` -- Three golden files that will be affected

### Secondary (MEDIUM confidence)
- PR #98 CI results -- Lint failure on "Run Go fmt check" step, tests pass on both platforms

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- No new libraries, all existing tooling
- Architecture: HIGH -- Patterns directly observed in codebase (fileNeedsRequestBody precedent)
- Pitfalls: HIGH -- Golden files, lint failure, and issue states all verified directly

**Research date:** 2026-02-05
**Valid until:** 2026-03-05 (stable -- no external dependencies to go stale)
