---
phase: 02-shared-annotations
plan: 04
subsystem: api
tags: [protogen, annotations, openapiv3, migration, code-deduplication, error-handling, serialization-audit]

# Dependency graph
requires:
  - phase: 02-shared-annotations plan 03
    provides: clientgen and tsclientgen migrated to shared annotations, validating the shared package API across 3 generators
provides:
  - openapiv3 fully migrated to shared annotations package
  - All 4 generators now import internal/annotations for all annotation parsing
  - ~1,289 total lines of duplicated annotation code eliminated across Phase 2
  - Cross-file annotation resolution fails hard on errors (no silent suppression)
  - Serialization audit confirmed correct (protojson for proto messages, encoding/json only for interface checks)
  - Phase 2 complete -- shared annotations foundation ready for all future features
affects: [03-existing-client-review, 04-json-primitive-encoding, all future phases]

# Tech tracking
tech-stack:
  added: []
  patterns: [shared annotation imports across all 4 generators, fail-hard error propagation in cross-file resolution]

key-files:
  created:
    - internal/openapiv3/types_test.go
  modified:
    - internal/openapiv3/generator.go
    - internal/openapiv3/types.go
    - internal/httpgen/unwrap.go
    - internal/httpgen/generator.go
  deleted:
    - internal/openapiv3/http_annotations.go (406 lines)
    - internal/openapiv3/http_annotations_test.go (389 lines)

key-decisions:
  - "Lowercase HTTP method constants in openapiv3 generator: OpenAPI requires lowercase method names, shared package returns UPPERCASE. Used strings.ToLower() at usage site plus local lowercase constants (httpMethodGet, etc.) to avoid goconst lint issues"
  - "convertHeadersToParameters and mapHeaderTypeToOpenAPI kept in openapiv3/types.go: these are OpenAPI-specific type conversion, not annotation parsing"
  - "Error propagation via function signature changes: collectFileUnwrapFields, collectUnwrapFieldsRecursive, collectAllUnwrapFields, collectUnwrapContext, and CollectGlobalUnwrapInfo all now return error"
  - "Serialization audit: no code changes needed -- encoding/json in httpgen correctly used only for json.Marshaler/json.Unmarshaler interface checks"

patterns-established:
  - "Fail-hard on annotation resolution: cross-file annotation errors propagate up to generator and stop code generation with descriptive messages"
  - "All 4 generators use shared annotations package exclusively: zero duplicated annotation parsing anywhere in codebase"

# Metrics
duration: 10min
completed: 2026-02-05
---

# Phase 2 Plan 4: Migrate openapiv3, Fix Error Suppression, Final Verification Summary

**openapiv3 fully migrated to internal/annotations with 795 lines deleted, cross-file error suppression replaced with fail-hard propagation, serialization audit confirming protojson-only proto marshaling -- Phase 2 complete with all 4 generators unified on shared annotations**

## Performance

- **Duration:** 10 min
- **Started:** 2026-02-05T17:13:00Z
- **Completed:** 2026-02-05T17:23:28Z
- **Tasks:** 3/3
- **Lines deleted:** 795 (406 from http_annotations.go + 389 from http_annotations_test.go)
- **Files modified:** 7

## Accomplishments
- Migrated openapiv3/generator.go to use shared `internal/annotations` package for all annotation parsing (the 4th and final generator)
- Moved OpenAPI-specific functions (convertHeadersToParameters, mapHeaderTypeToOpenAPI) from deleted http_annotations.go to types.go
- Deleted openapiv3/http_annotations.go (406 lines) and http_annotations_test.go (389 lines)
- Fixed cross-file error suppression in httpgen/unwrap.go: all annotation resolution errors now propagate up to the generator and halt code generation with descriptive messages
- Serialization audit confirmed correct: encoding/json only used for json.Marshaler/json.Unmarshaler interface checks, protojson used exclusively for proto message serialization
- All 4 generators build, all golden file tests pass unchanged, full test suite passes, lint clean
- Phase 2 complete: ~1,289 total lines of duplicated annotation code eliminated across all 4 plans

## Task Commits

Each task was committed atomically:

1. **Task 1: Migrate openapiv3 to shared annotations and delete old files** - `c12da4d` (refactor)
   - Updated imports in generator.go and types.go, added `internal/annotations` import
   - Replaced all local annotation calls with shared package equivalents
   - Used `strings.ToLower()` for OpenAPI-required lowercase HTTP methods
   - Added lowercase HTTP method constants to satisfy goconst lint
   - Moved convertHeadersToParameters, mapHeaderTypeToOpenAPI, and header type constants to types.go
   - Created types_test.go with TestMapHeaderTypeToOpenAPI and BenchmarkMapHeaderTypeToOpenAPI
   - Deleted internal/openapiv3/http_annotations.go (406 lines)
   - Deleted internal/openapiv3/http_annotations_test.go (389 lines)

2. **Task 2: Fix cross-file error suppression in httpgen/unwrap.go** - `11f9578` (fix)
   - Changed CollectGlobalUnwrapInfo, collectFileUnwrapFields, collectUnwrapFieldsRecursive, collectAllUnwrapFields, collectUnwrapContext to return errors
   - Replaced `continue` on annotation errors with `return fmt.Errorf(...)` wrapping
   - Updated Generator.Generate() to handle error from CollectGlobalUnwrapInfo
   - Fixed three govet shadow warnings (`:=` to `=` in error reassignment)

3. **Task 3: Serialization audit and final verification** - verification-only, no commit needed
   - Confirmed encoding/json usage in httpgen is correct (interface checks only)
   - Confirmed protojson used for all proto message serialization
   - Full test suite passes (6/6 packages)
   - All 4 plugin binaries build successfully
   - All golden file tests pass unchanged
   - proto.GetExtension only in internal/annotations/ and openapiv3/validation.go (buf.validate)
   - Lint clean (0 issues)

## Files Created/Modified
- **Created:** `internal/openapiv3/types_test.go` -- OpenAPI-specific tests (mapHeaderTypeToOpenAPI)
- **Deleted:** `internal/openapiv3/http_annotations.go` (406 lines of duplicated annotation parsing)
- **Deleted:** `internal/openapiv3/http_annotations_test.go` (389 lines of tests for now-shared/moved functions)
- **Modified:** `internal/openapiv3/generator.go` -- imports annotations package, uses shared functions, lowercase HTTP method constants
- **Modified:** `internal/openapiv3/types.go` -- uses shared annotations, received convertHeadersToParameters and mapHeaderTypeToOpenAPI from deleted file
- **Modified:** `internal/httpgen/unwrap.go` -- error propagation instead of silent suppression
- **Modified:** `internal/httpgen/generator.go` -- handles error from CollectGlobalUnwrapInfo

## Decisions Made
- **Lowercase HTTP methods via constants:** OpenAPI spec requires lowercase HTTP methods ("get", "post", etc.) but the shared annotations package returns uppercase ("GET", "POST") for HTTP-standard usage. Used `strings.ToLower()` at the usage site in generator.go and defined local lowercase constants (`httpMethodGet`, etc.) to avoid goconst lint issues with repeated string literals.
- **OpenAPI-specific functions stay in package:** `convertHeadersToParameters` and `mapHeaderTypeToOpenAPI` are OpenAPI type-conversion functions (not annotation parsing), so they were moved to types.go within the openapiv3 package rather than to the shared annotations package.
- **Error propagation signature change:** Changed 5 functions in unwrap.go to return errors, propagating all the way up to Generator.Generate(). This is a breaking API change for CollectGlobalUnwrapInfo, but the only caller is within the same package.
- **Serialization audit: no changes needed:** Confirmed that encoding/json in httpgen is used correctly for json.Marshaler/json.Unmarshaler interface checks on unwrap types, not for proto message serialization.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added lowercase HTTP method constants for goconst lint**
- **Found during:** Task 1 (openapiv3 migration)
- **Issue:** After replacing inline HTTP method references, the string `"post"` appeared 3 times triggering goconst lint error
- **Fix:** Added `httpMethodGet`, `httpMethodPost`, `httpMethodPut`, `httpMethodDelete`, `httpMethodPatch` constants and used them throughout generator.go
- **Files modified:** `internal/openapiv3/generator.go`
- **Commit:** `c12da4d`

**2. [Rule 1 - Bug] Fixed govet shadow warnings in error propagation**
- **Found during:** Task 2 (error suppression fix)
- **Issue:** Three places where `if err := ...` shadowed outer `err` variable after changing function signatures to return errors
- **Fix:** Changed `:=` to `=` for error reassignment in generator.go:56, unwrap.go:105, unwrap.go:163
- **Files modified:** `internal/httpgen/generator.go`, `internal/httpgen/unwrap.go`
- **Commit:** `11f9578`

---

**Total deviations:** 2 auto-fixed (1 blocking lint issue, 1 bug in variable shadowing)
**Impact on plan:** Both fixes required for clean compilation and lint compliance. No scope creep.

## Issues Encountered
None -- migration and error propagation changes executed cleanly with all tests passing.

## User Setup Required
None.

## Next Phase Readiness
- Phase 2 fully complete: all 4 generators unified on `internal/annotations` shared package
- ~1,289 lines of duplicated annotation code eliminated across 4 plans
- Cross-file annotation resolution is now fail-hard (no silent errors)
- Serialization confirmed consistent (protojson for proto messages)
- Ready for Phase 3: Existing Client Review (Go client and TypeScript client audit)
- All golden file tests pass, providing regression safety net for upcoming changes

---
*Phase: 02-shared-annotations*
*Completed: 2026-02-05*
