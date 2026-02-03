import {
  NoteServiceClient,
  ValidationError,
  ApiError,
  type Note,
  type NoteServiceClientOptions,
} from "./generated/proto/note_service_client.ts";

// ============================================================================
// Section 1: Client Configuration
// ============================================================================
// The generated client accepts typed constructor options for service-level
// headers (apiKey, tenantId) and advanced options (custom fetch, defaultHeaders).

function section1_clientConfiguration(): NoteServiceClient {
  console.log("========================================");
  console.log("  Section 1: Client Configuration");
  console.log("========================================\n");

  // Service headers from proto become typed constructor options.
  // X-API-Key -> apiKey, X-Tenant-ID -> tenantId
  const options: NoteServiceClientOptions = {
    apiKey: "550e8400-e29b-41d4-a716-446655440000",
    tenantId: "42",
    defaultHeaders: {
      "X-Custom-Header": "demo-value",
    },
  };

  const client = new NoteServiceClient("http://localhost:3000", options);
  console.log("Client created with:");
  console.log("  apiKey:    550e8400-e29b-41d4-a716-446655440000 (uuid format)");
  console.log("  tenantId:  42 (integer type)");
  console.log("  + custom default header: X-Custom-Header");

  return client;
}

// ============================================================================
// Section 2: CRUD Operations
// ============================================================================
// Demonstrates all HTTP methods: GET, POST, PUT, PATCH, DELETE

async function section2_crudOperations(client: NoteServiceClient) {
  console.log("\n========================================");
  console.log("  Section 2: CRUD Operations");
  console.log("========================================\n");

  // --- LIST (GET) - server has 4 seed notes ---
  console.log("--- LIST notes (GET /api/v1/notes) ---");
  const allNotes = await client.listNotes({} as any);
  console.log(`Found ${allNotes.total} seed notes:`);
  for (const n of allNotes.notes) {
    const due = n.dueDate ? ` (due: ${n.dueDate})` : "";
    console.log(`  ${n.id}: "${n.title}" [${n.priority}, ${n.status}]${due}`);
    if (n.tags?.length > 0) {
      console.log(`    tags: ${n.tags.map((t) => t.name).join(", ")}`);
    }
    if (Object.keys(n.metadata ?? {}).length > 0) {
      console.log(`    metadata: ${JSON.stringify(n.metadata)}`);
    }
  }

  // --- GET (GET with path param) ---
  console.log("\n--- GET note (GET /api/v1/notes/{id}) ---");
  const note1 = await client.getNote({ id: "note-1" });
  console.log(`Fetched: "${note1.title}" - priority=${note1.priority}, status=${note1.status}`);
  console.log(`  tags: [${note1.tags.map((t) => `${t.name}(${t.color})`).join(", ")}]`);

  // --- CREATE (POST with body + method header: X-Request-ID) ---
  console.log("\n--- CREATE note (POST /api/v1/notes) ---");
  const created = await client.createNote(
    {
      title: "Deploy to staging",
      content: "Run integration tests before prod",
      priority: "PRIORITY_HIGH",
      tags: [
        { name: "devops", color: "#06b6d4" },
        { name: "backend", color: "#3b82f6" },
      ],
      metadata: { environment: "staging", approver: "bob" },
      dueDate: "2025-07-01",
    },
    { requestId: "req-create-001" },
  );
  console.log(`Created: ${created.id} - "${created.title}"`);
  console.log(`  priority=${created.priority}, status=${created.status}`);
  console.log(`  dueDate=${created.dueDate}`);
  console.log(`  tags: ${created.tags.map((t) => t.name).join(", ")}`);
  console.log(`  metadata: ${JSON.stringify(created.metadata)}`);

  // --- UPDATE (PUT with path param + body + method header: X-Idempotency-Key) ---
  console.log("\n--- UPDATE note (PUT /api/v1/notes/{id}) ---");
  const updated = await client.updateNote(
    {
      id: created.id,
      title: "Deploy to staging (approved)",
      content: "Integration tests passed. Ready for deploy.",
      priority: "PRIORITY_URGENT",
      status: "STATUS_IN_PROGRESS",
      tags: created.tags,
      metadata: { ...created.metadata, approved: "true" },
      dueDate: "2025-06-30",
    },
    { idempotencyKey: "idem-update-001" },
  );
  console.log(`Updated: "${updated.title}" -> priority=${updated.priority}, status=${updated.status}`);

  // --- ARCHIVE (PATCH) ---
  console.log("\n--- ARCHIVE note (PATCH /api/v1/notes/{id}/archive) ---");
  const archived = await client.archiveNote({ id: "note-1" });
  console.log(`Archived: "${archived.title}" -> status=${archived.status}`);

  // --- DELETE (DELETE with path param) ---
  console.log("\n--- DELETE note (DELETE /api/v1/notes/{id}) ---");
  const deleted = await client.deleteNote({ id: created.id });
  console.log(`Deleted: success=${deleted.success}`);
}

// ============================================================================
// Section 3: Query Parameters
// ============================================================================
// ListNotes supports: status, priority, sort, limit, offset

async function section3_queryParameters(client: NoteServiceClient) {
  console.log("\n========================================");
  console.log("  Section 3: Query Parameters");
  console.log("========================================\n");

  // No filters
  console.log("--- All notes (no filters) ---");
  const all = await client.listNotes({} as any);
  console.log(`Total: ${all.total} notes`);

  // Filter by status
  console.log("\n--- Filter: status=pending ---");
  const pending = await client.listNotes({ status: "pending" } as any);
  console.log(`Found ${pending.total} pending notes:`);
  for (const n of pending.notes) {
    console.log(`  ${n.id}: "${n.title}"`);
  }

  // Filter by priority
  console.log("\n--- Filter: priority=urgent ---");
  const urgent = await client.listNotes({ priority: "urgent" } as any);
  console.log(`Found ${urgent.total} urgent notes:`);
  for (const n of urgent.notes) {
    console.log(`  ${n.id}: "${n.title}"`);
  }

  // Pagination: limit + offset
  console.log("\n--- Pagination: limit=2, offset=0 ---");
  const page1 = await client.listNotes({ limit: 2, offset: 0 } as any);
  console.log(`Page 1: ${page1.notes.length} of ${page1.total} notes`);
  for (const n of page1.notes) {
    console.log(`  ${n.id}: "${n.title}"`);
  }

  console.log("\n--- Pagination: limit=2, offset=2 ---");
  const page2 = await client.listNotes({ limit: 2, offset: 2 } as any);
  console.log(`Page 2: ${page2.notes.length} of ${page2.total} notes`);
  for (const n of page2.notes) {
    console.log(`  ${n.id}: "${n.title}"`);
  }

  // Sort
  console.log("\n--- Sort by title ---");
  const sorted = await client.listNotes({ sort: "title" } as any);
  for (const n of sorted.notes) {
    console.log(`  ${n.id}: "${n.title}"`);
  }

  // Combined: status + sort + limit
  console.log("\n--- Combined: status=pending, sort=priority, limit=1 ---");
  const combined = await client.listNotes({
    status: "pending",
    sort: "priority",
    limit: 1,
  } as any);
  console.log(`Got ${combined.notes.length} of ${combined.total}:`);
  for (const n of combined.notes) {
    console.log(`  ${n.id}: "${n.title}" [${n.priority}]`);
  }
}

// ============================================================================
// Section 4: Header Management
// ============================================================================
// Service headers (apiKey, tenantId) via constructor,
// method headers (requestId, idempotencyKey) via call options,
// per-call overrides via headers option.

async function section4_headerManagement(client: NoteServiceClient) {
  console.log("\n========================================");
  console.log("  Section 4: Header Management");
  console.log("========================================\n");

  // Method header: requestId for CreateNote
  console.log("--- Method header: X-Request-ID on CreateNote ---");
  const note = await client.createNote(
    { title: "Header test", content: "Testing headers", priority: "PRIORITY_LOW", tags: [], metadata: {} },
    { requestId: "req-header-test-001" },
  );
  console.log(`Created with X-Request-ID: ${note.id}`);

  // Method header: idempotencyKey for UpdateNote
  console.log("\n--- Method header: X-Idempotency-Key on UpdateNote ---");
  const updatedNote = await client.updateNote(
    { id: note.id, title: "Header test (updated)", content: "Testing idempotency", priority: "PRIORITY_LOW", status: "STATUS_DONE", tags: [], metadata: {} },
    { idempotencyKey: "idem-key-abc123" },
  );
  console.log(`Updated with X-Idempotency-Key: "${updatedNote.title}"`);

  // Per-call header overrides via headers option
  console.log("\n--- Per-call header override via headers option ---");
  const fetched = await client.getNote(
    { id: note.id },
    { headers: { "X-Trace-ID": "trace-12345" } },
  );
  console.log(`Fetched with custom X-Trace-ID header: "${fetched.title}"`);

  // Override service-level tenantId per-call
  console.log("\n--- Per-call service header override: tenantId ---");
  const withTenantOverride = await client.getNote(
    { id: note.id },
    { tenantId: "99" },
  );
  console.log(`Fetched with tenantId=99 override: "${withTenantOverride.title}"`);

  // Clean up
  await client.deleteNote({ id: note.id });
}

// ============================================================================
// Section 5: Validation Errors
// ============================================================================
// buf.validate rules on CreateNoteRequest: title min_len=1, max_len=200

async function section5_validationErrors(client: NoteServiceClient) {
  console.log("\n========================================");
  console.log("  Section 5: Validation Errors");
  console.log("========================================\n");

  // Empty title (min_len: 1)
  console.log("--- Validation: empty title (min_len=1) ---");
  try {
    await client.createNote(
      { title: "", content: "Should fail", priority: "PRIORITY_LOW", tags: [], metadata: {} },
      { requestId: "req-val-001" },
    );
  } catch (e) {
    if (e instanceof ValidationError) {
      console.log("Caught ValidationError!");
      for (const v of e.violations) {
        console.log(`  field: "${v.field}" -> ${v.description}`);
      }
    }
  }

  // Title too long (max_len: 200)
  console.log("\n--- Validation: title too long (max_len=200) ---");
  try {
    await client.createNote(
      { title: "A".repeat(201), content: "Should fail", priority: "PRIORITY_LOW", tags: [], metadata: {} },
      { requestId: "req-val-002" },
    );
  } catch (e) {
    if (e instanceof ValidationError) {
      console.log("Caught ValidationError!");
      for (const v of e.violations) {
        console.log(`  field: "${v.field}" -> ${v.description}`);
      }
    }
  }

  // Missing required header: X-Request-ID
  console.log("\n--- Validation: missing X-Request-ID header ---");
  try {
    await client.createNote(
      { title: "No request ID", content: "Should fail", priority: "PRIORITY_LOW", tags: [], metadata: {} },
      // Omitting requestId intentionally
    );
  } catch (e) {
    if (e instanceof ValidationError) {
      console.log("Caught ValidationError (missing header)!");
      for (const v of e.violations) {
        console.log(`  field: "${v.field}" -> ${v.description}`);
      }
    }
  }
}

// ============================================================================
// Section 6: Error Handling
// ============================================================================
// Custom NotFoundError proto -> 404, ApiError with body parsing

async function section6_errorHandling(client: NoteServiceClient) {
  console.log("\n========================================");
  console.log("  Section 6: Error Handling");
  console.log("========================================\n");

  // Get non-existent note -> 404 with NotFoundError body
  console.log("--- ApiError: get non-existent note (404) ---");
  try {
    await client.getNote({ id: "does-not-exist" });
  } catch (e) {
    if (e instanceof ApiError) {
      console.log(`ApiError caught:`);
      console.log(`  statusCode: ${e.statusCode}`);
      console.log(`  message: ${e.message}`);
      // Parse the NotFoundError from the body
      try {
        const errorBody = JSON.parse(e.body);
        console.log(`  body (NotFoundError):`);
        console.log(`    resourceType: ${errorBody.resourceType}`);
        console.log(`    resourceId: ${errorBody.resourceId}`);
      } catch {
        console.log(`  body (raw): ${e.body}`);
      }
    }
  }

  // Delete non-existent note -> 404
  console.log("\n--- ApiError: delete non-existent note (404) ---");
  try {
    await client.deleteNote({ id: "ghost-note" });
  } catch (e) {
    if (e instanceof ApiError) {
      console.log(`ApiError: status=${e.statusCode}`);
      const errorBody = JSON.parse(e.body);
      console.log(`  NotFoundError: ${errorBody.resourceType}/${errorBody.resourceId}`);
    }
  }

  // instanceof type checking
  console.log("\n--- Error type checking with instanceof ---");
  try {
    await client.getNote({ id: "nope" });
  } catch (e) {
    console.log(`  e instanceof ApiError:        ${e instanceof ApiError}`);
    console.log(`  e instanceof ValidationError: ${e instanceof ValidationError}`);
    console.log(`  e instanceof Error:           ${e instanceof Error}`);
  }
}

// ============================================================================
// Section 7: Advanced Features
// ============================================================================
// Custom fetch with logging, AbortController, unwrap response

async function section7_advancedFeatures(client: NoteServiceClient) {
  console.log("\n========================================");
  console.log("  Section 7: Advanced Features");
  console.log("========================================\n");

  // --- Custom fetch with logging interceptor ---
  console.log("--- Custom fetch: logging interceptor ---");
  const loggingClient = new NoteServiceClient("http://localhost:3000", {
    apiKey: "550e8400-e29b-41d4-a716-446655440000",
    tenantId: "42",
    fetch: async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      const method = init?.method ?? "GET";
      console.log(`  [LOG] -> ${method} ${url}`);
      const start = Date.now();
      const resp = await globalThis.fetch(input, init);
      const elapsed = Date.now() - start;
      console.log(`  [LOG] <- ${resp.status} (${elapsed}ms)`);
      return resp;
    },
  });

  await loggingClient.getNote({ id: "note-2" });
  console.log("  Logging interceptor captured request/response above");

  // --- AbortController ---
  console.log("\n--- AbortController: abort a request ---");
  const controller = new AbortController();
  // Abort immediately before the request
  controller.abort();
  try {
    await client.getNote({ id: "note-2" }, { signal: controller.signal });
  } catch (e) {
    if (e instanceof Error) {
      const isAbort = e.name === "AbortError" || (e.cause && (e.cause as Error).name === "AbortError");
      console.log(`  Caught abort: name="${e.name}", isAbortError=${isAbort}`);
      console.log(`  message: "${e.message}"`);
    }
  }

  // --- Unwrap response: getNotesByTag returns Note[] directly ---
  console.log("\n--- Unwrap response: getNotesByTag -> Note[] ---");
  const backendNotes: Note[] = await client.getNotesByTag({ tag: "backend" });
  console.log(`getNotesByTag("backend") returned Note[] with ${backendNotes.length} items:`);
  for (const n of backendNotes) {
    console.log(`  ${n.id}: "${n.title}" [${n.priority}]`);
  }
  console.log(`Return type is array: ${Array.isArray(backendNotes)}`);

  const bugNotes = await client.getNotesByTag({ tag: "bug" });
  console.log(`\ngetNotesByTag("bug") returned ${bugNotes.length} item(s):`);
  for (const n of bugNotes) {
    console.log(`  ${n.id}: "${n.title}"`);
  }
}

// ============================================================================
// Main
// ============================================================================

async function main() {
  console.log("=== Comprehensive TypeScript Client Demo ===");
  console.log("Showcases every feature of protoc-gen-ts-client\n");

  const client = section1_clientConfiguration();
  await section2_crudOperations(client);
  await section3_queryParameters(client);
  await section4_headerManagement(client);
  await section5_validationErrors(client);
  await section6_errorHandling(client);
  await section7_advancedFeatures(client);

  console.log("\n=== Demo complete ===");
}

main().catch(console.error);
