#!/usr/bin/env bash
# Demo: multi-agent communication via the ws-mcp bridge.
#
# Starts the bridge, connects a WS observer, posts sample agent lifecycle
# events, then queries them back via JSON-RPC.
#
# Usage:
#   scripts/demo.sh           # full run (starts bridge + observer)
#   scripts/demo.sh --dry     # dry-run: prints commands without executing
#
# Requires: curl. Optional: websocat or wscat for live WS observer.
set -euo pipefail

DRY=false
[ "${1:-}" = "--dry" ] && DRY=true

PORT="${WS_MCP_PORT:-8080}"
BASE="http://localhost:${PORT}"
BRIDGE_PID=""
OBSERVER_PID=""
TMPDIR_DEMO=$(mktemp -d)

cleanup() {
  echo ""
  echo "==> Cleaning up..."
  [ -n "$OBSERVER_PID" ] && kill "$OBSERVER_PID" 2>/dev/null && wait "$OBSERVER_PID" 2>/dev/null || true
  [ -n "$BRIDGE_PID" ]   && kill "$BRIDGE_PID"   2>/dev/null && wait "$BRIDGE_PID"   2>/dev/null || true
  rm -rf "$TMPDIR_DEMO"
  echo "    Done."
}
trap cleanup EXIT INT TERM

run() {
  echo "  \$ $*"
  if ! $DRY; then
    "$@"
  fi
}

post_event() {
  local etype="$1" source="$2" payload="$3"
  local id="evt-demo-$(date +%s)-${RANDOM}"
  local ts
  ts="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  local body="{\"id\":\"${id}\",\"source\":\"${source}\",\"type\":\"${etype}\",\"ts\":\"${ts}\",\"payload\":${payload}}"
  echo ""
  echo "  POST /event  type=${etype}  source=${source}"
  if ! $DRY; then
    curl -sf -X POST "${BASE}/event" \
      -H "Content-Type: application/json" \
      -d "$body" >/dev/null
  fi
}

rpc_call() {
  local method="$1"
  local params="${2:-"{}"}"
  echo ""
  echo "  RPC ${method}  params=${params}"
  if ! $DRY; then
    local resp
    resp=$(curl -sf -X POST "${BASE}/rpc" \
      -H "Content-Type: application/json" \
      -d "{\"jsonrpc\":\"2.0\",\"method\":\"${method}\",\"params\":${params},\"id\":1}")
    if command -v python3 &>/dev/null; then
      echo "$resp" | python3 -m json.tool
    else
      echo "$resp"
    fi
  fi
}

cd "$(dirname "$0")/.."

# ── Header ──────────────────────────────────────────────────────
echo "╔══════════════════════════════════════════════════════╗"
echo "║   ws-mcp bridge — multi-agent communication demo    ║"
echo "╚══════════════════════════════════════════════════════╝"
$DRY && echo "(dry-run mode — commands are printed, not executed)"
echo ""

# ── 1. Start bridge ────────────────────────────────────────────
echo "==> Step 1: Start the bridge"
run go build -o "${TMPDIR_DEMO}/bridge" ./src/cmd/bridge
if ! $DRY; then
  "${TMPDIR_DEMO}/bridge" --data-dir "${TMPDIR_DEMO}" &
  BRIDGE_PID=$!
  sleep 1
  # Quick healthcheck
  if ! curl -sf "${BASE}/healthz" >/dev/null; then
    echo "FAIL: bridge did not start" >&2
    exit 1
  fi
  echo "    Bridge running (pid ${BRIDGE_PID})"
else
  echo "  \$ go run src/cmd/bridge/main.go &"
fi
echo ""

# ── 2. Start WS observer ──────────────────────────────────────
echo "==> Step 2: Start a WebSocket observer"
if ! $DRY; then
  go run ./src/cmd/observer --pretty --url "ws://localhost:${PORT}/ws" \
    > "${TMPDIR_DEMO}/observer.log" 2>&1 &
  OBSERVER_PID=$!
  sleep 0.5
  echo "    Observer connected (pid ${OBSERVER_PID})"
else
  echo "  \$ go run src/cmd/observer/main.go --pretty --url ws://localhost:${PORT}/ws &"
fi
echo ""

# ── 3. Post sample agent lifecycle events ──────────────────────
echo "==> Step 3: Simulate agent lifecycle events"

post_event "agent.started" "multiclaude" \
  '{"agent":"clever-panda","task":"implement auth module"}'
$DRY || sleep 0.3

post_event "task.completed" "multiclaude" \
  '{"agent":"clever-panda","task":"implement auth module","result":"success"}'
$DRY || sleep 0.3

post_event "review.requested" "multiclaude" \
  '{"pr":42,"agent":"clever-panda","reviewer":"supervisor"}'
$DRY || sleep 0.3

echo ""

# ── 4. Query events via JSON-RPC ──────────────────────────────
echo "==> Step 4: Query events via JSON-RPC"

rpc_call "events.latest" '{"limit":5}'

echo ""

# ── 5. Show observer output ───────────────────────────────────
echo "==> Step 5: Observer received these events in real-time:"
if ! $DRY; then
  sleep 0.5
  if [ -s "${TMPDIR_DEMO}/observer.log" ]; then
    cat "${TMPDIR_DEMO}/observer.log"
  else
    echo "    (no output captured — observer may not have connected in time)"
  fi
else
  echo "  (observer log would be shown here)"
fi

echo ""
echo "==> Demo complete. Press Ctrl+C or wait for cleanup."
