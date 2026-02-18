import {
  NoteServiceClient,
  ValidationError,
  ApiError,
  type Note,
} from "./generated/proto/note_service_client.ts";

// ============================================================================
// Section 1: Client Setup
// ============================================================================

function createClient(): NoteServiceClient {
  console.log("========================================");
  console.log("  Section 1: Client Setup");
  console.log("========================================\n");

  const client = new NoteServiceClient("http://localhost:3000", {
    apiKey: "550e8400-e29b-41d4-a716-446655440000",
    tenantId: "42",
  });
  console.log("Client created with service headers:");
  console.log("  X-API-Key:   550e8400-e29b-41d4-a716-446655440000");
  console.log("  X-Tenant-ID: 42");

  return client;
}

// ============================================================================
// Section 2: CRUD Operations
// ============================================================================

async function demoCrud(client: NoteServiceClient) {
  console.log("\n========================================");
  console.log("  Section 2: CRUD Operations");
  console.log("========================================\n");

  // LIST
  console.log("--- LIST notes (GET /api/v1/notes) ---");
  const allNotes = await client.listNotes({} as any);
  console.log(`Found ${allNotes.total} seed notes:`);
  for (const n of allNotes.notes) {
    const due = n.dueDate ? ` (due: ${n.dueDate})` : "";
    console.log(`  ${n.id}: "${n.title}" [${n.priority}, ${n.status}]${due}`);
  }

  // GET
  console.log("\n--- GET note (GET /api/v1/notes/{id}) ---");
  const note1 = await client.getNote({ id: "note-1" });
  console.log(`Fetched: "${note1.title}" - ${note1.priority}`);
  console.log(`  tags: [${note1.tags.map((t) => t.name).join(", ")}]`);

  // CREATE
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
      metadata: { environment: "staging" },
      dueDate: "2025-07-01",
    },
    { requestId: "req-001" },
  );
  console.log(`Created: ${created.id} - "${created.title}" [${created.priority}]`);

  // UPDATE
  console.log("\n--- UPDATE note (PUT /api/v1/notes/{id}) ---");
  const updated = await client.updateNote(
    {
      id: created.id,
      title: "Deploy to staging (approved)",
      content: "Integration tests passed",
      priority: "PRIORITY_URGENT",
      status: "STATUS_IN_PROGRESS",
      tags: created.tags,
      metadata: { environment: "staging", approved: "true" },
      dueDate: "2025-06-30",
    },
    { idempotencyKey: "idem-001" },
  );
  console.log(`Updated: "${updated.title}" -> ${updated.priority}, ${updated.status}`);

  // ARCHIVE
  console.log("\n--- ARCHIVE note (PATCH /api/v1/notes/{id}/archive) ---");
  const archived = await client.archiveNote({ id: "note-1" });
  console.log(`Archived: "${archived.title}" -> ${archived.status}`);

  // DELETE
  console.log("\n--- DELETE note (DELETE /api/v1/notes/{id}) ---");
  const deleted = await client.deleteNote({ id: created.id });
  console.log(`Deleted: success=${deleted.success}`);
}

// ============================================================================
// Section 3: Query Parameters & Unwrap
// ============================================================================

async function demoQueries(client: NoteServiceClient) {
  console.log("\n========================================");
  console.log("  Section 3: Query Parameters & Unwrap");
  console.log("========================================\n");

  // Filter by status
  console.log("--- Filter: status=pending ---");
  const pending = await client.listNotes({ status: "pending" } as any);
  console.log(`Found ${pending.total} pending notes:`);
  for (const n of pending.notes) {
    console.log(`  ${n.id}: "${n.title}"`);
  }

  // Pagination
  console.log("\n--- Pagination: limit=2, offset=0 ---");
  const page = await client.listNotes({ limit: 2, offset: 0 } as any);
  console.log(`Page: ${page.notes.length} of ${page.total}`);

  // Unwrap: getNotesByTag returns Note[] directly
  console.log("\n--- Unwrap: getNotesByTag -> Note[] ---");
  const backendNotes: Note[] = await client.getNotesByTag({ tag: "backend" });
  console.log(`getNotesByTag("backend") returned ${backendNotes.length} items:`);
  for (const n of backendNotes) {
    console.log(`  ${n.id}: "${n.title}"`);
  }
  console.log(`Return type is array: ${Array.isArray(backendNotes)}`);
}

// ============================================================================
// Section 4: Error Handling
// ============================================================================

async function demoErrors(client: NoteServiceClient) {
  console.log("\n========================================");
  console.log("  Section 4: Error Handling");
  console.log("========================================\n");

  // Header validation: missing required header
  console.log("--- Missing X-Request-ID header on CreateNote ---");
  try {
    await client.createNote(
      { title: "No header", content: "Should fail", priority: "PRIORITY_LOW", tags: [], metadata: {} },
      // requestId intentionally omitted
    );
  } catch (e) {
    if (e instanceof ValidationError) {
      console.log("Caught ValidationError:");
      for (const v of e.violations) {
        console.log(`  field: "${v.field}" -> ${v.description}`);
      }
    }
  }

  // Not found error
  console.log("\n--- Get non-existent note (404) ---");
  try {
    await client.getNote({ id: "does-not-exist" });
  } catch (e) {
    if (e instanceof ApiError) {
      console.log(`ApiError: status=${e.statusCode}`);
      const body = JSON.parse(e.body);
      console.log(`  resourceType: ${body.resourceType}`);
      console.log(`  resourceId: ${body.resourceId}`);
    }
  }

  // Type checking
  console.log("\n--- Error type checking ---");
  try {
    await client.getNote({ id: "nope" });
  } catch (e) {
    console.log(`  instanceof ApiError:        ${e instanceof ApiError}`);
    console.log(`  instanceof ValidationError: ${e instanceof ValidationError}`);
    console.log(`  instanceof Error:           ${e instanceof Error}`);
  }
}

// ============================================================================
// Main
// ============================================================================

async function main() {
  console.log("=== TypeScript Full-Stack Demo ===");
  console.log("TS client (protoc-gen-ts-client) -> TS server (protoc-gen-ts-server)\n");

  const client = createClient();
  await demoCrud(client);
  await demoQueries(client);
  await demoErrors(client);

  console.log("\n=== Demo complete ===");
}

main().catch(console.error);
