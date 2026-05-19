# Python Client Generator — Handoff (Phase 2)

> Implementation phase is done. This doc hands off the testing / examples / docs / PR work to a fresh agent.
> **Delete this file** when the PR merges.

## What's done

- `cmd/protoc-gen-py-client/main.go` + `internal/pyclientgen/` — complete Python client generator
- Lint-clean (`golangci-lint run` returns 0 issues)
- Smoke-tested end-to-end (generated Python parses with `ast.parse` and runs at runtime)
- 3 commits on branch `feat/py-client`:
  1. `feat(py-client): scaffold protoc-gen-py-client plugin`
  2. `feat(py-client): render message dataclasses with JSON-mapping annotations`
  3. `feat(py-client): implement RPC methods, options, and typed error hierarchy`
- PR #132 was closed with a thank-you comment to @elzalem
- All annotations are wired through `internal/annotations/` directly (no `contractmodel` dependency)

## What's left (in suggested order)

1. **Set up testdata + golden tests** (`internal/pyclientgen/testdata/proto/`, `internal/pyclientgen/golden_test.go`)
2. **Write unit tests** for `keyword.go`, `types.go`, `encoding.go`, and the snakeCase / headerOptionName helpers in `client.go`
3. **Build `examples/python-client-demo/`** mirroring the `examples/ts-client-demo/` Makefile + structure
4. **Write `docs/python-generation.md`**
5. **Update `README.md` + `CLAUDE.md`** plugin lists to include `protoc-gen-py-client`
6. **File GitHub issues** for follow-ups (SSE, async transport, proto binary, docstrings, `__init__.py`)
7. **Delete `PYTHON_CLIENT_REWRITE.md`** and **delete `PY_CLIENT_HANDOFF.md`** (this file)
8. **Open PR** with credit to @elzalem in the description

## Where to look in the repo

| Need a pattern for... | Look at |
|---|---|
| Plugin entry binary | `cmd/protoc-gen-py-client/main.go` (already done) vs `cmd/protoc-gen-ts-client/main.go` |
| Generator structure | `internal/pyclientgen/generator.go` mirrors `internal/tsclientgen/generator.go` |
| Per-feature test protos | `internal/tsclientgen/testdata/proto/*.proto` — copy the per-file pattern |
| Golden test harness | `internal/tsclientgen/golden_test.go` — same skeleton, swap `_client.ts` → `_client.py` |
| Runtime demo with server | `examples/ts-client-demo/Makefile` + `main.go` + `client/` — much richer than PR #132's demo |
| Docs page | `docs/client-generation.md` (Go client) — match its sections |
| Annotation reference list | `CLAUDE.md` lines starting at "Annotation Extension Number Registry" |

## Test patterns

- Use `UPDATE_GOLDEN=1 go test -run TestPyClientGenGoldenFiles` to refresh expected output after intentional changes.
- After writing each generated `.py` file in the golden test, also run `python3 -c "import ast; ast.parse(open(<path>).read())"` — protects against silent syntactic regressions golden-string-compare cannot catch.
- For unit tests, mirror what `internal/tsclientgen/helpers_test.go` does: pure-function tests on the renderer helpers.
- Suggested per-feature golden test protos (one each, mirroring tsclientgen):
  `basic.proto`, `http_verbs.proto`, `query_params.proto`, `unwrap.proto`,
  `int64_encoding.proto`, `enum_encoding.proto`, `nullable.proto`,
  `empty_behavior.proto`, `timestamp_format.proto`, `bytes_encoding.proto`,
  `flatten.proto`, `oneof_discriminator.proto`, `headers.proto`, `errors.proto`, `sse.proto`.

## Linting

```bash
# golangci-lint must be built with Go 1.26 (matches go.mod)
GOTOOLCHAIN=go1.26.0 go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
golangci-lint run ./internal/pyclientgen/... ./cmd/protoc-gen-py-client/...
```

Re-lint after every meaningful diff. The user explicitly asked us to lint as we go.

## Known repo state at handoff

- **Pre-existing test failure on main**: `internal/openapiv3` has 2 failing tests around `google.protobuf.Timestamp`/`Duration` descriptor comments. Not caused by py-client work — confirmed reproducible on `main`. Don't try to fix as part of this PR; either flag it separately or run only `go test ./internal/pyclientgen ./internal/tsclientgen ./internal/clientgen ./internal/tscommon ./internal/tsservergen ./internal/httpgen ./internal/annotations` to skip it.
- **PYTHON_CLIENT_REWRITE.md** still exists at repo root (the original handoff). Delete it as part of this work.
- **PR #132** is closed; @elzalem was notified. Credit in PR body.
- **Branch**: `feat/py-client`, 3 commits ahead of `main`. Not pushed yet.

## Architectural decisions baked in

- **Stdlib-only generated output**: `urllib`, `dataclasses`, `enum.IntEnum`, `typing.Protocol`. No third-party Python dep is forced.
- **Transport injection via Protocol**: users can pass any duck-typed `HttpTransport` (requests/httpx/aiohttp). Default is `UrllibTransport`.
- **Per-`*Error`-message Exception subclasses**: each generated message ending in `Error` becomes an `ApiError` subclass with constructor kwargs for its fields, plus a `populate()` classmethod. A `_ERROR_CLASSES` registry keyed by required JSON field-name set lets `_raise_for_status` pick the most specific exception.
- **JSON-mapping annotations all applied at generation time** in `internal/pyclientgen/encoding.go` — call `internal/annotations/` helpers directly.
- **SSE**: detected via `HttpConfig.stream`, emits a stub that raises `NotImplementedError` with a link to the follow-up issue. Don't implement SSE in this PR.
- **Content-Type**: JSON only. Proto binary path raises `NotImplementedError` until a follow-up adds it (likely needs `google-protobuf` dep — defer the decision).
- **Python keywords**: `internal/pyclientgen/keyword.go` has the hard-coded Python 3.10 keyword list. Regenerate command is documented in that file's comment.
- **Minimum Python**: 3.10 (uses `X | None`, `list[T]`, `from __future__ import annotations`).

## What was NOT cherry-picked from PR #132 (and why)

- `comprehensive_models.proto` / `comprehensive_services.proto` — kitchen-sinky; the per-feature proto pattern from tsclientgen is more maintainable. Write fresh focused ones.
- `examples/python-client-demo/Makefile` — way thinner than `examples/ts-client-demo/Makefile`. Use the ts-client-demo pattern instead (server + client + runnable `make demo` target).
- The `contractmodel` dependency — entire architectural decision was to avoid it.

The PR #132 worktree was added at `/tmp/pr132-ref/` during the implementation; that may or may not survive context-clear. If it's gone, re-fetch:
```bash
git fetch origin pull/132/head:pr132-reference
git worktree add /tmp/pr132-ref pr132-reference
```

## Verification checklist before opening the PR

- [ ] `golangci-lint run ./...` clean (or unchanged from main's existing issues)
- [ ] `go build ./...` clean
- [ ] All py-client golden tests pass
- [ ] `python3 -c "import ast; ast.parse(...)"` passes on every generated golden file
- [ ] Demo runs end-to-end (`cd examples/python-client-demo && make demo`)
- [ ] README + CLAUDE.md include `protoc-gen-py-client`
- [ ] Follow-up GitHub issues filed and linked from the PR description
- [ ] `PYTHON_CLIENT_REWRITE.md` and `PY_CLIENT_HANDOFF.md` deleted
- [ ] PR description credits @elzalem
