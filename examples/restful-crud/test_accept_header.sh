#!/bin/bash
# Regression test for #173: Response format negotiation must use Accept header.
# Per HTTP semantics (RFC 9110), the Accept header governs response format.
# Previously, the server used request Content-Type for response format, making
# it impossible to send JSON and receive protobuf (or vice versa).

set -e

PORT=18173
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
go build -o /tmp/accept-header-test-server . 2>/dev/null

echo "Starting server on :$PORT..."
PORT=$PORT /tmp/accept-header-test-server >/dev/null 2>&1 &
SERVER_PID=$!
trap "kill $SERVER_PID 2>/dev/null; rm -f /tmp/accept-header-test-server" EXIT

# Wait for server
for i in $(seq 1 30); do
    if curl -s -o /dev/null "http://localhost:$PORT/api/v1/products?page=1&limit=1" \
        -H "X-API-Key: $API_KEY" 2>/dev/null; then
        break
    fi
    sleep 0.5
done

echo ""
echo "=== #173 Regression: Accept header controls response format ==="
echo ""

# Test 1: Accept: application/json — should return JSON with application/json Content-Type
echo "Test 1: Accept: application/json"
CT=$(curl -s -o /dev/null -w '%{content_type}' -X GET \
    "http://localhost:$PORT/api/v1/products/$PRODUCT_ID" \
    -H "X-API-Key: $API_KEY" \
    -H "Accept: application/json")
check "Accept: application/json → response Content-Type is JSON" "$CT" "application/json"

# Test 2: Accept: application/x-protobuf — should return protobuf binary
echo "Test 2: Accept: application/x-protobuf"
CT=$(curl -s -o /dev/null -w '%{content_type}' -X GET \
    "http://localhost:$PORT/api/v1/products/$PRODUCT_ID" \
    -H "X-API-Key: $API_KEY" \
    -H "Accept: application/x-protobuf")
check "Accept: application/x-protobuf → response is protobuf" "$CT" "application/x-protobuf"

# Test 3: Send JSON body + Accept protobuf — the core bug from #173
# Before the fix, this returned JSON because it mirrored request Content-Type.
echo "Test 3: Send JSON body + Accept protobuf (the #173 bug)"
CT=$(curl -s -o /dev/null -w '%{content_type}' -X POST \
    "http://localhost:$PORT/api/v1/products" \
    -H "Content-Type: application/json" \
    -H "Accept: application/x-protobuf" \
    -H "X-API-Key: $API_KEY" \
    -d '{"name": "Test Product", "price": 1.00, "stock_quantity": 1}')
check "JSON request + Accept: protobuf → response is protobuf" "$CT" "application/x-protobuf"

# Test 4: No Accept header, JSON Content-Type — should default to matching Content-Type (JSON)
echo "Test 4: No Accept header (fallback to Content-Type)"
CT=$(curl -s -o /dev/null -w '%{content_type}' -X POST \
    "http://localhost:$PORT/api/v1/products" \
    -H "Content-Type: application/json" \
    -H "X-API-Key: $API_KEY" \
    -d '{"name": "Test Fallback", "price": 2.00, "stock_quantity": 1}')
check "No Accept + JSON Content-Type → response is JSON" "$CT" "application/json"

# Test 5: Accept: */* — should fall back to Content-Type
echo "Test 5: Accept: */* (wildcard fallback)"
CT=$(curl -s -o /dev/null -w '%{content_type}' -X GET \
    "http://localhost:$PORT/api/v1/products/$PRODUCT_ID" \
    -H "X-API-Key: $API_KEY" \
    -H "Accept: */*")
check "Accept: */* → response defaults to JSON" "$CT" "application/json"

# Test 6: Accept: text/html (unknown) — should default to JSON
echo "Test 6: Accept: text/html (unrecognized)"
CT=$(curl -s -o /dev/null -w '%{content_type}' -X GET \
    "http://localhost:$PORT/api/v1/products/$PRODUCT_ID" \
    -H "X-API-Key: $API_KEY" \
    -H "Accept: text/html")
check "Accept: text/html → response defaults to JSON" "$CT" "application/json"

echo ""
if [ $FAIL -eq 0 ]; then
    echo "All tests passed."
else
    echo "SOME TESTS FAILED."
    exit 1
fi
