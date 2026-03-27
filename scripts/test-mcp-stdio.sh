#!/usr/bin/env bash
# test-mcp-stdio.sh — End-to-end test of the MCP stdio transport.
# Pipes JSON-RPC requests into ./bridge --stdio and validates responses.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

PASS=0
FAIL=0

pass() { echo "  ✓ $1"; PASS=$((PASS+1)); }
fail() { echo "  ✗ $1: $2"; FAIL=$((FAIL+1)); }

echo "Building bridge..."
go build -o bridge ./src/cmd/bridge

echo "Running MCP stdio tests..."

# Send a sequence of JSON-RPC requests and capture all responses
RESPONSES=$(printf '%s\n' \
  '{"jsonrpc":"2.0","method":"initialize","id":1}' \
  '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
  '{"jsonrpc":"2.0","method":"tools/list","id":2}' \
  '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"events_latest","arguments":{"limit":5}},"id":3}' \
  '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"events_ack","arguments":{"id":"no-such-id"}},"id":4}' \
  | ./bridge --stdio 2>/dev/null)

# We expect exactly 4 response lines (notifications/initialized produces none)
LINE_COUNT=$(echo "$RESPONSES" | wc -l | tr -d ' ')

echo ""
echo "--- initialize ---"
RESP1=$(echo "$RESPONSES" | sed -n '1p')
if echo "$RESP1" | jq -e '.result.protocolVersion == "2024-11-05"' >/dev/null 2>&1; then
  pass "protocolVersion is 2024-11-05"
else
  fail "protocolVersion" "$RESP1"
fi
if echo "$RESP1" | jq -e '.result.serverInfo.name == "ws-mcp"' >/dev/null 2>&1; then
  pass "serverInfo.name is ws-mcp"
else
  fail "serverInfo.name" "$RESP1"
fi
if echo "$RESP1" | jq -e '.result.capabilities.tools != null' >/dev/null 2>&1; then
  pass "capabilities.tools present"
else
  fail "capabilities.tools" "$RESP1"
fi

echo ""
echo "--- tools/list ---"
RESP2=$(echo "$RESPONSES" | sed -n '2p')
TOOL_COUNT=$(echo "$RESP2" | jq '.result.tools | length' 2>/dev/null)
if [ "$TOOL_COUNT" = "4" ]; then
  pass "4 tools returned"
else
  fail "tool count" "expected 4, got $TOOL_COUNT"
fi

# Check each expected tool name
for TOOL_NAME in events_latest events_filter events_ack report_summary; do
  if echo "$RESP2" | jq -e ".result.tools[] | select(.name == \"$TOOL_NAME\")" >/dev/null 2>&1; then
    pass "tool '$TOOL_NAME' present"
  else
    fail "tool '$TOOL_NAME'" "not found"
  fi
done

echo ""
echo "--- tools/call events_latest ---"
RESP3=$(echo "$RESPONSES" | sed -n '3p')
if echo "$RESP3" | jq -e '.result.content[0].type == "text"' >/dev/null 2>&1; then
  pass "content[0].type is text"
else
  fail "content format" "$RESP3"
fi
if echo "$RESP3" | jq -e '.error == null' >/dev/null 2>&1; then
  pass "no error"
else
  fail "unexpected error" "$RESP3"
fi

echo ""
echo "--- tools/call events_ack (error case) ---"
RESP4=$(echo "$RESPONSES" | sed -n '4p')
if echo "$RESP4" | jq -e '.error != null' >/dev/null 2>&1; then
  pass "error returned for nonexistent event"
else
  fail "expected error" "$RESP4"
fi

echo ""
echo "--- .mcp.json validation ---"
if [ -f "$REPO_ROOT/.mcp.json" ]; then
  CMD=$(jq -r '.mcpServers["ws-mcp"].command' "$REPO_ROOT/.mcp.json")
  ARGS=$(jq -r '.mcpServers["ws-mcp"].args | join(" ")' "$REPO_ROOT/.mcp.json")
  if [ "$CMD" = "go" ] && [ "$ARGS" = "run ./src/cmd/bridge --stdio" ]; then
    pass ".mcp.json command and args correct"
  else
    fail ".mcp.json" "command=$CMD args=$ARGS"
  fi
else
  fail ".mcp.json" "file not found"
fi

# Cleanup
rm -f bridge

echo ""
echo "=============================="
echo "Results: $PASS passed, $FAIL failed"
echo "=============================="
[ "$FAIL" -eq 0 ] || exit 1
