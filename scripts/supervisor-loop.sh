#!/usr/bin/env bash
# supervisor-loop.sh — Monitor workers via the bridge and report status.
# Usage: supervisor-loop.sh [interval_seconds]
#
# Polls the bridge for multiclaude events, checks worker git progress,
# and prints a summary. Designed to be run in background by the supervisor agent.
set -euo pipefail

INTERVAL="${1:-30}"
BASE_URL="${WS_MCP_URL:-http://localhost:8080}"
SEEN_IDS="/tmp/ws-mcp-supervisor-seen.txt"
touch "$SEEN_IDS"

while true; do
  echo "── $(date +%H:%M:%S) ──────────────────────────────"

  # Get recent multiclaude events we haven't seen
  EVENTS=$(curl -sf -X POST "${BASE_URL}/rpc" \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"events.filter","params":{"source":"multiclaude"},"id":1}' 2>/dev/null || echo '{"result":[]}')

  echo "$EVENTS" | python3 -c "
import json, sys
seen_file = '${SEEN_IDS}'
with open(seen_file) as f: seen = set(f.read().split())
d = json.load(sys.stdin)
events = d.get('result', []) or []
new_events = [e for e in events if e['id'] not in seen]
if new_events:
    print(f'  NEW: {len(new_events)} event(s)')
    for e in new_events:
        agent = e.get('payload',{}).get('agent','?')
        print(f'    [{agent}] {e[\"type\"]}: {json.dumps(e.get(\"payload\",{}))[:60]}')
    with open(seen_file, 'a') as f:
        for e in new_events: f.write(e['id'] + '\n')
else:
    print('  No new multiclaude events')
" 2>/dev/null || echo "  (bridge unreachable)"

  # Check workers
  WORKERS=$(multiclaude work list --repo ws-mcp 2>&1 | grep -E '●|○' || true)
  if [ -n "$WORKERS" ]; then
    echo "  Workers:"
    echo "$WORKERS" | while read -r line; do
      NAME=$(echo "$line" | awk '{print $1}')
      WT="$HOME/.multiclaude/wts/ws-mcp/$NAME"
      COMMITS=0
      if [ -d "$WT" ]; then
        COMMITS=$(cd "$WT" && git log --oneline origin/main..HEAD 2>/dev/null | wc -l | tr -d ' ')
      fi
      echo "    $NAME: $COMMITS commit(s)"
    done
  else
    echo "  No active workers"
  fi

  # Check for new merged PRs
  RECENT_PR=$(gh pr list --repo whitmo/ws-mcp --state merged --limit 1 --json number,title,mergedAt 2>/dev/null | python3 -c "
import json,sys
prs = json.load(sys.stdin)
if prs: print(f'  Latest merge: #{prs[0][\"number\"]} {prs[0][\"title\"]}')
" 2>/dev/null || true)
  [ -n "$RECENT_PR" ] && echo "$RECENT_PR"

  sleep "$INTERVAL"
done
