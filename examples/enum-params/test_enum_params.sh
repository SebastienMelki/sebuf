#!/bin/bash
# Regression test for #151: Enum fields as query and path parameters.
# Verifies enum values are accepted by name, by number, and rejected when invalid.

set -e

PORT=18151
FAIL=0

pass() { echo "  [PASS] $1"; }
fail() { echo "  [FAIL] $1 (expected: $2, got: $3)"; FAIL=1; }
check() {
    local label="$1" actual="$2" expected="$3"
    if [ "$actual" = "$expected" ]; then pass "$label"; else fail "$label" "$expected" "$actual"; fi
}
check_contains() {
    local label="$1" actual="$2" expected="$3"
    if echo "$actual" | grep -q "$expected"; then pass "$label"; else fail "$label" "contains '$expected'" "$actual"; fi
}

echo "Building server..."
go build -o /tmp/enum-params-test-server . 2>/dev/null

echo "Starting server on :$PORT..."
PORT=$PORT /tmp/enum-params-test-server >/dev/null 2>&1 &
SERVER_PID=$!
trap "kill $SERVER_PID 2>/dev/null; rm -f /tmp/enum-params-test-server" EXIT

# Wait for server
for i in $(seq 1 30); do
    if curl -s -o /dev/null "http://localhost:$PORT/api/v1/portfolio?timeframe=TIMEFRAME_1D" 2>/dev/null; then
        break
    fi
    sleep 0.5
done

echo ""
echo "=== #151 Regression: Enum query & path parameters ==="
echo ""

# Test 1: Enum query param by name
echo "Test 1: Enum query param by name"
STATUS=$(curl -s -o /dev/null -w '%{http_code}' \
    "http://localhost:$PORT/api/v1/portfolio?timeframe=TIMEFRAME_1D")
check "GET /portfolio?timeframe=TIMEFRAME_1D -> 200" "$STATUS" "200"

# Test 2: Enum query param by number
echo "Test 2: Enum query param by number"
STATUS=$(curl -s -o /dev/null -w '%{http_code}' \
    "http://localhost:$PORT/api/v1/portfolio?timeframe=3")
check "GET /portfolio?timeframe=3 -> 200" "$STATUS" "200"

# Test 3: Enum query param invalid name
echo "Test 3: Enum query param invalid name"
STATUS=$(curl -s -o /dev/null -w '%{http_code}' \
    "http://localhost:$PORT/api/v1/portfolio?timeframe=INVALID")
check "GET /portfolio?timeframe=INVALID -> 400" "$STATUS" "400"

# Test 4: Enum path param by name
echo "Test 4: Enum path param by name"
BODY=$(curl -s "http://localhost:$PORT/api/v1/portfolio/asset-class/ASSET_CLASS_EQUITY")
STATUS=$(curl -s -o /dev/null -w '%{http_code}' \
    "http://localhost:$PORT/api/v1/portfolio/asset-class/ASSET_CLASS_EQUITY")
check "GET /portfolio/asset-class/ASSET_CLASS_EQUITY -> 200" "$STATUS" "200"
check_contains "Response contains equity holdings" "$BODY" "AAPL"

# Test 5: Enum path param by number
echo "Test 5: Enum path param by number"
STATUS=$(curl -s -o /dev/null -w '%{http_code}' \
    "http://localhost:$PORT/api/v1/portfolio/asset-class/4")
check "GET /portfolio/asset-class/4 (CRYPTO) -> 200" "$STATUS" "200"

# Test 6: Enum path param invalid name
echo "Test 6: Enum path param invalid name"
STATUS=$(curl -s -o /dev/null -w '%{http_code}' \
    "http://localhost:$PORT/api/v1/portfolio/asset-class/BOGUS")
check "GET /portfolio/asset-class/BOGUS -> 400" "$STATUS" "400"

# Test 7: Combined enum path + query params
echo "Test 7: Combined enum path + query params"
STATUS=$(curl -s -o /dev/null -w '%{http_code}' \
    "http://localhost:$PORT/api/v1/portfolio/asset-class/ASSET_CLASS_CRYPTO?timeframe=TIMEFRAME_ALL")
check "GET /portfolio/asset-class/ASSET_CLASS_CRYPTO?timeframe=TIMEFRAME_ALL -> 200" "$STATUS" "200"

# Test 8: Empty query param treated as unset
echo "Test 8: Empty query param treated as unset"
STATUS=$(curl -s -o /dev/null -w '%{http_code}' \
    "http://localhost:$PORT/api/v1/portfolio?timeframe=")
check "GET /portfolio?timeframe= -> 200 (empty = unset)" "$STATUS" "200"

echo ""
if [ $FAIL -eq 0 ]; then
    echo "All tests passed."
else
    echo "SOME TESTS FAILED."
    exit 1
fi
