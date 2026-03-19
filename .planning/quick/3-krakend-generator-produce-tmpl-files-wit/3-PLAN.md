---
phase: quick-3
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/krakendgen/tmpl_generator.go
  - internal/krakendgen/tmpl_generator_test.go
  - internal/krakendgen/types.go
  - internal/krakendgen/generator.go
  - cmd/protoc-gen-krakend/main.go
  - internal/krakendgen/golden_test.go
  - internal/krakendgen/testdata/golden/*.krakend.tmpl
autonomous: true
requirements: [TMPL-01]

must_haves:
  truths:
    - "Generator produces .tmpl files alongside .json files when invoked"
    - "Template files use Go template syntax for host variables ({{ .vars.xxx_host }})"
    - "Template files include sd:static, disable_host_sanitize:false, and backend/http return_error_code:true in every backend"
    - "Template files use {{ include }} or {{ template }} directives for shared configs (JWT, input headers)"
    - "Existing .json output is unchanged (no regression)"
    - "Golden tests cover all 11 existing proto test cases for .tmpl output"
  artifacts:
    - path: "internal/krakendgen/tmpl_generator.go"
      provides: "Template string generation from Endpoint slice"
      min_lines: 80
    - path: "internal/krakendgen/testdata/golden/simple_service.krakend.tmpl"
      provides: "Golden file for simplest template case"
      min_lines: 10
    - path: "cmd/protoc-gen-krakend/main.go"
      provides: "Plugin entry point writing both .json and .tmpl files"
  key_links:
    - from: "cmd/protoc-gen-krakend/main.go"
      to: "internal/krakendgen/tmpl_generator.go"
      via: "GenerateTemplateFile function call"
      pattern: "GenerateTemplateFile"
    - from: "internal/krakendgen/tmpl_generator.go"
      to: "internal/krakendgen/types.go"
      via: "Uses Endpoint and Backend types"
      pattern: "Endpoint"
    - from: "internal/krakendgen/golden_test.go"
      to: "internal/krakendgen/testdata/golden/*.krakend.tmpl"
      via: "Golden file comparison"
      pattern: "krakend\\.tmpl"
---

<objective>
Add Go template (.tmpl) file generation to the KrakenD protoc plugin for KrakenD Flexible Config compatibility.

Purpose: The current generator produces static .json files with hardcoded hosts. Real KrakenD deployments use Flexible Config with Go templates for variable hosts, shared config includes, and environment-specific settings. This makes generated output directly usable in production KrakenD setups.

Output: Each protoc invocation now produces TWO files: `krakend.json` (existing, unchanged) and `krakend.tmpl` (new, with Go template syntax). The .tmpl file follows the user's existing conventions from their hand-written KrakenD templates.
</objective>

<execution_context>
@/Users/sebastienmelki/.claude/get-shit-done/workflows/execute-plan.md
@/Users/sebastienmelki/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@internal/krakendgen/generator.go
@internal/krakendgen/types.go
@internal/krakendgen/namespaces.go
@internal/krakendgen/golden_test.go
@cmd/protoc-gen-krakend/main.go
@internal/krakendgen/testdata/golden/full_gateway_service.krakend.json
@internal/krakendgen/testdata/golden/simple_service.krakend.json
</context>

<tasks>

<task type="auto">
  <name>Task 1: Create template generator and derive host variable name convention</name>
  <files>
    internal/krakendgen/tmpl_generator.go
    internal/krakendgen/tmpl_generator_test.go
  </files>
  <action>
Create `internal/krakendgen/tmpl_generator.go` with a `GenerateTemplateFile(endpoints []Endpoint, serviceName string) (string, error)` function that converts a slice of Endpoint objects into a KrakenD Flexible Config `.tmpl` string.

The template output must follow these conventions derived from the user's existing hand-written templates:

**Host variable derivation:**
- The `host` array from the Endpoint's Backend is replaced with a Go template variable reference.
- Convention: `{{ .vars.SERVICE_NAME_host }}` where SERVICE_NAME is the proto service name converted to snake_case lowercase.
- Example: if the service is "UserService", the host becomes `{{ .vars.user_service_host }}`.
- If an endpoint has a method-level host override (different from the service-level host), use `{{ .vars.SERVICE_NAME_METHOD_NAME_host }}` (e.g., `{{ .vars.user_service_get_user_host }}`).
- For the common case where all endpoints share the service-level host, all backends use the same variable.

**Backend always includes these fields (matching user's patterns):**
- `"sd": "static"` — service discovery mode
- `"disable_host_sanitize": false`
- `"extra_config"` always contains at minimum `"backend/http": { "return_error_code": true }` (merged with any existing extra_config like circuit breaker, cache, rate limit)

**Shared config directives:**
- When the service has JWT auth (auth/validator in extra_config), emit `{{ template "jwt_auth_validator.tmpl" . }}` in the endpoint's extra_config instead of inlining the full JWT config. The user can create their own `jwt_auth_validator.tmpl` partial. Keep a comment showing what values the template should contain.
- When input_headers are present, emit `"input_headers": [...]` inline (not as an include) since header lists vary per endpoint.
- Rate limit, circuit breaker, cache configs remain inline since they are endpoint-specific.

**Output structure:**
- NOT wrapped in KrakenDConfig ($schema/version) — .tmpl files contain only the array of endpoint objects, since they will be `{{ include }}`d into a parent krakend.tmpl.
- Output is a JSON array of endpoint objects (valid JSON except for the Go template directives).
- Use 2-space indentation matching the existing .json golden files.

**Implementation approach:**
- Use `text/template` from Go stdlib to render the output. Define a Go template that iterates over the endpoints.
- Alternatively, build the string manually with a strings.Builder for precise control over Go template directive placement (this is probably easier since we need to mix JSON with Go template syntax — the output IS a Go template, we are not using Go templates to generate it).
- The function takes a `serviceName` parameter (the proto service name) to derive the host variable name.

Create unit tests in `tmpl_generator_test.go` that test:
1. `hostVarName(serviceName)` helper: "UserService" -> "user_service_host", "FullGatewayService" -> "full_gateway_service_host"
2. A simple Endpoint (no extra_config) produces correct template output with `{{ .vars.xxx_host }}`, sd:static, disable_host_sanitize:false, backend/http return_error_code
3. An endpoint with JWT auth uses `{{ template "jwt_auth_validator.tmpl" . }}` directive
4. An endpoint with circuit breaker + cache in backend extra_config merges with backend/http return_error_code

Do NOT use Go's `text/template` to generate the output — the output itself IS a Go template file. Use strings.Builder or similar to construct the string with careful formatting. Use `encoding/json` for individual value serialization where appropriate (e.g., marshal a map to get the rate limit config inline).
  </action>
  <verify>
Run `go test -v -run TestHostVarName ./internal/krakendgen/` and `go test -v -run TestGenerateTemplate ./internal/krakendgen/` — all tests pass.
Run `go vet ./internal/krakendgen/` — no errors.
  </verify>
  <done>
GenerateTemplateFile function exists, converts Endpoint slices to .tmpl strings with Go template syntax for hosts, always includes sd:static/disable_host_sanitize/return_error_code in backends, uses template directives for JWT auth. Unit tests confirm the conversion logic.
  </done>
</task>

<task type="auto">
  <name>Task 2: Wire template generation into protoc plugin and add golden tests</name>
  <files>
    cmd/protoc-gen-krakend/main.go
    internal/krakendgen/golden_test.go
    internal/krakendgen/generator.go
    internal/krakendgen/testdata/golden/simple_service.krakend.tmpl
    internal/krakendgen/testdata/golden/timeout_config.krakend.tmpl
    internal/krakendgen/testdata/golden/host_config.krakend.tmpl
    internal/krakendgen/testdata/golden/headers_forwarding.krakend.tmpl
    internal/krakendgen/testdata/golden/query_forwarding.krakend.tmpl
    internal/krakendgen/testdata/golden/combined_forwarding.krakend.tmpl
    internal/krakendgen/testdata/golden/rate_limit_service.krakend.tmpl
    internal/krakendgen/testdata/golden/jwt_auth_service.krakend.tmpl
    internal/krakendgen/testdata/golden/circuit_breaker_service.krakend.tmpl
    internal/krakendgen/testdata/golden/cache_concurrent_service.krakend.tmpl
    internal/krakendgen/testdata/golden/full_gateway_service.krakend.tmpl
  </files>
  <action>
**1. Update `cmd/protoc-gen-krakend/main.go`:**

After generating `krakend.json` (existing logic, unchanged), also generate `krakend.tmpl`. The generateFiles function needs to:
- Track endpoints per service (currently all endpoints are flattened into one list). Modify to group endpoints by service name so each service gets its own template variable name.
- Since the current architecture collects ALL endpoints across services into one file, the simplest approach is: after generating krakend.json, also call `krakendgen.GenerateTemplateFile(allEndpoints, serviceName)` and write the result to a second `plugin.NewGeneratedFile("krakend.tmpl", "")`.
- For the service name: if there's only one service across all files, use that service name. If there are multiple services, the host variable needs to be per-service. The simplest approach: change GenerateService to also return the service name, or have GenerateTemplateFile accept a mapping of endpoint->serviceName. Alternatively, add a ServiceName field to the Endpoint type.

The cleanest approach: **Add a `ServiceName string` field to the Endpoint struct** (with `json:"-"` tag so it doesn't appear in JSON output). Populate it in `buildEndpoint()`. Then GenerateTemplateFile can read it from each endpoint to derive per-service host variables. This way, multi-service proto files get correct per-service host variables (e.g., `user_service_host` vs `product_service_host`).

**2. Update `internal/krakendgen/types.go`:**
Add `ServiceName string \`json:"-"\`` to the Endpoint struct.

**3. Update `internal/krakendgen/generator.go`:**
In `buildEndpoint()`, set `ep.ServiceName = string(service.Desc.Name())`.

**4. Update `internal/krakendgen/golden_test.go`:**
Add a `TestKrakenDTemplateGoldenFiles` test function (parallel to TestKrakenDGoldenFiles) that:
- Runs protoc with protoc-gen-krakend for each test case proto
- Reads the generated `krakend.tmpl` file from the temp dir
- Compares against `testdata/golden/{name}.krakend.tmpl` golden files
- Supports UPDATE_GOLDEN=1 to create/update golden files

**5. Create golden files:**
Run with `UPDATE_GOLDEN=1` to auto-generate all 11 `.krakend.tmpl` golden files. Then manually verify at least the full_gateway_service.krakend.tmpl and simple_service.krakend.tmpl look correct.

**6. Verify existing tests still pass:**
The existing krakend.json golden tests and validation error tests must be completely unaffected.

After all changes: `rm -rf bin && make build` to rebuild the binary, then run all krakend tests.
  </action>
  <verify>
Run `rm -rf bin && make build` — builds successfully.
Run `go test -v -run TestKrakenDGoldenFiles ./internal/krakendgen/` — all 11 existing JSON golden tests pass (no regression).
Run `go test -v -run TestKrakenDTemplateGoldenFiles ./internal/krakendgen/` — all 11 template golden tests pass.
Run `go test -v -run TestKrakenDValidationErrors ./internal/krakendgen/` — all validation error tests pass.
Run `make lint-fix` — no lint errors.
Run `./scripts/run_tests.sh --fast` — all project tests pass.
  </verify>
  <done>
The protoc-gen-krakend plugin now produces both krakend.json (unchanged) and krakend.tmpl (new) on every invocation. All 11 test cases have .tmpl golden files. The template files contain Go template syntax for host variables, sd:static, disable_host_sanitize:false, return_error_code:true, and template directives for JWT auth. No regression in existing JSON output or validation error tests.
  </done>
</task>

</tasks>

<verification>
1. Build: `rm -rf bin && make build` succeeds
2. Existing JSON golden tests: `go test -v -run TestKrakenDGoldenFiles ./internal/krakendgen/` all pass
3. New template golden tests: `go test -v -run TestKrakenDTemplateGoldenFiles ./internal/krakendgen/` all pass
4. Validation error tests: `go test -v -run TestKrakenDValidationErrors ./internal/krakendgen/` all pass
5. Full test suite: `./scripts/run_tests.sh --fast` passes
6. Lint: `make lint-fix` reports 0 issues
7. Manual inspection: `full_gateway_service.krakend.tmpl` contains `{{ .vars.full_gateway_service_host }}`, `{{ template "jwt_auth_validator.tmpl" . }}`, `"sd": "static"`, `"disable_host_sanitize": false`, `"backend/http": { "return_error_code": true }`
</verification>

<success_criteria>
- protoc-gen-krakend produces TWO output files: krakend.json and krakend.tmpl
- krakend.json output is identical to before (zero regression)
- krakend.tmpl uses {{ .vars.SERVICE_host }} for backend hosts
- krakend.tmpl backends always include sd:static, disable_host_sanitize:false, backend/http return_error_code:true
- krakend.tmpl uses {{ template "jwt_auth_validator.tmpl" . }} for JWT auth config
- All 11 test proto files have corresponding .krakend.tmpl golden files
- All tests pass, no lint issues
</success_criteria>

<output>
After completion, create `.planning/quick/3-krakend-generator-produce-tmpl-files-wit/3-SUMMARY.md`
</output>
