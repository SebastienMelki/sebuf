// Wire-conformance proof for the protobuf-es TS server.
//
// The client-side proof (internal/tsclientgen/testdata/es/conformance.test.mjs)
// covers the DECODE direction: a Go server's protojson body must materialize
// correctly through fromJson. This file covers the ENCODE direction, which is
// what an es-mode *server* actually does: a handler returns a message built
// with create(), the generated route encodes it with toJson, and that output is
// the body every sebuf client — Go, Python or TS — has to be able to read.
//
// Nothing here asserts protobuf-es's behaviour in isolation; every assertion is
// pinned against server_response.canonical.json, the shape Go's protojson would
// have produced for the same message. If a protobuf-es upgrade changed how it
// omits zero values, encodes 64-bit ints, or renders enums, the es-mode server
// would silently start speaking a different wire than the Go server, and only a
// test in this direction would catch it.
//
// It asserts five things against ServerResponse (see conformance.proto):
//   1. toJson(create(...)) equals the canonical body EXACTLY — the whole
//      contract in one comparison.
//   2. Every zero-valued field is OMITTED, not emitted as "" / 0 / false / []
//      / {} — checked field-by-field so a failure names the culprit.
//   3. 64-bit ints leave as JSON STRINGS and survive past 2^53 exactly.
//   4. Enums leave as their NAME string, and the zero value is omitted.
//   5. The canonical body round-trips: decoding it and re-encoding reproduces
//      it, so server output is stable under a client's decode/encode cycle.
//
// How to run
// ----------
// This file does bare `import`s of @bufbuild/protobuf (via the generated
// conformance_pb.js), so node must be able to resolve @bufbuild/protobuf@2.12.1
// from a node_modules in this file's directory tree. The simplest way is a
// symlink (git-ignored) pointing at the Task-1 spike install:
//
//   ln -s ../../../../.scratch/es-spike/node_modules \
//         internal/tsservergen/testdata/es/node_modules
//   node internal/tsservergen/testdata/es/conformance.test.mjs
//
// The Go wrapper (conformance_test.go) creates and removes that symlink
// automatically and SKIPS cleanly when node or @bufbuild/protobuf is absent.
//
// Exit code 0 = all assertions passed; non-zero = a failure was printed.

import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";
import assert from "node:assert/strict";

import { create, fromJson, toJson } from "@bufbuild/protobuf";
import { ServerResponseSchema, Status } from "./conformance_pb.js";

const here = dirname(fileURLToPath(import.meta.url));

// The body a Go server would emit for the message built in makeResponse().
const canonical = JSON.parse(
  readFileSync(join(here, "server_response.canonical.json"), "utf8"),
);

// The single message under test, as a handler would return it. Only these five
// fields are set; every other field on ServerResponse stays at its zero value
// and must not reach the wire.
function makeResponse() {
  return create(ServerResponseSchema, {
    id: "note-123",
    bigTotal: 9223372036854775807n,
    item: { sku: "widget-1" }, // qty left zero
    attributes: { region: "eu-west" },
    status: Status.ACTIVE,
  });
}

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

console.log("wire-conformance: protobuf-es ServerResponse (encode)");

// --- Assertion 1: the encode matches the canonical body exactly -------------
check("toJson(create(...)) equals the canonical server body", () => {
  const encoded = toJson(ServerResponseSchema, makeResponse());
  assert.deepEqual(
    encoded,
    canonical,
    "encoded body should equal the JSON a Go server would emit",
  );
});

// --- Assertion 2: zero-valued fields are omitted ----------------------------
check("zero-valued fields are omitted from the encoded body", () => {
  const encoded = toJson(ServerResponseSchema, makeResponse());

  // Named individually so a regression points at the exact field that leaked.
  for (const field of [
    "name",
    "count",
    "active",
    "ratio",
    "labels",
    "tags",
    "detail",
    "fallbackStatus",
  ]) {
    assert.ok(!(field in encoded), `zero-valued ${field} must be omitted`);
  }

  // The nested message omits its own zero field rather than emitting qty: 0.
  assert.deepEqual(encoded.item, { sku: "widget-1" }, "nested zero field must be omitted");
});

// --- Assertion 3: 64-bit ints encode as strings, without precision loss -----
check("64-bit ints encode as strings and survive past 2^53", () => {
  const encoded = toJson(ServerResponseSchema, makeResponse());

  assert.equal(typeof encoded.bigTotal, "string", "int64 must encode as a JSON string");
  assert.equal(encoded.bigTotal, "9223372036854775807", "int64 max must encode exactly");

  // Guard the guard: prove the value genuinely exceeds double precision, so
  // this assertion would have caught a number-based encode rather than merely
  // restating it.
  assert.notEqual(
    String(Number(encoded.bigTotal)),
    encoded.bigTotal,
    "int64 max must be corrupted by a number round-trip (else this test proves nothing)",
  );
});

// --- Assertion 4: enums encode as names, zero value omitted -----------------
check("enums encode as their name string and the zero value is omitted", () => {
  const encoded = toJson(ServerResponseSchema, makeResponse());

  assert.equal(encoded.status, "STATUS_ACTIVE", "enum must encode as its proto name");
  assert.notEqual(encoded.status, 1, "enum must not encode as a number");
  // fallback_status was left at STATUS_UNSPECIFIED (0) and must be omitted.
  assert.ok(!("fallbackStatus" in encoded), "zero-valued enum must be omitted");
});

// --- Assertion 5: the canonical body round-trips ----------------------------
check("canonical body survives a decode/re-encode cycle unchanged", () => {
  const decoded = fromJson(ServerResponseSchema, canonical, {
    ignoreUnknownFields: true,
  });
  const reencoded = toJson(ServerResponseSchema, decoded);
  assert.deepEqual(
    reencoded,
    canonical,
    "server output should be stable through a client decode/encode cycle",
  );

  // And the decode agrees with the message the server built in the first place.
  assert.deepEqual(
    reencoded,
    toJson(ServerResponseSchema, makeResponse()),
    "round-tripped body should match the freshly encoded one",
  );
});

if (failures > 0) {
  console.error(`\nwire-conformance: ${failures} assertion(s) FAILED`);
  process.exit(1);
}
console.log("\nwire-conformance: all assertions passed");
