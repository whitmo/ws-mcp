#!/usr/bin/env bash
# Observe the ws-mcp event stream and report issues.
# Usage: observe-and-report.sh [--since MINUTES] [--watch]
#
# Captures recent events, analyzes for problems, and writes a report.
# Designed to be called periodically by a supervisor agent.
#
# Environment:
#   WS_MCP_URL — Bridge base URL (default: http://localhost:8080)
set -euo pipefail

BASE_URL="${WS_MCP_URL:-http://localhost:8080}"
SINCE="${1:-5}"
REPORT="/tmp/ws-mcp-observation.json"

# Get summary
SUMMARY=$(curl -sf -X POST "${BASE_URL}/rpc" \
  -H "Content-Type: application/json" \
  -d "{\"jsonrpc\":\"2.0\",\"method\":\"report.summary\",\"params\":{\"window\":${SINCE}},\"id\":1}")

# Get latest events
EVENTS=$(curl -sf -X POST "${BASE_URL}/rpc" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"events.latest","params":{"limit":50},"id":2}')

# Get unacked events (potential issues)
ALL_EVENTS=$(echo "$EVENTS" | python3 -c "
import json, sys
data = json.load(sys.stdin)
events = data.get('result', [])
unacked = [e for e in events if not e.get('acked', False)]
errors = [e for e in events if 'error' in e.get('type','') or 'fail' in e.get('type','')]
sources = {}
types = {}
for e in events:
    sources[e.get('source','')] = sources.get(e.get('source',''), 0) + 1
    types[e.get('type','')] = types.get(e.get('type',''), 0) + 1

report = {
    'total_events': len(events),
    'unacked_count': len(unacked),
    'error_events': len(errors),
    'sources': sources,
    'types': types,
    'errors': [{'id': e['id'], 'type': e['type'], 'source': e['source'], 'payload': e.get('payload',{})} for e in errors],
    'recent_unacked': [{'id': e['id'], 'type': e['type'], 'source': e['source'], 'ts': e['ts']} for e in unacked[:10]]
}
json.dump(report, sys.stdout, indent=2)
" 2>/dev/null || echo '{"error": "parse failed"}')

# Write combined report
python3 -c "
import json
summary = json.loads('''${SUMMARY}''')
observation = json.loads('''${ALL_EVENTS}''')
report = {
    'summary': summary.get('result', {}),
    'observation': observation,
    'bridge_url': '${BASE_URL}',
    'window_minutes': ${SINCE}
}
with open('${REPORT}', 'w') as f:
    json.dump(report, f, indent=2)
print(json.dumps(report, indent=2))
"
