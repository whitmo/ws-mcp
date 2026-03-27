#!/usr/bin/env bash
# ralph-hook.sh — POST ralph lifecycle events to the ws-mcp bridge.
# Usage: ralph-hook.sh <event_type> [source] [payload_json]
#
# Arguments:
#   event_type   — Required. e.g. "loop.start", "iteration.complete"
#   source       — Optional. Defaults to "ralph"
#   payload_json — Optional. JSON string for payload. Defaults to "{}"
#
# Environment:
#   WS_MCP_URL — Bridge base URL (default: http://localhost:8080)
#
# Called by ralph hooks or standalone. Generates a UUID-style id and ISO 8601 ts.
set -euo pipefail

EVENT_TYPE="${1:?Usage: ralph-hook.sh <event_type> [source] [payload_json]}"
SOURCE="${2:-ralph}"
PAYLOAD="${3:-{}}"
BASE_URL="${WS_MCP_URL:-http://localhost:8080}"

# Generate a pseudo-UUID (v4-ish) using /dev/urandom
ID="$(od -An -tx1 -N16 /dev/urandom | tr -d ' \n' | sed 's/\(.\{8\}\)\(.\{4\}\)\(.\{4\}\)\(.\{4\}\)\(.\{12\}\)/\1-\2-\3-\4-\5/')"

# ISO 8601 timestamp
TS="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# Build and POST the event
curl -sf -X POST "${BASE_URL}/event" \
  -H "Content-Type: application/json" \
  -d "$(cat <<EOF
{
  "id": "${ID}",
  "source": "${SOURCE}",
  "type": "${EVENT_TYPE}",
  "ts": "${TS}",
  "payload": ${PAYLOAD}
}
EOF
)" >/dev/null 2>&1 || {
  echo "ralph-hook: failed to POST event to ${BASE_URL}/event" >&2
  exit 1
}
