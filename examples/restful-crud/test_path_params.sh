#!/bin/bash
# Regression test for #176: Path parameter values must survive body unmarshaling.
# protojson.Unmarshal calls proto.Reset() which wipes the message. The fix binds
# body first, then overlays path/query params so URL-stated values always win.
#
# Key insight: the path param (product_id) comes from the URL, NOT from the JSON
# body. If the body contained product_id it would be overwritten by the URL value.
# Before the fix, the body unmarshal wiped the path-bound product_id to "".

set -e

PORT=18176
API_KEY="123e4567-e89b-12d3-a456-426614174000"
PRODUCT_ID="123e4567-e89b-12d3-a456-426614174001"
FAIL=0

pass() { echo "  [PASS] $1"; }
fail() { echo "  [FAIL] $1 (expected: $2, got: $3)"; FAIL=1; }
check() {
    local label="$1" actual="$2" expected="$3"
    if [ "$actual" = "$expected" ]; then pass "$label"; else fail "$label" "$expected" "$actual"; fi
}

echo "Building server..."
go build -o /tmp/restful-crud-test-server . 2>/dev/null

echo "Starting server on :$PORT..."
PORT=$PORT /tmp/restful-crud-test-server >/dev/null 2>&1 &
SERVER_PID=$!
trap "kill $SERVER_PID 2>/dev/null; rm -f /tmp/restful-crud-test-server" EXIT

# Wait for server
for i in $(seq 1 30); do
    if curl -s -o /dev/null "http://localhost:$PORT/api/v1/products?page=1&limit=1" \
        -H "X-API-Key: $API_KEY" 2>/dev/null; then
        break
    fi
    sleep 0.5
done

echo ""
echo "=== #176 Regression: Path params survive body unmarshaling ==="
echo ""

# Test 1: PUT — body does NOT include product_id, it comes only from the URL path.
# Before the fix, protojson.Unmarshal({...}) would Reset() the message, wiping the
# path-bound product_id to "". The handler would then fail with "product not found: ".
echo "Test 1: PUT with path param (product_id from URL only)"
RESP=$(curl -s -X PUT "http://localhost:$PORT/api/v1/products/$PRODUCT_ID" \
    -H "Content-Type: application/json" \
    -H "X-API-Key: $API_KEY" \
    -d '{"name": "Updated Name", "price": 42.00, "stock_quantity": 10}')
GOT_ID=$(echo "$RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)
check "PUT: product_id from URL preserved after body unmarshal" "$GOT_ID" "$PRODUCT_ID"

# Test 2: PUT with body that also sets product_id — URL must win.
echo "Test 2: PUT with conflicting product_id in body (URL must win)"
RESP=$(curl -s -X PUT "http://localhost:$PORT/api/v1/products/$PRODUCT_ID" \
    -H "Content-Type: application/json" \
    -H "X-API-Key: $API_KEY" \
    -d "{\"name\": \"Conflict Test\", \"price\": 1.00, \"stock_quantity\": 1, \"product_id\": \"wrong-id\"}")
GOT_ID=$(echo "$RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)
check "PUT: URL path param overrides conflicting body value" "$GOT_ID" "$PRODUCT_ID"

# Test 3: PATCH — partial body, product_id only from URL.
echo "Test 3: PATCH with path param + partial body"
RESP=$(curl -s -X PATCH "http://localhost:$PORT/api/v1/products/$PRODUCT_ID" \
    -H "Content-Type: application/json" \
    -H "X-API-Key: $API_KEY" \
    -d '{"name": "Patched", "price": 99.99}')
GOT_ID=$(echo "$RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)
check "PATCH: product_id from URL preserved after body unmarshal" "$GOT_ID" "$PRODUCT_ID"

# Test 4: GET — baseline (no body, path param only).
echo "Test 4: GET with path param (baseline, no body)"
RESP=$(curl -s -X GET "http://localhost:$PORT/api/v1/products/$PRODUCT_ID" \
    -H "X-API-Key: $API_KEY")
GOT_ID=$(echo "$RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)
check "GET: path param works (baseline)" "$GOT_ID" "$PRODUCT_ID"

# Test 5: PUT without body — empty body fast-path.
echo "Test 5: PUT without body (no Content-Length)"
RESP=$(curl -s -X PUT "http://localhost:$PORT/api/v1/products/$PRODUCT_ID" \
    -H "X-API-Key: $API_KEY")
# This should either succeed (empty body skipped) or fail validation, but NOT with "product not found: "
GOT_MSG=$(echo "$RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('id','') or 'validation_error' if 'violations' in d else '')" 2>/dev/null)
if [ "$GOT_MSG" = "validation_error" ] || [ "$GOT_MSG" = "$PRODUCT_ID" ]; then
    pass "PUT without body: path param not wiped (got validation error or success)"
else
    fail "PUT without body: unexpected response" "validation_error or $PRODUCT_ID" "$GOT_MSG"
fi

echo ""
if [ $FAIL -eq 0 ]; then
    echo "All tests passed."
else
    echo "SOME TESTS FAILED."
    exit 1
fi
