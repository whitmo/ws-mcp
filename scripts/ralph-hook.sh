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
# Auto-detects the current git repo name and includes it in events.
set -euo pipefail

EVENT_TYPE="${1:?Usage: ralph-hook.sh <event_type> [source] [payload_json]}"
SOURCE="${2:-ralph}"
PAYLOAD="${3:-{}}"
BASE_URL="${WS_MCP_URL:-http://localhost:8080}"

# Auto-detect repo name from git remote or directory name
REPO=""
if command -v git >/dev/null 2>&1 && git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  REPO="$(git remote get-url origin 2>/dev/null | sed 's|.*/||;s|\.git$||' || basename "$(git rev-parse --show-toplevel 2>/dev/null)" 2>/dev/null || true)"
fi

# Generate a pseudo-UUID (v4-ish) using /dev/urandom
ID="$(od -An -tx1 -N16 /dev/urandom | tr -d ' \n' | sed 's/\(.\{8\}\)\(.\{4\}\)\(.\{4\}\)\(.\{4\}\)\(.\{12\}\)/\1-\2-\3-\4-\5/')"

# ISO 8601 timestamp
TS="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# Build the repo field if detected
REPO_FIELD=""
if [ -n "$REPO" ]; then
  REPO_FIELD="\"repo\": \"${REPO}\","
fi

# Build and POST the event
curl -sf -X POST "${BASE_URL}/event" \
  -H "Content-Type: application/json" \
  -d "$(cat <<EOF
{
  "id": "${ID}",
  "source": "${SOURCE}",
  "type": "${EVENT_TYPE}",
  ${REPO_FIELD}
  "ts": "${TS}",
  "payload": ${PAYLOAD}
}
EOF
)" >/dev/null 2>&1 || {
  echo "ralph-hook: failed to POST event to ${BASE_URL}/event" >&2
  exit 1
}
