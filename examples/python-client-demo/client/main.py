"""End-to-end demo for protoc-gen-py-client.

Mirrors examples/ts-client-demo/client/main.ts section-by-section so the
two client surfaces can be compared side by side. Talks to the Go HTTP
server in ../main.go which exposes a CRUD-style NoteService.
"""

from __future__ import annotations

import sys
import time
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent / "generated" / "proto"))

from note_service_client import (  # noqa: E402  (sys.path manipulation above)
    ApiError,
    CreateNoteRequest,
    DeleteNoteRequest,
    GetNoteRequest,
    GetNotesByTagRequest,
    HttpResponse,
    HttpTransport,
    ListNotesRequest,
    NoteServiceCallOptions,
    NoteServiceClient,
    NoteServiceClientOptions,
    NotFoundError,
    Priority,
    Status,
    Tag,
    UpdateNoteRequest,
    UrllibTransport,
    ValidationError,
    ArchiveNoteRequest,
)


# ============================================================================
# Section 1: Client Configuration
# ============================================================================
# The generated client accepts a typed options dataclass. Service-level
# headers (X-API-Key, X-Tenant-ID) become typed kwargs (`api_key`,
# `tenant_id`); default_headers covers anything not declared in proto.

def section1_client_configuration() -> NoteServiceClient:
    print("========================================")
    print("  Section 1: Client Configuration")
    print("========================================\n")

    options = NoteServiceClientOptions(
        api_key="550e8400-e29b-41d4-a716-446655440000",
        tenant_id="42",
        default_headers={"X-Custom-Header": "demo-value"},
    )
    client = NoteServiceClient("http://localhost:3000", options)

    print("Client created with:")
    print("  api_key:    550e8400-e29b-41d4-a716-446655440000 (uuid format)")
    print("  tenant_id:  42 (integer type, stringified for the wire)")
    print("  + custom default header: X-Custom-Header")
    return client


# ============================================================================
# Section 2: CRUD Operations
# ============================================================================
# Exercises every HTTP verb: GET, POST, PUT, PATCH, DELETE.

def section2_crud_operations(client: NoteServiceClient) -> None:
    print("\n========================================")
    print("  Section 2: CRUD Operations")
    print("========================================\n")

    print("--- LIST notes (GET /api/v1/notes) ---")
    all_notes = client.list_notes(ListNotesRequest())
    print(f"Found {all_notes.total} seed notes:")
    for n in all_notes.notes:
        due = f" (due: {n.due_date})" if n.due_date else ""
        print(f"  {n.id}: \"{n.title}\" [{n.priority.name}, {n.status.name}]{due}")
        if n.tags:
            print(f"    tags: {', '.join(t.name for t in n.tags)}")
        if n.metadata:
            print(f"    metadata: {dict(n.metadata)}")

    print("\n--- GET note (GET /api/v1/notes/{id}) ---")
    note1 = client.get_note(GetNoteRequest(id="note-1"))
    print(f"Fetched: \"{note1.title}\" -- priority={note1.priority.name}, status={note1.status.name}")
    print(f"  tags: [{', '.join(f'{t.name}({t.color})' for t in note1.tags)}]")

    print("\n--- CREATE note (POST /api/v1/notes) ---")
    created = client.create_note(
        CreateNoteRequest(
            title="Deploy to staging",
            content="Run integration tests before prod",
            priority=Priority.PRIORITY_HIGH,
            tags=[
                Tag(name="devops", color="#06b6d4"),
                Tag(name="backend", color="#3b82f6"),
            ],
            metadata={"environment": "staging", "approver": "bob"},
            due_date="2025-07-01",
        ),
        NoteServiceCallOptions(request_id="req-create-001"),
    )
    print(f"Created: {created.id} -- \"{created.title}\"")
    print(f"  priority={created.priority.name}, status={created.status.name}")
    print(f"  due_date={created.due_date}")
    print(f"  tags: {', '.join(t.name for t in created.tags)}")
    print(f"  metadata: {dict(created.metadata)}")

    print("\n--- UPDATE note (PUT /api/v1/notes/{id}) ---")
    updated = client.update_note(
        UpdateNoteRequest(
            id=created.id,
            title="Deploy to staging (approved)",
            content="Integration tests passed. Ready for deploy.",
            priority=Priority.PRIORITY_URGENT,
            status=Status.STATUS_IN_PROGRESS,
            tags=list(created.tags),
            metadata={**created.metadata, "approved": "true"},
            due_date="2025-06-30",
        ),
        NoteServiceCallOptions(idempotency_key="idem-update-001"),
    )
    print(f"Updated: \"{updated.title}\" -> priority={updated.priority.name}, status={updated.status.name}")

    print("\n--- ARCHIVE note (PATCH /api/v1/notes/{id}/archive) ---")
    archived = client.archive_note(ArchiveNoteRequest(id="note-1"))
    print(f"Archived: \"{archived.title}\" -> status={archived.status.name}")

    print("\n--- DELETE note (DELETE /api/v1/notes/{id}) ---")
    deleted = client.delete_note(DeleteNoteRequest(id=created.id))
    print(f"Deleted: success={deleted.success}")


# ============================================================================
# Section 3: Query Parameters
# ============================================================================
# ListNotes supports status, priority, sort, limit, offset as query params.

def section3_query_parameters(client: NoteServiceClient) -> None:
    print("\n========================================")
    print("  Section 3: Query Parameters")
    print("========================================\n")

    print("--- All notes (no filters) ---")
    all_notes = client.list_notes(ListNotesRequest())
    print(f"Total: {all_notes.total} notes")

    print("\n--- Filter: status=pending ---")
    pending = client.list_notes(ListNotesRequest(status="pending"))
    print(f"Found {pending.total} pending notes:")
    for n in pending.notes:
        print(f"  {n.id}: \"{n.title}\"")

    print("\n--- Filter: priority=urgent ---")
    urgent = client.list_notes(ListNotesRequest(priority="urgent"))
    print(f"Found {urgent.total} urgent notes:")
    for n in urgent.notes:
        print(f"  {n.id}: \"{n.title}\"")

    print("\n--- Pagination: limit=2, offset=0 ---")
    page1 = client.list_notes(ListNotesRequest(limit=2, offset=0))
    print(f"Page 1: {len(page1.notes)} of {page1.total} notes")
    for n in page1.notes:
        print(f"  {n.id}: \"{n.title}\"")

    print("\n--- Pagination: limit=2, offset=2 ---")
    page2 = client.list_notes(ListNotesRequest(limit=2, offset=2))
    print(f"Page 2: {len(page2.notes)} of {page2.total} notes")
    for n in page2.notes:
        print(f"  {n.id}: \"{n.title}\"")

    print("\n--- Sort by title ---")
    sorted_notes = client.list_notes(ListNotesRequest(sort="title"))
    for n in sorted_notes.notes:
        print(f"  {n.id}: \"{n.title}\"")

    print("\n--- Combined: status=pending, sort=priority, limit=1 ---")
    combined = client.list_notes(ListNotesRequest(status="pending", sort="priority", limit=1))
    print(f"Got {len(combined.notes)} of {combined.total}:")
    for n in combined.notes:
        print(f"  {n.id}: \"{n.title}\" [{n.priority.name}]")


# ============================================================================
# Section 4: Header Management
# ============================================================================
# Service headers via client options, method headers via call options,
# arbitrary per-call headers via the `headers` field on call options.

def section4_header_management(client: NoteServiceClient) -> None:
    print("\n========================================")
    print("  Section 4: Header Management")
    print("========================================\n")

    print("--- Method header: X-Request-ID on create_note ---")
    note = client.create_note(
        CreateNoteRequest(
            title="Header test",
            content="Testing headers",
            priority=Priority.PRIORITY_LOW,
        ),
        NoteServiceCallOptions(request_id="req-header-test-001"),
    )
    print(f"Created with X-Request-ID: {note.id}")

    print("\n--- Method header: X-Idempotency-Key on update_note ---")
    updated = client.update_note(
        UpdateNoteRequest(
            id=note.id,
            title="Header test (updated)",
            content="Testing idempotency",
            priority=Priority.PRIORITY_LOW,
            status=Status.STATUS_DONE,
        ),
        NoteServiceCallOptions(idempotency_key="idem-key-abc123"),
    )
    print(f"Updated with X-Idempotency-Key: \"{updated.title}\"")

    print("\n--- Per-call header override via headers field ---")
    fetched = client.get_note(
        GetNoteRequest(id=note.id),
        NoteServiceCallOptions(headers={"X-Trace-ID": "trace-12345"}),
    )
    print(f"Fetched with custom X-Trace-ID header: \"{fetched.title}\"")

    print("\n--- Per-call service header override: tenant_id ---")
    with_tenant_override = client.get_note(
        GetNoteRequest(id=note.id),
        NoteServiceCallOptions(tenant_id="99"),
    )
    print(f"Fetched with tenant_id=99 override: \"{with_tenant_override.title}\"")

    client.delete_note(DeleteNoteRequest(id=note.id))


# ============================================================================
# Section 5: Validation Errors
# ============================================================================
# buf.validate rules on CreateNoteRequest enforce title min_len=1, max_len=200.
# The Go server returns HTTP 400 with a `violations` body that the Python
# client parses into a typed ValidationError.

def section5_validation_errors(client: NoteServiceClient) -> None:
    print("\n========================================")
    print("  Section 5: Validation Errors")
    print("========================================\n")

    print("--- Validation: empty title (min_len=1) ---")
    try:
        client.create_note(
            CreateNoteRequest(title="", content="Should fail", priority=Priority.PRIORITY_LOW),
            NoteServiceCallOptions(request_id="req-val-001"),
        )
    except ValidationError as e:
        print("Caught ValidationError!")
        for v in e.violations:
            print(f"  field: \"{v.field}\" -> {v.description}")

    print("\n--- Validation: title too long (max_len=200) ---")
    try:
        client.create_note(
            CreateNoteRequest(title="A" * 201, content="Should fail", priority=Priority.PRIORITY_LOW),
            NoteServiceCallOptions(request_id="req-val-002"),
        )
    except ValidationError as e:
        print("Caught ValidationError!")
        for v in e.violations:
            print(f"  field: \"{v.field}\" -> {v.description}")

    print("\n--- Validation: missing X-Request-ID header ---")
    try:
        client.create_note(
            CreateNoteRequest(title="No request ID", content="Should fail", priority=Priority.PRIORITY_LOW),
        )
    except ValidationError as e:
        print("Caught ValidationError (missing header)!")
        for v in e.violations:
            print(f"  field: \"{v.field}\" -> {v.description}")


# ============================================================================
# Section 6: Error Handling
# ============================================================================
# Custom proto error (NotFoundError) -> 404 with a typed exception subclass
# automatically chosen from _ERROR_CLASSES based on the body shape.

def section6_error_handling(client: NoteServiceClient) -> None:
    print("\n========================================")
    print("  Section 6: Error Handling")
    print("========================================\n")

    print("--- Typed NotFoundError: get non-existent note (404) ---")
    try:
        client.get_note(GetNoteRequest(id="does-not-exist"))
    except NotFoundError as e:
        print("Caught NotFoundError (typed exception subclass)!")
        print(f"  status:       {e.status}")
        print(f"  resource_type: {e.resource_type}")
        print(f"  resource_id:   {e.resource_id}")

    print("\n--- Typed NotFoundError: delete non-existent note (404) ---")
    try:
        client.delete_note(DeleteNoteRequest(id="ghost-note"))
    except NotFoundError as e:
        print(f"NotFoundError: {e.resource_type}/{e.resource_id}")

    print("\n--- Exception hierarchy: NotFoundError IS an ApiError ---")
    try:
        client.get_note(GetNoteRequest(id="nope"))
    except Exception as e:
        print(f"  isinstance(e, NotFoundError):   {isinstance(e, NotFoundError)}")
        print(f"  isinstance(e, ApiError):        {isinstance(e, ApiError)}")
        print(f"  isinstance(e, ValidationError): {isinstance(e, ValidationError)}")
        print(f"  isinstance(e, Exception):       {isinstance(e, Exception)}")


# ============================================================================
# Section 7: Advanced Features
# ============================================================================
# Custom transport injection (logging middleware) and unwrap response (get
# notes by tag returns Note[] directly thanks to the (sebuf.http.unwrap)
# annotation on NoteList).

class LoggingTransport:
    """Wraps any HttpTransport and prints a one-line trace per request."""

    def __init__(self, inner: HttpTransport):
        self._inner = inner

    def request(self, method, url, headers, body, timeout):
        print(f"  [LOG] -> {method} {url}")
        start = time.monotonic()
        resp = self._inner.request(method, url, headers, body, timeout)
        elapsed_ms = int((time.monotonic() - start) * 1000)
        print(f"  [LOG] <- {resp.status} ({elapsed_ms}ms)")
        return resp


def section7_advanced_features() -> None:
    print("\n========================================")
    print("  Section 7: Advanced Features")
    print("========================================\n")

    print("--- Custom transport: logging interceptor ---")
    logging_client = NoteServiceClient(
        "http://localhost:3000",
        NoteServiceClientOptions(
            transport=LoggingTransport(UrllibTransport()),
            api_key="550e8400-e29b-41d4-a716-446655440000",
            tenant_id="42",
        ),
    )
    logging_client.get_note(GetNoteRequest(id="note-2"))
    print("  Logging transport captured request/response above")

    print("\n--- Unwrap response: get_notes_by_tag returns NoteList wrapping Note[] ---")
    plain_client = NoteServiceClient(
        "http://localhost:3000",
        NoteServiceClientOptions(
            api_key="550e8400-e29b-41d4-a716-446655440000",
            tenant_id="42",
        ),
    )
    backend_notes = plain_client.get_notes_by_tag(GetNotesByTagRequest(tag="backend"))
    # The (sebuf.http.unwrap) annotation on NoteList.notes collapses the
    # wire body from {"notes": [...]} to [...]. The Python class still
    # exposes the items via the .notes attribute for ergonomic access.
    print(f"get_notes_by_tag(\"backend\") returned NoteList with {len(backend_notes.notes)} item(s):")
    for n in backend_notes.notes:
        print(f"  {n.id}: \"{n.title}\" [{n.priority.name}]")

    bug_notes = plain_client.get_notes_by_tag(GetNotesByTagRequest(tag="bug"))
    print(f"\nget_notes_by_tag(\"bug\") returned {len(bug_notes.notes)} item(s):")
    for n in bug_notes.notes:
        print(f"  {n.id}: \"{n.title}\"")


# ============================================================================
# Main
# ============================================================================

def main() -> None:
    print("=== Comprehensive Python Client Demo ===")
    print("Showcases every feature of protoc-gen-py-client\n")

    client = section1_client_configuration()
    section2_crud_operations(client)
    section3_query_parameters(client)
    section4_header_management(client)
    section5_validation_errors(client)
    section6_error_handling(client)
    section7_advanced_features()

    print("\n=== Demo complete ===")


if __name__ == "__main__":
    try:
        main()
    except ApiError as e:
        print(f"\nUnexpected ApiError: status={e.status}, body={e.body!r}", file=sys.stderr)
        sys.exit(1)
