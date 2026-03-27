#!/usr/bin/env bash
# Post an event to the ws-mcp bridge.
# Usage: post-event.sh <event_type> [source] [payload_json]
#
# Examples:
#   post-event.sh worker.status multiclaude '{"worker":"witty-deer","status":"done"}'
#   post-event.sh system.healthcheck
#
# Environment:
#   WS_MCP_URL — Bridge base URL (default: http://localhost:8080)
set -euo pipefail

EVENT_TYPE="${1:?Usage: post-event.sh <event_type> [source] [payload_json]}"
SOURCE="${2:-system}"
PAYLOAD="${3:-{}}"
BASE_URL="${WS_MCP_URL:-http://localhost:8080}"

ID="evt-$(date +%s)-${RANDOM}"
TS="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

curl -sf -X POST "${BASE_URL}/event" \
  -H "Content-Type: application/json" \
  -d "{\"id\":\"${ID}\",\"source\":\"${SOURCE}\",\"type\":\"${EVENT_TYPE}\",\"ts\":\"${TS}\",\"payload\":${PAYLOAD}}" \
  >/dev/null 2>&1 || {
  echo "post-event: failed to POST to ${BASE_URL}/event" >&2
  exit 1
}
