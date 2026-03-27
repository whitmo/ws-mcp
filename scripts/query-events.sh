#!/usr/bin/env bash
# Query events from the ws-mcp bridge via JSON-RPC.
# Usage: query-events.sh [method] [params_json]
#
# Methods:
#   events.latest   '{"limit":10}'         (default)
#   events.filter   '{"source":"ralph"}'
#   events.ack      '{"id":"...","acked_by":"me"}'
#   report.summary  '{"window":60}'
#
# Examples:
#   query-events.sh                                    # latest 10
#   query-events.sh events.latest '{"limit":5}'
#   query-events.sh events.filter '{"source":"multiclaude"}'
#
# Environment:
#   WS_MCP_URL — Bridge base URL (default: http://localhost:8080)
set -euo pipefail

METHOD="${1:-events.latest}"
PARAMS="${2:-{}}"
BASE_URL="${WS_MCP_URL:-http://localhost:8080}"
ID="${RANDOM}"

RESP=$(curl -sf -X POST "${BASE_URL}/rpc" \
  -H "Content-Type: application/json" \
  -d "{\"jsonrpc\":\"2.0\",\"method\":\"${METHOD}\",\"params\":${PARAMS},\"id\":${ID}}")

if command -v python3 &>/dev/null; then
  echo "$RESP" | python3 -m json.tool
else
  echo "$RESP"
fi
