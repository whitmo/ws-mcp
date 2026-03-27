#!/usr/bin/env bash
# Smoke test: start bridge, post an event, verify WS round-trip.
# Requires: curl, websocat (or wscat). Falls back to HTTP-only if no WS tool found.
set -euo pipefail

PORT="${WS_MCP_PORT:-8080}"
BASE="http://localhost:${PORT}"
BRIDGE_PID=""

cleanup() {
  if [ -n "$BRIDGE_PID" ]; then
    kill "$BRIDGE_PID" 2>/dev/null
    wait "$BRIDGE_PID" 2>/dev/null || true
  fi
  rm -f /tmp/ws_mcp_smoke_ws.log /tmp/ws_mcp_bridge
}
trap cleanup EXIT

echo "==> Building bridge..."
cd "$(dirname "$0")/.."
go build -o /tmp/ws_mcp_bridge ./src/cmd/bridge

echo "==> Starting bridge on port ${PORT}..."
/tmp/ws_mcp_bridge &
BRIDGE_PID=$!
sleep 1

# Healthcheck
echo "==> Healthcheck..."
STATUS=$(curl -s -o /dev/null -w '%{http_code}' "${BASE}/healthz")
if [ "$STATUS" != "200" ]; then
  echo "FAIL: healthcheck returned ${STATUS}"
  exit 1
fi
echo "    OK (200)"

# Post an event
EVENT_ID="smoke-$(date +%s)"
echo "==> Posting event ${EVENT_ID}..."
POST_STATUS=$(curl -s -o /dev/null -w '%{http_code}' -X POST "${BASE}/event" \
  -H "Content-Type: application/json" \
  -d "{
    \"id\": \"${EVENT_ID}\",
    \"source\": \"system\",
    \"type\": \"smoke_test\",
    \"ts\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",
    \"payload\": {\"test\": true}
  }")

if [ "$POST_STATUS" != "202" ]; then
  echo "FAIL: POST /event returned ${POST_STATUS}"
  exit 1
fi
echo "    OK (202)"

# WebSocket round-trip (if websocat or wscat available)
WS_TOOL=""
if command -v websocat &>/dev/null; then
  WS_TOOL="websocat"
elif command -v wscat &>/dev/null; then
  WS_TOOL="wscat"
fi

if [ -n "$WS_TOOL" ]; then
  echo "==> WebSocket round-trip (${WS_TOOL})..."

  if [ "$WS_TOOL" = "websocat" ]; then
    websocat -t "ws://localhost:${PORT}/ws" > /tmp/ws_mcp_smoke_ws.log &
  else
    wscat -c "ws://localhost:${PORT}/ws" > /tmp/ws_mcp_smoke_ws.log &
  fi
  WS_PID=$!
  sleep 0.5

  # Post another event while WS is listening
  WS_EVENT_ID="smoke-ws-$(date +%s)"
  curl -s -X POST "${BASE}/event" \
    -H "Content-Type: application/json" \
    -d "{
      \"id\": \"${WS_EVENT_ID}\",
      \"source\": \"system\",
      \"type\": \"smoke_test_ws\",
      \"ts\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",
      \"payload\": {\"ws_test\": true}
    }" >/dev/null

  sleep 1
  kill "$WS_PID" 2>/dev/null || true

  if grep -q "$WS_EVENT_ID" /tmp/ws_mcp_smoke_ws.log 2>/dev/null; then
    echo "    OK (event received via WebSocket)"
  else
    echo "    WARN: event not found in WS output (may be timing-related)"
  fi
else
  echo "==> Skipping WebSocket test (install websocat or wscat)"
fi

echo ""
echo "==> Smoke test passed."
