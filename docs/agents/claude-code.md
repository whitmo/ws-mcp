# Using ws-mcp with Claude Code

You are connected to the **ws-mcp bridge** — an event bus that lets you communicate with other AI agents (Codex, Gemini, Ralph, multiclaude workers) in real time.

## What you can do

You have MCP tools available:

- **events_latest** — See what other agents have been doing. Call with `{"limit": 20}` to get recent events.
- **events_filter** — Filter by source: `ralph`, `multiclaude`, or `system`.
- **events_ack** — Acknowledge an event so other agents know you've seen it. Call with `{"id": "<event_id>", "acked_by": "claude-code"}`.
- **report_summary** — Get a summary of all activity in the last N minutes: `{"window": 60}`.

## How to talk to other agents

Post events to the bridge via HTTP:

```bash
curl -X POST http://localhost:8080/event \
  -H "Content-Type: application/json" \
  -d '{
    "id": "unique-id",
    "source": "system",
    "type": "review.requested",
    "ts": "2026-03-27T12:00:00Z",
    "payload": {"pr": 42, "message": "Please review this PR"}
  }'
```

Or use the helper script:
```bash
bash scripts/post-event.sh review.requested system '{"pr":42,"message":"Please review"}'
```

## How to listen for events

Use the observer CLI to watch the event stream:
```bash
go run ./src/cmd/observer --pretty --source ralph
```

Or query via your MCP tools — call `events_latest` periodically to check for new events.

## Event types you'll see

| Type | Source | Meaning |
|---|---|---|
| `task.started` | any | An agent started a task |
| `task.completed` | any | An agent finished a task |
| `task.failed` | any | An agent's task failed |
| `worker.status` | multiclaude | A multiclaude worker status update |
| `review.requested` | any | Someone wants a code review |
| `loop.start` | ralph | Ralph started an orchestration loop |
| `system.error` | system | Something went wrong |

## Setup

ws-mcp is already configured as an MCP server. If you need to add it manually:

**Project-level** (`.mcp.json` in repo root):
```json
{
  "mcpServers": {
    "ws-mcp": {
      "command": "ws-mcp-bridge",
      "args": ["--stdio"]
    }
  }
}
```

**Global** (`~/.claude/settings.json`):
```json
{
  "mcpServers": {
    "ws-mcp": {
      "command": "ws-mcp-bridge",
      "args": ["--stdio"]
    }
  }
}
```

### Prerequisites

1. Build and install the bridge binary:
   ```bash
   cd /path/to/ws-mcp
   go build -o ~/bin/ws-mcp-bridge ./src/cmd/bridge
   ```

2. Ensure `~/bin` is on your PATH.

3. For the HTTP server (needed for event posting and WebSocket):
   ```bash
   bash scripts/bridge-start.sh
   ```

## Coordination patterns

**Check what's happening before starting work:**
Call `events_latest` with `{"limit":20}` to see recent activity.

**Announce what you're doing:**
```bash
bash scripts/post-event.sh task.started system '{"agent":"claude-code","description":"refactoring auth module"}'
```

**Request a review from another agent:**
```bash
bash scripts/post-event.sh review.requested system '{"pr":42,"reviewer":"gemini"}'
```

**Acknowledge events you've handled:**
Call `events_ack` with `{"id":"<event_id>","acked_by":"claude-code"}`.
