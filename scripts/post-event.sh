#!/usr/bin/env bash
# Post an event to the ws-mcp bridge.
# Usage: post-event.sh <event_type> [source] [payload_json] [--repo <repo>]
#
# Examples:
#   post-event.sh worker.status multiclaude '{"worker":"witty-deer","status":"done"}'
#   post-event.sh system.healthcheck
#   post-event.sh task.started ralph '{}' --repo enriched-alert
#
# Environment:
#   WS_MCP_URL — Bridge base URL (default: http://localhost:8080)
set -euo pipefail

# Parse positional and flag arguments
REPO=""
POSITIONAL=()
while [[ $# -gt 0 ]]; do
  case "$1" in
    --repo)
      REPO="${2:?--repo requires a value}"
      shift 2
      ;;
    *)
      POSITIONAL+=("$1")
      shift
      ;;
  esac
done

EVENT_TYPE="${POSITIONAL[0]:?Usage: post-event.sh <event_type> [source] [payload_json] [--repo <repo>]}"
SOURCE="${POSITIONAL[1]:-system}"
PAYLOAD="${POSITIONAL[2]:-{}}"
BASE_URL="${WS_MCP_URL:-http://localhost:8080}"

ID="evt-$(date +%s)-${RANDOM}"
TS="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# Build the repo field if provided
REPO_FIELD=""
if [ -n "$REPO" ]; then
  REPO_FIELD="\"repo\":\"${REPO}\","
fi

curl -sf -X POST "${BASE_URL}/event" \
  -H "Content-Type: application/json" \
  -d "{\"id\":\"${ID}\",\"source\":\"${SOURCE}\",\"type\":\"${EVENT_TYPE}\",${REPO_FIELD}\"ts\":\"${TS}\",\"payload\":${PAYLOAD}}" \
  >/dev/null 2>&1 || {
  echo "post-event: failed to POST to ${BASE_URL}/event" >&2
  exit 1
}
