#!/usr/bin/env bash
# test-ralph-hook.sh — Integration test for ralph-hook.sh → ws-mcp bridge round-trip.
# Starts the bridge, fires a test event via ralph-hook.sh, queries /rpc events.latest.
set -euo pipefail

PORT="${WS_MCP_PORT:-8080}"
BASE="http://localhost:${PORT}"
BRIDGE_PID=""

cleanup() {
  if [ -n "$BRIDGE_PID" ]; then
    kill "$BRIDGE_PID" 2>/dev/null
    wait "$BRIDGE_PID" 2>/dev/null || true
  fi
  rm -f /tmp/ws_mcp_ralph_test
}
trap cleanup EXIT

cd "$(dirname "$0")/.."

echo "==> Building bridge..."
go build -o /tmp/ws_mcp_ralph_test ./src/cmd/bridge

echo "==> Starting bridge on port ${PORT}..."
/tmp/ws_mcp_ralph_test &
BRIDGE_PID=$!
sleep 1

# Healthcheck
STATUS=$(curl -s -o /dev/null -w '%{http_code}' "${BASE}/healthz")
if [ "$STATUS" != "200" ]; then
  echo "FAIL: healthcheck returned ${STATUS}"
  exit 1
fi
echo "    Bridge healthy"

# Fire a test event via ralph-hook.sh
echo "==> Posting test event via ralph-hook.sh..."
./scripts/ralph-hook.sh "test.ralph_hook" "ralph" '{"test": true, "origin": "ralph-hook-test"}'
echo "    Posted"

# Query events.latest via JSON-RPC /rpc
echo "==> Querying /rpc events.latest..."
RESPONSE=$(curl -sf -X POST "${BASE}/rpc" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"events.latest","params":{"limit":5},"id":1}')

# Check the response contains our event
if echo "$RESPONSE" | grep -q "test.ralph_hook"; then
  echo "    OK — event found in events.latest"
else
  echo "FAIL: event not found in response:"
  echo "$RESPONSE"
  exit 1
fi

if echo "$RESPONSE" | grep -q '"source":"ralph"'; then
  echo "    OK — source is 'ralph'"
else
  echo "FAIL: source mismatch in response:"
  echo "$RESPONSE"
  exit 1
fi

echo ""
echo "==> Ralph hook integration test passed."
