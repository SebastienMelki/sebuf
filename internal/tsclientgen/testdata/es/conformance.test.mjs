// Wire-conformance proof for the protobuf-es TS client.
//
// This is the guarantee behind ts_runtime=protobuf-es: a JSON body produced by
// the Go server (default protojson, which OMITS zero-valued scalars and empty
// lists) decodes through protobuf-es into a fully-materialized message, and is
// forward-compatible with fields the client does not yet know about.
//
// It asserts three things against ConformanceResponse (see conformance.proto):
//   1. fromJson(schema, canonical, { ignoreUnknownFields: true }) MATERIALIZES
//      every omitted zero-value: scalars = "" / 0, bool = false, int64 = 0n,
//      lists = []. The one present field (`id`) round-trips unchanged.
//   2. Re-serialising with toJson yields the SAME canonical form (zero-values
//      omitted again) — superset-consistent with what the server sent.
//   3. fromJson with an EXTRA unknown field does NOT throw when
//      ignoreUnknownFields is set (and DOES throw without it) — proving the
//      client tolerates server fields added in the future.
//
// How to run
// ----------
// This file does bare `import`s of @bufbuild/protobuf (via the generated
// conformance_pb.js), so node must be able to resolve @bufbuild/protobuf@2.12.1
// from a node_modules in this file's directory tree. The simplest way is a
// symlink (git-ignored) pointing at the Task-1 spike install:
//
//   ln -s ../../../../.scratch/es-spike/node_modules \
//         internal/tsclientgen/testdata/es/node_modules
//   node internal/tsclientgen/testdata/es/conformance.test.mjs
//
// The Go wrapper (conformance_test.go) creates and removes that symlink
// automatically and SKIPS cleanly when node or @bufbuild/protobuf is absent.
//
// Exit code 0 = all assertions passed; non-zero = a failure was printed.

import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";
import assert from "node:assert/strict";

import { fromJson, toJson } from "@bufbuild/protobuf";
import { ConformanceResponseSchema } from "./conformance_pb.js";

const here = dirname(fileURLToPath(import.meta.url));

// The canonical body as the Go server would emit it: only the non-zero `id`.
// Every other field (scalars, bool, int64, both repeated fields) is OMITTED.
const canonical = JSON.parse(
  readFileSync(join(here, "conformance_response.canonical.json"), "utf8"),
);

let failures = 0;
function check(label, fn) {
  try {
    fn();
    console.log(`  ok   - ${label}`);
  } catch (err) {
    failures++;
    console.error(`  FAIL - ${label}`);
    console.error(`         ${err && err.message ? err.message : err}`);
  }
}

console.log("wire-conformance: protobuf-es ConformanceResponse");

// --- Assertion 1: defaults are materialized --------------------------------
check("zero-values omitted by the server are materialized after fromJson", () => {
  const msg = fromJson(ConformanceResponseSchema, canonical, {
    ignoreUnknownFields: true,
  });

  // Present field round-trips unchanged.
  assert.equal(msg.id, "note-123", "id should round-trip");

  // Omitted scalars materialize to their zero value.
  assert.equal(msg.name, "", 'string default should be ""');
  assert.equal(msg.count, 0, "int32 default should be 0");
  assert.equal(msg.active, false, "bool default should be false");
  assert.equal(msg.ratio, 0, "double default should be 0");

  // int64 materializes as a bigint zero, not undefined.
  assert.equal(typeof msg.total, "bigint", "int64 should be a bigint");
  assert.equal(msg.total, 0n, "int64 default should be 0n");

  // Omitted repeated fields materialize as empty arrays, not undefined.
  assert.ok(Array.isArray(msg.labels), "labels should be an array");
  assert.deepEqual(msg.labels, [], "repeated scalar default should be []");
  assert.ok(Array.isArray(msg.tags), "tags should be an array");
  assert.deepEqual(msg.tags, [], "repeated message default should be []");
});

// --- Assertion 2: re-serialization is canonical (zero-values omitted) ------
check("toJson re-emits the canonical form (zero-values omitted)", () => {
  const msg = fromJson(ConformanceResponseSchema, canonical, {
    ignoreUnknownFields: true,
  });
  const reencoded = toJson(ConformanceResponseSchema, msg);

  // toJson must not leak the materialized defaults back onto the wire.
  assert.deepEqual(
    reencoded,
    canonical,
    "re-encoded JSON should equal the canonical body the server sent",
  );
});

// --- Assertion 3: unknown fields are tolerated -----------------------------
check("unknown server field is ignored with ignoreUnknownFields", () => {
  const withUnknown = { ...canonical, future_field: { nested: [1, 2, 3] }, extra: "x" };

  // Must NOT throw with the flag the es-mode client always sets.
  const msg = fromJson(ConformanceResponseSchema, withUnknown, {
    ignoreUnknownFields: true,
  });
  assert.equal(msg.id, "note-123", "known fields still decode alongside unknown ones");

  // And it MUST throw without the flag — proving the flag is what makes the
  // client forward-compatible (not merely lax input).
  assert.throws(
    () => fromJson(ConformanceResponseSchema, withUnknown),
    "fromJson without ignoreUnknownFields should reject unknown fields",
  );
});

if (failures > 0) {
  console.error(`\nwire-conformance: ${failures} assertion(s) FAILED`);
  process.exit(1);
}
console.log("\nwire-conformance: all assertions passed");
