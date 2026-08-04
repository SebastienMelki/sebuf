// Wire-conformance proof for the protobuf-es TS client.
//
// This is the guarantee behind ts_runtime=protobuf-es: a JSON body produced by
// the Go server (default protojson, which OMITS zero-valued scalars and empty
// lists) decodes through protobuf-es into a fully-materialized message, and is
// forward-compatible with fields the client does not yet know about.
//
// It asserts six things against ConformanceResponse (see conformance.proto):
//   1. fromJson(schema, canonical, { ignoreUnknownFields: true }) MATERIALIZES
//      every omitted zero-value: scalars = "" / 0, bool = false, int64 = 0n,
//      lists = [], maps = {}. The present fields round-trip unchanged.
//   2. Re-serialising with toJson yields the SAME canonical form (zero-values
//      omitted again) — superset-consistent with what the server sent.
//   3. fromJson with an EXTRA unknown field does NOT throw when
//      ignoreUnknownFields is set (and DOES throw without it) — proving the
//      client tolerates server fields added in the future.
//   4. 64-bit fields beyond the 2^53 double limit decode EXACTLY as bigint, and
//      would have been corrupted had they been decoded as numbers.
//   5. A present-but-partially-populated nested message materializes its own
//      omitted fields rather than leaving them undefined.
//   6. Populated maps (string- and message-valued) round-trip, and an omitted
//      map materializes as {}.
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

  // An omitted map materializes as {}, not undefined — the map analogue of the
  // empty-list case above.
  assert.ok(
    msg.emptyAttributes && typeof msg.emptyAttributes === "object",
    "emptyAttributes should be an object",
  );
  assert.deepEqual(msg.emptyAttributes, {}, "omitted map default should be {}");
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

// --- Assertion 4: 64-bit precision boundary --------------------------------
check("64-bit fields beyond 2^53 decode exactly as bigint", () => {
  const msg = fromJson(ConformanceResponseSchema, canonical, {
    ignoreUnknownFields: true,
  });

  // int64 / uint64 max, which no JavaScript number can hold.
  assert.equal(typeof msg.bigTotal, "bigint", "int64 should decode to bigint");
  assert.equal(msg.bigTotal, 9223372036854775807n, "int64 max must survive exactly");
  assert.equal(typeof msg.bigUnsigned, "bigint", "uint64 should decode to bigint");
  assert.equal(msg.bigUnsigned, 18446744073709551615n, "uint64 max must survive exactly");

  // Guard the guard: prove these values genuinely exceed double precision, so
  // this assertion would have caught a number-based decode rather than merely
  // restating it. Number() collapses both to a different value.
  assert.notEqual(
    BigInt(Number(canonical.bigTotal)),
    msg.bigTotal,
    "int64 max must be corrupted by a number round-trip (else this test proves nothing)",
  );
  assert.notEqual(
    BigInt(Number(canonical.bigUnsigned)),
    msg.bigUnsigned,
    "uint64 max must be corrupted by a number round-trip (else this test proves nothing)",
  );

  // The wire form is a STRING (protojson's 64-bit contract), not a JSON number.
  assert.equal(typeof canonical.bigTotal, "string", "int64 crosses the wire as a string");
  const reencoded = toJson(ConformanceResponseSchema, msg);
  assert.equal(typeof reencoded.bigTotal, "string", "toJson must re-emit int64 as a string");
  assert.equal(reencoded.bigTotal, canonical.bigTotal, "int64 must re-emit byte-identically");
});

// --- Assertion 5: partially-populated nested message -----------------------
check("nested message materializes its own omitted fields", () => {
  const msg = fromJson(ConformanceResponseSchema, canonical, {
    ignoreUnknownFields: true,
  });

  assert.ok(msg.detail, "detail should be present");
  // The populated nested field round-trips.
  assert.equal(msg.detail.label, "primary", "populated nested field should round-trip");
  // The omitted nested fields materialize, exactly as at the top level.
  assert.equal(msg.detail.note, "", 'omitted nested string should be ""');
  assert.equal(msg.detail.weight, 0, "omitted nested int32 should be 0");

  // A nested message left out entirely stays undefined — presence is preserved
  // for message fields, unlike scalars. This is the distinction that makes
  // partially-populated nested messages worth pinning separately.
  const withoutDetail = { ...canonical };
  delete withoutDetail.detail;
  const bare = fromJson(ConformanceResponseSchema, withoutDetail, {
    ignoreUnknownFields: true,
  });
  assert.equal(bare.detail, undefined, "an absent message field should stay undefined");
});

// --- Assertion 6: maps ------------------------------------------------------
check("populated string- and message-valued maps round-trip", () => {
  const msg = fromJson(ConformanceResponseSchema, canonical, {
    ignoreUnknownFields: true,
  });

  // Scalar-valued map decodes to a plain object.
  assert.deepEqual(
    msg.attributes,
    { region: "eu-west", tier: "gold" },
    "string map should round-trip",
  );

  // Message-valued map decodes to nested messages keyed by the map key.
  assert.deepEqual(Object.keys(msg.tagByKey), ["alpha"], "message map should keep its key");
  assert.equal(msg.tagByKey.alpha.name, "first", "message map value should round-trip");

  // Both re-emit in canonical form.
  const reencoded = toJson(ConformanceResponseSchema, msg);
  assert.deepEqual(reencoded.attributes, canonical.attributes, "string map must re-emit as sent");
  assert.deepEqual(reencoded.tagByKey, canonical.tagByKey, "message map must re-emit as sent");
  // The empty map must NOT leak back onto the wire as {}.
  assert.ok(
    !("emptyAttributes" in reencoded),
    "an empty map must be omitted by toJson, not emitted as {}",
  );
});

if (failures > 0) {
  console.error(`\nwire-conformance: ${failures} assertion(s) FAILED`);
  process.exit(1);
}
console.log("\nwire-conformance: all assertions passed");
