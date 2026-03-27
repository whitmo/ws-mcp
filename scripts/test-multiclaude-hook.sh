#!/usr/bin/env bash
# test-multiclaude-hook.sh — Integration test for multiclaude-hook.sh → ws-mcp bridge round-trip.
# Starts the bridge, fires test events via multiclaude-hook.sh, queries /rpc events.latest.
set -euo pipefail

PORT="${WS_MCP_PORT:-8080}"
BASE="http://localhost:${PORT}"
BRIDGE_PID=""

cleanup() {
  if [ -n "$BRIDGE_PID" ]; then
    kill "$BRIDGE_PID" 2>/dev/null
    wait "$BRIDGE_PID" 2>/dev/null || true
  fi
  rm -f /tmp/ws_mcp_multiclaude_test
}
trap cleanup EXIT

cd "$(dirname "$0")/.."

echo "==> Building bridge..."
go build -o /tmp/ws_mcp_multiclaude_test ./src/cmd/bridge

echo "==> Starting bridge on port ${PORT}..."
/tmp/ws_mcp_multiclaude_test &
BRIDGE_PID=$!
sleep 1

# Healthcheck
STATUS=$(curl -s -o /dev/null -w '%{http_code}' "${BASE}/healthz")
if [ "$STATUS" != "200" ]; then
  echo "FAIL: healthcheck returned ${STATUS}"
  exit 1
fi
echo "    Bridge healthy"

# Test 1: worker.started event
echo "==> Posting worker.started event..."
./scripts/multiclaude-hook.sh worker.started test-lion
echo "    Posted"

# Test 2: worker.completed with extra payload
echo "==> Posting worker.completed event with payload..."
./scripts/multiclaude-hook.sh worker.completed test-lion '{"pr_url":"https://github.com/test/repo/pull/42"}'
echo "    Posted"

# Query events.latest via JSON-RPC /rpc
echo "==> Querying /rpc events.latest..."
RESPONSE=$(curl -sf -X POST "${BASE}/rpc" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"events.latest","params":{"limit":10},"id":1}')

# Check: worker.started event present
if echo "$RESPONSE" | grep -q "worker.started"; then
  echo "    OK — worker.started event found"
else
  echo "FAIL: worker.started not found in response:"
  echo "$RESPONSE"
  exit 1
fi

# Check: worker.completed event present
if echo "$RESPONSE" | grep -q "worker.completed"; then
  echo "    OK — worker.completed event found"
else
  echo "FAIL: worker.completed not found in response:"
  echo "$RESPONSE"
  exit 1
fi

# Check: source is multiclaude
if echo "$RESPONSE" | grep -q '"source":"multiclaude"'; then
  echo "    OK — source is 'multiclaude'"
else
  echo "FAIL: source mismatch in response:"
  echo "$RESPONSE"
  exit 1
fi

# Check: worker name in payload
if echo "$RESPONSE" | grep -q '"worker":"test-lion"'; then
  echo "    OK — worker name 'test-lion' in payload"
else
  echo "FAIL: worker name not found in response:"
  echo "$RESPONSE"
  exit 1
fi

# Check: extra payload merged
if echo "$RESPONSE" | grep -q 'pr_url'; then
  echo "    OK — extra payload (pr_url) present"
else
  echo "FAIL: extra payload not found in response:"
  echo "$RESPONSE"
  exit 1
fi

# Test 3: invalid event type should fail
echo "==> Testing invalid event type rejection..."
if ./scripts/multiclaude-hook.sh worker.invalid test-lion 2>/dev/null; then
  echo "FAIL: invalid event type was accepted"
  exit 1
else
  echo "    OK — invalid event type rejected"
fi

echo ""
echo "==> Multiclaude hook integration test passed."
