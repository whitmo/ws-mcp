#!/usr/bin/env bash
# Build and start the ws-mcp bridge in the background.
# Usage: bridge-start.sh [port]
#
# Kills any existing bridge on the port first.
# Prints the PID on success.
#
# Environment:
#   WS_MCP_PORT — Port to listen on (default: 8080, or first arg)
set -euo pipefail

PORT="${1:-${WS_MCP_PORT:-8080}}"
BRIDGE_BIN="/tmp/ws_mcp_bridge"
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

# Kill existing bridge on this port
lsof -ti:"${PORT}" 2>/dev/null | xargs kill 2>/dev/null || true
sleep 0.5

cd "$REPO_ROOT"
go build -o "$BRIDGE_BIN" ./src/cmd/bridge

"$BRIDGE_BIN" &
PID=$!
sleep 1

if curl -sf -o /dev/null "http://localhost:${PORT}/healthz"; then
  echo "Bridge running on :${PORT} (PID ${PID})"
else
  echo "Bridge failed to start" >&2
  kill "$PID" 2>/dev/null
  exit 1
fi
