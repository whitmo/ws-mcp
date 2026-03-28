#!/usr/bin/env bash
# demo.sh — Launch the ws-mcp bridge and a swarm of agents communicating through it.
#
# What this does:
#   1. Starts the bridge hub
#   2. Starts a WebSocket observer (live event stream in terminal)
#   3. Spawns 3 multiclaude workers that post events as they work
#   4. Spawns a reactor that dispatches events to actions
#   5. Runs ralph with a review hat to monitor worker output
#
# Usage:
#   bash scripts/demo.sh          # run the full demo
#   bash scripts/demo.sh --dry    # show what would run without executing
#
# Prerequisites:
#   - go 1.24+, ws-mcp-bridge in ~/bin, multiclaude, ralph
#   - Bridge not already running on :8080
#
# Press Ctrl+C to tear everything down.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DRY_RUN="${1:-}"
PIDS=()

cleanup() {
  echo ""
  echo "==> Tearing down..."
  for pid in "${PIDS[@]}"; do
    kill "$pid" 2>/dev/null || true
  done
  # Give workers a moment to stop
  sleep 1
  # Kill bridge last
  lsof -ti:8080 2>/dev/null | xargs kill 2>/dev/null || true
  echo "==> Done."
}
trap cleanup EXIT

step() {
  echo ""
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo "  $1"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

run() {
  if [ "$DRY_RUN" = "--dry" ]; then
    echo "  [dry] $*"
  else
    "$@"
  fi
}

cd "$REPO_ROOT"

# ─── Step 1: Start the bridge hub ─────────────────────────────────────────────
step "Starting ws-mcp bridge hub on :8080"

if curl -sf -o /dev/null http://localhost:8080/healthz 2>/dev/null; then
  echo "  Bridge already running."
else
  run go build -o /tmp/ws_mcp_demo_bridge ./src/cmd/bridge
  if [ "$DRY_RUN" != "--dry" ]; then
    /tmp/ws_mcp_demo_bridge &
    PIDS+=($!)
    sleep 1
    echo "  Bridge started (PID ${PIDS[-1]})"
  fi
fi

# Post a demo-start event
run bash scripts/post-event.sh demo.started system '{"description":"agent swarm demo","agents":["worker-a","worker-b","worker-c","reactor","observer","ralph"]}'

# ─── Step 2: Start the live observer ──────────────────────────────────────────
step "Starting live event observer (filtered, no noise)"

if [ "$DRY_RUN" != "--dry" ]; then
  go run ./src/cmd/observer --pretty --url ws://localhost:8080/ws 2>/dev/null &
  PIDS+=($!)
  echo "  Observer running (PID ${PIDS[-1]})"
fi

sleep 1

# ─── Step 3: Spawn multiclaude workers ────────────────────────────────────────
step "Spawning 3 multiclaude workers"

WORKER_A_TASK="You are Worker A in a ws-mcp demo. 1) Post task.started: bash scripts/post-event.sh task.started multiclaude '{\"agent\":\"worker-a\",\"description\":\"add godoc comments to src/internal/types/event.go\"}'. 2) Add godoc comments to the exported types in src/internal/types/event.go. 3) Post progress events as you work. 4) When done, post task.completed and review.requested for worker-b. 5) go test ./... must pass. 6) Commit and push."

WORKER_B_TASK="You are Worker B in a ws-mcp demo. 1) Post task.started: bash scripts/post-event.sh task.started multiclaude '{\"agent\":\"worker-b\",\"description\":\"add godoc comments to src/internal/store/ring_buffer.go\"}'. 2) Add godoc comments to exported types/functions in src/internal/store/ring_buffer.go. 3) Post progress events. 4) Periodically check events (bash scripts/query-events.sh events.latest) — if you see review.requested from worker-a, acknowledge it and review their changes. 5) Post review.completed with your verdict. 6) go test ./... must pass. Commit and push."

WORKER_C_TASK="You are Worker C in a ws-mcp demo. 1) Post task.started: bash scripts/post-event.sh task.started multiclaude '{\"agent\":\"worker-c\",\"description\":\"add a benchmark test for ring buffer\"}'. 2) Create src/internal/store/ring_buffer_bench_test.go with benchmarks for Push and Latest operations. 3) Post progress events. 4) Run go test -bench=. ./src/internal/store/ and post the results as a system event. 5) Post task.completed when done. 6) Commit and push."

run multiclaude work "$WORKER_A_TASK" --repo ws-mcp
sleep 2
run multiclaude work "$WORKER_B_TASK" --repo ws-mcp
sleep 2
run multiclaude work "$WORKER_C_TASK" --repo ws-mcp

# ─── Step 4: Start the reactor ────────────────────────────────────────────────
step "Starting event reactor (dispatches events to actions)"

if [ -f configs/reactor.yaml ]; then
  if [ "$DRY_RUN" != "--dry" ]; then
    go run ./src/cmd/reactor --config configs/reactor.yaml --url ws://localhost:8080/ws 2>/dev/null &
    PIDS+=($!)
    echo "  Reactor running (PID ${PIDS[-1]})"
  fi
else
  echo "  No reactor.yaml found, skipping."
fi

# ─── Step 5: Launch ralph as reviewer ─────────────────────────────────────────
step "Launching ralph with review hat (autonomous, monitoring workers)"

if [ "$DRY_RUN" != "--dry" ]; then
  ralph run \
    -p "Monitor the ws-mcp demo swarm. Three multiclaude workers are running: worker-a (godoc for types), worker-b (godoc for store + reviewing worker-a), worker-c (benchmarks). Check their progress by querying the bridge: bash scripts/query-events.sh events.filter '{\"source\":\"multiclaude\"}'. Also check their worktrees and git logs. Post your observations as events: bash scripts/post-event.sh review.observation ralph '{\"finding\":\"...\"}'. When all workers have pushed and tests pass, post demo.completed and emit REVIEW_COMPLETE." \
    -H builtin:review --autonomous 2>/dev/null &
  PIDS+=($!)
  echo "  Ralph running (PID ${PIDS[-1]})"
fi

# ─── Status ───────────────────────────────────────────────────────────────────
step "Demo is running!"

echo ""
echo "  Bridge:   http://localhost:8080 (hub)"
echo "  Observer: live event stream in this terminal"
echo "  Workers:  multiclaude work list --repo ws-mcp"
echo "  Events:   bash scripts/query-events.sh events.latest"
echo "  Summary:  bash scripts/query-events.sh report.summary"
echo ""
echo "  Agents are communicating through the bridge."
echo "  Watch the observer output above for live events."
echo ""
echo "  Press Ctrl+C to stop everything."
echo ""

# Wait for Ctrl+C
if [ "$DRY_RUN" != "--dry" ]; then
  # Poll and display summary every 30 seconds
  while true; do
    sleep 30
    echo ""
    echo "--- Status update ($(date +%H:%M:%S)) ---"
    bash scripts/query-events.sh report.summary '{"window":5}' 2>/dev/null | python3 -c "
import json, sys
try:
    d = json.load(sys.stdin)
    r = d.get('result', {})
    print(f\"  Events (last 5min): {r.get('total_events',0)}\")
    print(f\"  By source: {r.get('by_source',{})}\")
    print(f\"  Unacked: {r.get('unacked_count',0)}\")
except: pass
" 2>/dev/null || true
    multiclaude work list --repo ws-mcp 2>/dev/null | grep -E '●|○' || echo "  No active workers"
  done
fi
