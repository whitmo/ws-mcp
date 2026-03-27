#!/usr/bin/env bash
# Check multiclaude worker status and post results to the ws-mcp bridge.
# Usage: check-workers.sh [repo]
#
# Posts worker.status events for each worker, plus a pr.status summary.
#
# Environment:
#   WS_MCP_URL — Bridge base URL (default: http://localhost:8080)
set -euo pipefail

REPO="${1:-ws-mcp}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
POST="${SCRIPT_DIR}/post-event.sh"

# Parse multiclaude work list output (columns: NAME STATUS BRANCH MSGS TASK)
while IFS= read -r LINE; do
  # Skip header/separator lines
  [[ "$LINE" =~ ^(NAME|---) ]] && continue
  [ -z "$LINE" ] && continue

  WORKER=$(echo "$LINE" | awk '{print $1}')
  # Status field includes bullet character, grab the word after it
  STATUS=$(echo "$LINE" | awk '{gsub(/[●○]/, ""); print $2}')
  BRANCH=$(echo "$LINE" | awk '{print $3}')
  [ -z "$WORKER" ] && continue

  WT="${HOME}/.multiclaude/wts/${REPO}/${WORKER}"
  COMMITS=0
  LATEST=""
  if [ -d "$WT" ]; then
    COMMITS=$(cd "$WT" && git log --oneline origin/001-websocket-mcp-bridge..HEAD 2>/dev/null | wc -l | tr -d ' ')
    LATEST=$(cd "$WT" && git log --oneline -1 2>/dev/null | sed 's/"/\\"/g')
  fi

  "$POST" "worker.status" "multiclaude" \
    "{\"worker\":\"${WORKER}\",\"status\":\"${STATUS}\",\"branch\":\"${BRANCH}\",\"commits\":${COMMITS},\"latest\":\"${LATEST}\"}"
done < <(multiclaude work list --repo "$REPO" 2>&1 | grep -E '●|○')

# Post merged PR summary
PR_COUNT=$(gh pr list --repo "whitmo/${REPO}" --state merged --limit 20 --json number,title | python3 -c "import json,sys; print(len(json.load(sys.stdin)))" 2>/dev/null || echo "0")
"$POST" "pr.summary" "system" "{\"repo\":\"${REPO}\",\"merged_count\":${PR_COUNT}}"

echo "Worker status posted to bridge."
