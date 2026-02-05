---
phase: 02-shared-annotations
plan: 01
subsystem: api
tags: [protogen, annotations, shared-package, convention-based-api]

# Dependency graph
requires:
  - phase: 01-foundation-quick-wins
    provides: cross-file unwrap resolution and URL encoding fixes
provides:
  - internal/annotations package with all shared annotation types and functions
  - Convention-based extensibility pattern for adding new annotation types
  - Unified QueryParam struct with all fields needed by all 4 generators
  - HTTPMethodToString (uppercase) and HTTPMethodToLower (lowercase) for consistent method representation
affects: [02-shared-annotations plan 02 (generator migration), 04-07 JSON mapping phases (new annotations)]

# Tech tracking
tech-stack:
  added: []
  patterns: [convention-based package with one file per annotation concept, GetXxx function signatures]

key-files:
  created:
    - internal/annotations/doc.go
    - internal/annotations/http_config.go
    - internal/annotations/headers.go
    - internal/annotations/query.go
    - internal/annotations/unwrap.go
    - internal/annotations/field_examples.go
    - internal/annotations/path.go
    - internal/annotations/method.go
    - internal/annotations/helpers.go
    - internal/annotations/annotations_test.go
  modified: []

key-decisions:
  - "Exported all structs with transparent fields (HTTPConfig, ServiceConfig, QueryParam, UnwrapFieldInfo, UnwrapValidationError)"
  - "Accept protogen types (*protogen.Method, *protogen.Service, etc.) not protoreflect types -- matches codebase patterns"
  - "GetServiceBasePath returns string directly rather than *ServiceConfig struct -- simpler API for single-field config"
  - "Unified QueryParam struct has all fields from all 4 generators (FieldName, FieldGoName, FieldJSONName, ParamName, Required, FieldKind, Field)"
  - "CombineHeaders uses sort.Strings instead of bubble sort from openapiv3 -- cleaner idiomatic Go"
  - "HTTP method constants kept unexported in method.go -- only exported via HTTPMethodToString/HTTPMethodToLower functions"

patterns-established:
  - "Convention-based extensibility: one file per annotation concept, GetXxx() function signatures"
  - "Each annotation file follows: imports, types, GetXxx functions with protogen parameters"
  - "Two unwrap APIs: GetUnwrapField (full validation) and FindUnwrapField (simple lookup) for different generator needs"

# Metrics
duration: 5min
completed: 2026-02-05
---

# Phase 2 Plan 1: Shared Annotations Package Summary

**Convention-based internal/annotations package with 22 exports covering HTTP config, headers, query params, unwrap, path utils, and method conversion -- ready for generator migration**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-05T16:46:51Z
- **Completed:** 2026-02-05T16:51:57Z
- **Tasks:** 2
- **Files created:** 10

## Accomplishments
- Created `internal/annotations` package with 9 source files following convention-based extensibility pattern
- Unified all annotation parsing from 4 generators into shared types and functions with zero import cycles
- QueryParam struct covers all fields from all 4 generators (httpgen, clientgen, tsclientgen, openapiv3)
- Comprehensive unit test suite with 12 test functions and 8 benchmarks covering all pure functions

## Task Commits

Each task was committed atomically:

1. **Task 1: Create internal/annotations package** - `e813d6c` (feat)
2. **Task 2: Write unit tests** - `a91d27c` (test)
3. **Fix: nolint directive for linter auto-fix** - `a452694` (fix)

## Files Created/Modified
- `internal/annotations/doc.go` - Package documentation with convention-based extensibility pattern
- `internal/annotations/http_config.go` - GetMethodHTTPConfig, GetServiceBasePath, HTTPConfig, ServiceConfig types
- `internal/annotations/headers.go` - GetServiceHeaders, GetMethodHeaders, CombineHeaders (sorted merge)
- `internal/annotations/query.go` - GetQueryParams with unified QueryParam struct (7 fields)
- `internal/annotations/unwrap.go` - HasUnwrapAnnotation, GetUnwrapField (validated), FindUnwrapField (simple), IsRootUnwrap
- `internal/annotations/field_examples.go` - GetFieldExamples
- `internal/annotations/path.go` - ExtractPathParams, BuildHTTPPath, EnsureLeadingSlash
- `internal/annotations/method.go` - HTTPMethodToString (uppercase), HTTPMethodToLower (lowercase)
- `internal/annotations/helpers.go` - LowerFirst
- `internal/annotations/annotations_test.go` - 12 test functions + 8 benchmarks

## Decisions Made
- **Transparent structs with protogen parameters:** All exported structs have exported fields, all functions accept protogen types. This matches the existing codebase style where generators work with protogen types directly.
- **GetServiceBasePath returns string:** Simpler API than returning a single-field struct. The struct is still available as `ServiceConfig` for generators that need it.
- **Unified QueryParam with all fields:** Rather than each generator selecting subset fields, the shared struct populates all fields upfront. Generators use what they need.
- **Two unwrap APIs:** `GetUnwrapField` has full validation (httpgen needs this), `FindUnwrapField` is simple lookup (tsclientgen/openapiv3 only need repeated field). Both exported for generator flexibility.
- **sort.Strings in CombineHeaders:** Replaced bubble sort from openapiv3 with stdlib sort for idiomatic Go.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed linter auto-fix conflict with sebuf/http import**
- **Found during:** Task 2 verification (lint-fix)
- **Issue:** Linter auto-replaced `"GET"` with `http.MethodGet` in test file, but `http` import refers to `sebuf/http` not `net/http`, causing compilation failure
- **Fix:** Added `//nolint:usestdlibvars` directive with explanation comment
- **Files modified:** `internal/annotations/annotations_test.go`
- **Verification:** `go test` passes, `make lint-fix` clean
- **Committed in:** `a452694`

**2. [Rule 1 - Bug] Fixed goconst linter warning for repeated "POST" string**
- **Found during:** Task 1 verification (lint-fix)
- **Issue:** String `"POST"` appeared 3 times in `method.go` triggering goconst linter
- **Fix:** Extracted HTTP method strings into unexported constants (methodGET, methodPOST, etc.)
- **Files modified:** `internal/annotations/method.go`
- **Verification:** `make lint-fix` reports 0 issues
- **Committed in:** `e813d6c` (part of Task 1 commit)

---

**Total deviations:** 2 auto-fixed (2 bug fixes)
**Impact on plan:** Both fixes necessary for clean compilation and linting. No scope creep.

## Issues Encountered
None -- plan executed smoothly.

## User Setup Required
None -- no external service configuration required.

## Next Phase Readiness
- Shared annotations package is compiled, tested, and lint-clean
- All 22 exported symbols ready for generator migration in plan 02-02
- No generator code was modified -- migration is next step
- Convention-based pattern documented in doc.go for future annotation additions

---
*Phase: 02-shared-annotations*
*Completed: 2026-02-05*
