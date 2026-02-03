import {
  NoteServiceClient,
  ValidationError,
  ApiError,
  type Note,
} from "./generated/proto/note_service_client.ts";

const client = new NoteServiceClient("http://localhost:3000", {
  apiKey: "test-key-123",
});

async function main() {
  console.log("=== TypeScript Client Demo ===\n");

  // 1. Create notes (POST with X-Request-ID header)
  console.log("--- Creating notes ---");
  const note1 = await client.createNote(
    { title: "Buy groceries", content: "Milk, eggs, bread" },
    { requestId: "req-001" },
  );
  console.log("Created:", note1);

  const note2 = await client.createNote(
    { title: "Read book", content: "Finish chapter 5" },
    { requestId: "req-002" },
  );
  console.log("Created:", note2);

  const note3 = await client.createNote(
    { title: "Write tests", content: "Cover edge cases" },
    { requestId: "req-003" },
  );
  console.log("Created:", note3);

  // 2. List all notes (GET with no filters)
  console.log("\n--- Listing all notes ---");
  const allNotes = await client.listNotes({});
  console.log(`Found ${allNotes.total} notes:`);
  for (const n of allNotes.notes) {
    console.log(`  [${n.done ? "x" : " "}] ${n.id}: ${n.title}`);
  }

  // 3. Get a single note (GET with path param)
  console.log("\n--- Getting single note ---");
  const fetched = await client.getNote({ id: note1.id });
  console.log("Fetched:", fetched);

  // 4. Update a note (PUT with path param + body)
  console.log("\n--- Updating note ---");
  const updated = await client.updateNote({
    id: note1.id,
    title: "Buy groceries (done!)",
    content: "Milk, eggs, bread, butter",
    done: true,
  });
  console.log("Updated:", updated);

  // 5. List with query params: filter by status
  console.log("\n--- Listing done notes (query param: status=done) ---");
  const doneNotes = await client.listNotes({ status: "done" });
  console.log(`Found ${doneNotes.total} done notes:`);
  for (const n of doneNotes.notes) {
    console.log(`  [x] ${n.id}: ${n.title}`);
  }

  console.log("\n--- Listing pending notes (query param: status=pending) ---");
  const pendingNotes = await client.listNotes({ status: "pending" });
  console.log(`Found ${pendingNotes.total} pending notes:`);
  for (const n of pendingNotes.notes) {
    console.log(`  [ ] ${n.id}: ${n.title}`);
  }

  // 6. List with limit
  console.log("\n--- Listing with limit=1 ---");
  const limited = await client.listNotes({ limit: 1 });
  console.log(`Got ${limited.notes.length} of ${limited.total} notes`);

  // 7. Delete a note (DELETE with path param)
  console.log("\n--- Deleting note ---");
  const deleted = await client.deleteNote({ id: note2.id });
  console.log("Deleted:", deleted);

  // 8. Verify deletion
  console.log("\n--- Final note list ---");
  const finalNotes = await client.listNotes({});
  console.log(`${finalNotes.total} notes remaining:`);
  for (const n of finalNotes.notes) {
    console.log(`  [${n.done ? "x" : " "}] ${n.id}: ${n.title}`);
  }

  // 9. Error handling: get non-existent note
  console.log("\n--- Error handling: non-existent note ---");
  try {
    await client.getNote({ id: "does-not-exist" });
  } catch (e) {
    if (e instanceof ApiError) {
      console.log(`ApiError (${e.statusCode}): ${e.message}`);
    } else {
      console.log("Unexpected error:", e);
    }
  }

  // 10. Error handling: missing required header (X-Request-ID)
  console.log("\n--- Error handling: missing required header ---");
  try {
    // Call createNote without requestId — server requires X-Request-ID
    await client.createNote({ title: "Should fail", content: "No request ID" });
  } catch (e) {
    if (e instanceof ValidationError) {
      console.log("ValidationError:", e.violations);
    } else if (e instanceof ApiError) {
      console.log(`ApiError (${e.statusCode}): ${e.message}`);
    }
  }

  console.log("\n=== Demo complete ===");
}

main().catch(console.error);
