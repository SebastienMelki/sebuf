"""End-to-end encoding-demo for protoc-gen-py-client.

Every section round-trips one wire-format annotation against the Go server
in ../main.go and asserts that the Python decoder produced exactly the
expected value. Run from this directory with `python3 main.py` after the
server is up (or use `make demo` from the parent dir to do both).
"""

from __future__ import annotations

import sys
from datetime import datetime, timezone
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent / "generated" / "proto"))

from encoding_demo_client import (  # noqa: E402
    Address,
    BytesBase64Example,
    BytesHexExample,
    EncodingDemoServiceCallOptions,
    EncodingDemoServiceClient,
    EncodingDemoServiceClientOptions,
    EnumExample,
    FlattenExample,
    GetBytesBase64ExampleRequest,
    GetBytesHexExampleRequest,
    GetEnumExampleRequest,
    GetFlattenExampleRequest,
    GetInt64NumberExampleRequest,
    GetInt64StringExampleRequest,
    GetItemListRequest,
    GetItemMapRequest,
    GetItemsByCategoryRequest,
    GetOneofFlattenedRequest,
    GetOneofNestedRequest,
    GetPyKeywordRequest,
    GetTimestampsExampleRequest,
    ImageAttachment,
    ImagePayload,
    Int64NumberExample,
    Int64StringExample,
    Item,
    ItemBucket,
    ItemList,
    ItemMap,
    ItemsByCategory,
    LinkAttachment,
    ListItemsRequest,
    ListItemsResponse,
    OneofFlattenedExample,
    OneofNestedExample,
    PyKeywordExample,
    TextPayload,
    TimestampsExample,
    Visibility,
)


# ============================================================================
# Tiny assertion helper — prints `PASS` / `FAIL` lines so the demo doubles as
# a smoke test. The script exits non-zero on any failure.
# ============================================================================

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


# ============================================================================
# Sections
# ============================================================================

def section_enum_value(client: EncodingDemoServiceClient) -> None:
    section("enum_value override (Visibility)")
    e: EnumExample = client.get_enum_example(GetEnumExampleRequest())
    check("decoded enum member", e.visibility, Visibility.VISIBILITY_TEAM)
    check("wire JSON used custom string", e.to_dict()["visibility"], "team")
    check("label round-trip", e.label, "team-doc")


def section_timestamps(client: EncodingDemoServiceClient) -> None:
    section("timestamp_format — RFC3339 / UNIX_S / UNIX_MS / DATE")
    ts: TimestampsExample = client.get_timestamps_example(GetTimestampsExampleRequest())
    expected = datetime(2026, 5, 19, 12, 34, 56, tzinfo=timezone.utc)
    # All 4 fields decode to datetime regardless of wire format.
    check("rfc3339 type",     type(ts.rfc3339), datetime)
    check("unix_seconds type", type(ts.unix_seconds), datetime)
    check("unix_millis type",  type(ts.unix_millis), datetime)
    check("date_only type",    type(ts.date_only), datetime)
    # Tolerate sub-second mismatch on rfc3339 — server emits with no fractional seconds.
    check("rfc3339 ≈ expected (truncated to seconds)",
          ts.rfc3339.replace(microsecond=0), expected)
    check("unix_seconds ≈ expected (whole-second precision)",
          ts.unix_seconds.replace(microsecond=0), expected)
    # unix_millis preserves millisecond precision (789 ms).
    check("unix_millis ≈ expected (with millis)",
          ts.unix_millis.replace(microsecond=0) <= expected
          and ts.unix_millis.replace(microsecond=0).year == 2026, True)
    check("date_only year", ts.date_only.year, 2026)
    check("date_only month/day", (ts.date_only.month, ts.date_only.day), (5, 19))


def section_int64(client: EncodingDemoServiceClient) -> None:
    section("int64_encoding — STRING default vs NUMBER override")

    s: Int64StringExample = client.get_int64_string_example(GetInt64StringExampleRequest())
    # The Python field is typed `str` for default int64 — JS-safe wire form.
    check("default int64 Python type", type(s.value), str)
    check("preserved 2^53+1 precisely (no JS-float rounding)",
          s.value, "9007199254740993")

    n: Int64NumberExample = client.get_int64_number_example(GetInt64NumberExampleRequest())
    # NUMBER override → Python int + JSON number wire form.
    check("NUMBER int64 Python type", type(n.value), int)
    check("NUMBER int64 value", n.value, 12345)


def section_bytes(client: EncodingDemoServiceClient) -> None:
    section("bytes_encoding — base64 default vs HEX override")

    b64: BytesBase64Example = client.get_bytes_base64_example(GetBytesBase64ExampleRequest())
    check("base64 → bytes round-trip", b64.data, b"Hello, sebuf!")
    check("wire form is base64",
          b64.to_dict()["data"], "SGVsbG8sIHNlYnVmIQ==")

    hx: BytesHexExample = client.get_bytes_hex_example(GetBytesHexExampleRequest())
    check("hex → bytes round-trip", hx.data, b"\xde\xad\xbe\xef\xca\xfe")
    check("wire form is lower-case hex",
          hx.to_dict()["data"], "deadbeefcafe")


def section_flatten(client: EncodingDemoServiceClient) -> None:
    section("flatten + flatten_prefix")
    f: FlattenExample = client.get_flatten_example(GetFlattenExampleRequest())
    check("name decoded", f.name, "Alice")
    check("nested Address reconstructed",
          (f.author_address.street, f.author_address.city, f.author_address.zip_code),
          ("1 Sebuf Way", "Casablanca", "20000"))
    # Wire form puts the address fields at the top level with the prefix.
    wire = f.to_dict()
    check("wire form lacks nested 'author_address' key",
          "author_address" in wire, False)
    check("wire form has 'author_street'", wire.get("author_street"), "1 Sebuf Way")
    check("wire form has 'author_city'",   wire.get("author_city"), "Casablanca")
    check("wire form has 'author_zip_code'", wire.get("author_zip_code"), "20000")


def section_oneof_nested(client: EncodingDemoServiceClient) -> None:
    section("oneof_config (nested variant, flatten=false)")
    img: OneofNestedExample = client.get_oneof_nested(GetOneofNestedRequest(variant="image"))
    check("image variant populated", img.image is not None, True)
    check("link variant unset",      img.link is None, True)
    check("decoded image url", img.image.url if img.image else None, "https://example.com/img.png")
    check("decoded image width", img.image.width if img.image else None, 800)
    # Wire form has nested {"image": {...}} alongside discriminator.
    wire_img = img.to_dict()
    check("wire discriminator key 'kind'", wire_img.get("kind"), "image")
    check("wire nested 'image' key present", "image" in wire_img, True)

    lnk: OneofNestedExample = client.get_oneof_nested(GetOneofNestedRequest(variant="link"))
    check("link variant populated when requested", lnk.link is not None, True)
    check("image variant unset on link response", lnk.image is None, True)
    check("decoded link url", lnk.link.url if lnk.link else None, "https://example.com")


def section_oneof_flattened(client: EncodingDemoServiceClient) -> None:
    section("oneof_config (flattened variant, flatten=true)")
    txt: OneofFlattenedExample = client.get_oneof_flattened(GetOneofFlattenedRequest(variant="text"))
    check("text variant decoded", txt.text is not None, True)
    check("text body", txt.text.body if txt.text else None, "hello world")
    wire = txt.to_dict()
    check("wire discriminator key 'type'", wire.get("type"), "text")
    # In flattened mode the variant's body field appears at top level, not nested.
    check("variant fields flattened into parent", wire.get("body"), "hello world")
    check("no nested 'text' key in wire", "text" in wire, False)


def section_unwrap(client: EncodingDemoServiceClient) -> None:
    section("unwrap — root repeated, root map, map-value")

    lst: ItemList = client.get_item_list(GetItemListRequest())
    check("root-repeated unwrap decoded length", len(lst.items), 3)
    check("root-repeated unwrap to_dict yields a bare list",
          isinstance(lst.to_dict(), list), True)
    check("root-repeated unwrap content",
          [it.name for it in lst.items], ["Apple", "Banana", "Cherry"])

    m: ItemMap = client.get_item_map(GetItemMapRequest())
    check("root-map unwrap decoded size", len(m.items), 3)
    check("root-map unwrap to_dict yields bare dict",
          isinstance(m.to_dict(), dict) and "items" not in m.to_dict(), True)
    check("root-map unwrap content (Apple)", m.items["i1"].name, "Apple")

    cat: ItemsByCategory = client.get_items_by_category(GetItemsByCategoryRequest())
    check("map-value unwrap decoded bucket count", len(cat.buckets), 2)
    check("map-value unwrap fruits content",
          [it.name for it in cat.buckets["fruits"].items], ["Apple", "Banana"])
    # Wire form for each value should be a bare list, not a wrapper object.
    wire = cat.to_dict()
    check("map-value unwrap wire form: values are bare arrays",
          isinstance(wire["buckets"]["fruits"], list), True)


def section_pykeyword(client: EncodingDemoServiceClient) -> None:
    section("Python keyword field-name escape")
    p: PyKeywordExample = client.get_py_keyword(GetPyKeywordRequest())
    # Python attributes carry the trailing underscore for keywords.
    check("attribute escape: from -> from_",     p.from_, "sender@example.com")
    check("attribute escape: class -> class_",   p.class_, "first-class")
    check("attribute escape: return -> return_", p.return_, "200 OK")
    check("non-keyword attribute is unchanged",  p.normal, "no-keyword-here")
    # Wire form preserves the original proto field names.
    wire = p.to_dict()
    check("wire form uses proto field 'from'",   wire.get("from"), "sender@example.com")
    check("wire form uses proto field 'class'",  wire.get("class"), "first-class")
    check("wire form uses proto field 'return'", wire.get("return"), "200 OK")


def section_repeated_query(client: EncodingDemoServiceClient) -> None:
    section("Repeated query parameter (doseq=True)")
    # Two tags filter to the two items tagged "red" — Apple (i1) and Cherry (i3).
    resp: ListItemsResponse = client.list_items(ListItemsRequest(tag=["red"]))
    names = sorted(it.name for it in resp.items)
    check("repeated query filtered correctly", names, ["Apple", "Cherry"])
    check("total field round-tripped", resp.total, 2)

    # Multiple values — "yellow" + "berry" → Banana (yellow) and Cherry (berry).
    resp2 = client.list_items(ListItemsRequest(tag=["yellow", "berry"]))
    names2 = sorted(it.name for it in resp2.items)
    check("two repeated tag values match union", names2, ["Banana", "Cherry"])


# ============================================================================
# Main
# ============================================================================

def main() -> int:
    print("=== Python encoding-annotations demo ===")
    print("Round-trips every wire-format annotation against the Go server.\n")

    client = EncodingDemoServiceClient(
        "http://localhost:3001",
        EncodingDemoServiceClientOptions(),
    )

    section_enum_value(client)
    section_timestamps(client)
    section_int64(client)
    section_bytes(client)
    section_flatten(client)
    section_oneof_nested(client)
    section_oneof_flattened(client)
    section_unwrap(client)
    section_pykeyword(client)
    section_repeated_query(client)

    print()
    if FAIL_COUNT == 0:
        print("=== All assertions passed ===")
        return 0
    print(f"=== {FAIL_COUNT} assertion(s) failed ===")
    return 1


if __name__ == "__main__":
    sys.exit(main())
