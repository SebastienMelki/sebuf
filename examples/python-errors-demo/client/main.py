"""End-to-end error-handling demo for protoc-gen-py-client.

Exercises every error surface the generator produces:
  - ValidationError (buf.validate body)
  - Typed *Error subclasses (NotFoundError, ConflictError, RateLimitError)
    selected from the response shape by _ERROR_CLASSES
  - *Error embedded as a field on a regular response (BatchCreateItemResult)
    — verifies the from_dict alias that lives alongside populate()

Run after the Go server is up, or use `make demo` from the parent dir.
"""

from __future__ import annotations

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent / "generated" / "proto"))

from errors_demo_client import (  # noqa: E402
    ApiError,
    BatchCreateItemsRequest,
    BatchCreateItemsResponse,
    BatchItemInput,
    ConflictError,
    CreateItemRequest,
    DeleteItemRequest,
    ErrorsDemoServiceClient,
    ErrorsDemoServiceClientOptions,
    FieldValidationError,
    GetItemRequest,
    Item,
    NotFoundError,
    RateLimitError,
    TriggerRateLimitRequest,
    ValidationError,
)


FAIL_COUNT = 0


def check(label: str, actual, expected) -> None:
    global FAIL_COUNT
    ok = actual == expected
    status = "PASS" if ok else "FAIL"
    print(f"  [{status}] {label}")
    if not ok:
        print(f"        expected: {expected!r}")
        print(f"        actual:   {actual!r}")
        FAIL_COUNT += 1


def section(title: str) -> None:
    print(f"\n========================================")
    print(f"  {title}")
    print(f"========================================")


def section_not_found(client: ErrorsDemoServiceClient) -> None:
    section("Typed NotFoundError (registry pick on resource_type/id shape)")
    try:
        client.get_item(GetItemRequest(id="missing"))
        check("expected NotFoundError to be raised", "no exception", "NotFoundError")
        return
    except NotFoundError as e:
        check("caught NotFoundError subclass", isinstance(e, NotFoundError), True)
        check("isinstance ApiError",            isinstance(e, ApiError), True)
        check("isinstance ValidationError",     isinstance(e, ValidationError), False)
        check("status field",                   e.status, 404)
        check("resource_type populated",        e.resource_type, "item")
        check("resource_id populated",          e.resource_id, "missing")


def section_conflict(client: ErrorsDemoServiceClient) -> None:
    section("Typed ConflictError (different field shape from NotFoundError)")
    try:
        client.create_item(CreateItemRequest(title="duplicate"))
        check("expected ConflictError to be raised", "no exception", "ConflictError")
        return
    except ConflictError as e:
        check("caught ConflictError subclass", isinstance(e, ConflictError), True)
        check("not mistakenly NotFoundError",  isinstance(e, NotFoundError), False)
        check("status field",                  e.status, 409)
        check("resource_type populated",       e.resource_type, "item")
        check("title populated",               e.title, "duplicate")
        check("existing_id populated",         e.existing_id, "existing-1")


def section_rate_limit(client: ErrorsDemoServiceClient) -> None:
    section("Typed RateLimitError (matches on retry_after_seconds + detail)")
    try:
        client.trigger_rate_limit(TriggerRateLimitRequest())
        check("expected RateLimitError to be raised", "no exception", "RateLimitError")
        return
    except RateLimitError as e:
        check("caught RateLimitError subclass",  isinstance(e, RateLimitError), True)
        check("not NotFoundError",               isinstance(e, NotFoundError), False)
        check("not ConflictError",               isinstance(e, ConflictError), False)
        check("status field",                    e.status, 429)
        check("retry_after_seconds populated",   e.retry_after_seconds, 30)
        check("detail populated",                e.detail, "demo rate limit")


def section_validation(client: ErrorsDemoServiceClient) -> None:
    section("ValidationError (buf.validate body)")
    try:
        client.create_item(CreateItemRequest(title=""))
        check("expected ValidationError to be raised", "no exception", "ValidationError")
        return
    except ValidationError as e:
        check("caught ValidationError",       isinstance(e, ValidationError), True)
        check("isinstance ApiError",          isinstance(e, ApiError), True)
        check("not mistakenly NotFoundError", isinstance(e, NotFoundError), False)
        check("status field",                 e.status, 400)
        check("has at least one violation",   len(e.violations) >= 1, True)
        check("violation field name",         e.violations[0].field, "title")


def section_embedded_error(client: ErrorsDemoServiceClient) -> None:
    section("*Error as a field on a regular message (Yash's #172 case)")
    resp: BatchCreateItemsResponse = client.batch_create_items(
        BatchCreateItemsRequest(items=[
            BatchItemInput(title="ok-one"),
            BatchItemInput(title=""),           # → embedded validation error
            BatchItemInput(title="duplicate"),  # → embedded validation error
            BatchItemInput(title="ok-two"),
        ])
    )
    check("4 results in response", len(resp.results), 4)

    # Row 0 — happy path
    r0 = resp.results[0]
    check("row 0 title round-trip", r0.title, "ok-one")
    check("row 0 has Item",         r0.item is not None, True)
    check("row 0 has no error",     r0.error, None)
    check("row 0 item is Item type", isinstance(r0.item, Item), True)
    check("row 0 item title",        r0.item.title if r0.item else None, "ok-one")

    # Row 1 — empty title → embedded FieldValidationError
    r1 = resp.results[1]
    check("row 1 has no item",                  r1.item, None)
    check("row 1 has embedded error",           r1.error is not None, True)
    check("row 1 error is FieldValidationError",
          isinstance(r1.error, FieldValidationError), True)
    check("row 1 error field",                  r1.error.field if r1.error else None, "title")
    check("row 1 error description",            r1.error.description if r1.error else None,
          "title is required")

    # Row 2 — "duplicate" title → embedded validation error with different desc
    r2 = resp.results[2]
    check("row 2 has embedded error",       r2.error is not None, True)
    check("row 2 error description differs", r2.error.description if r2.error else None,
          "title already exists")

    # Row 3 — happy path again
    r3 = resp.results[3]
    check("row 3 has Item", r3.item is not None, True)
    check("row 3 has no error", r3.error, None)


def section_exception_hierarchy(client: ErrorsDemoServiceClient) -> None:
    section("Exception hierarchy & propagation")
    try:
        client.delete_item(DeleteItemRequest(id="missing"))
    except Exception as e:
        check("e is NotFoundError",  isinstance(e, NotFoundError), True)
        check("e is ApiError",       isinstance(e, ApiError), True)
        check("e is Exception",      isinstance(e, Exception), True)
        check("e is not ValueError", isinstance(e, ValueError), False)


def main() -> int:
    print("=== Python error-handling demo ===")
    print("Round-trips every error surface the generator produces.\n")

    client = ErrorsDemoServiceClient(
        "http://localhost:3002",
        ErrorsDemoServiceClientOptions(),
    )

    section_not_found(client)
    section_conflict(client)
    section_rate_limit(client)
    section_validation(client)
    section_embedded_error(client)
    section_exception_hierarchy(client)

    print()
    if FAIL_COUNT == 0:
        print("=== All assertions passed ===")
        return 0
    print(f"=== {FAIL_COUNT} assertion(s) failed ===")
    return 1


if __name__ == "__main__":
    sys.exit(main())
