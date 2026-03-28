#!/usr/bin/env bash
# multiclaude-hook.sh — POST multiclaude worker lifecycle events to the ws-mcp bridge.
# Usage: multiclaude-hook.sh <event_type> <worker_name> [payload_json]
#
# Arguments:
#   event_type   — Required. One of: worker.started, worker.committed, worker.pushed, worker.completed
#   worker_name  — Required. Name of the multiclaude worker (e.g. "nice-lion")
#   payload_json — Optional. JSON string for extra payload fields. Defaults to "{}"
#
# Environment:
#   WS_MCP_URL — Bridge base URL (default: http://localhost:8080)
#
# This script wraps scripts/post-event.sh, merging the worker name into the payload.
# Auto-detects the current git repo name and includes it in events.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

EVENT_TYPE="${1:?Usage: multiclaude-hook.sh <event_type> <worker_name> [payload_json]}"
WORKER_NAME="${2:?Usage: multiclaude-hook.sh <event_type> <worker_name> [payload_json]}"
EXTRA_PAYLOAD="${3:-{}}"

# Validate event type
case "$EVENT_TYPE" in
  worker.started|worker.committed|worker.pushed|worker.completed)
    ;;
  *)
    echo "multiclaude-hook: unknown event type '${EVENT_TYPE}'" >&2
    echo "  expected: worker.started, worker.committed, worker.pushed, worker.completed" >&2
    exit 1
    ;;
esac

# Auto-detect repo name from git remote or directory name
REPO=""
if command -v git >/dev/null 2>&1 && git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  REPO="$(git remote get-url origin 2>/dev/null | sed 's|.*/||;s|\.git$||' || basename "$(git rev-parse --show-toplevel 2>/dev/null)" 2>/dev/null || true)"
fi

# Merge worker name into payload. If extra payload is "{}", just use worker field.
# For non-empty extra payload, merge them together.
if [ "$EXTRA_PAYLOAD" = "{}" ]; then
  PAYLOAD="{\"worker\":\"${WORKER_NAME}\"}"
else
  # Strip leading { from extra, prepend worker field
  MERGED="$(echo "$EXTRA_PAYLOAD" | sed 's/^{//')"
  PAYLOAD="{\"worker\":\"${WORKER_NAME}\",${MERGED}"
fi

# Build repo flag if detected
REPO_FLAG=()
if [ -n "$REPO" ]; then
  REPO_FLAG=(--repo "$REPO")
fi

exec "${SCRIPT_DIR}/post-event.sh" "$EVENT_TYPE" "multiclaude" "$PAYLOAD" "${REPO_FLAG[@]}"
