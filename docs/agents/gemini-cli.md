# Using ws-mcp with Gemini CLI

You are connected to the **ws-mcp bridge** — an event bus that lets you communicate with other AI agents (Claude Code, Codex, Ralph, multiclaude workers) in real time.

## What you can do

You have MCP tools available:

- **events_latest** — See what other agents have been doing. Call with `{"limit": 20}`.
- **events_filter** — Filter by source: `ralph`, `multiclaude`, or `system`.
- **events_ack** — Acknowledge an event: `{"id": "<event_id>", "acked_by": "gemini"}`.
- **report_summary** — Activity summary for the last N minutes: `{"window": 60}`.

## How to talk to other agents

Post events to the bridge:

```bash
curl -X POST http://localhost:8080/event \
  -H "Content-Type: application/json" \
  -d '{
    "id": "unique-id",
    "source": "system",
    "type": "task.started",
    "ts": "2026-03-27T12:00:00Z",
    "payload": {"agent": "gemini", "description": "analyzing codebase"}
  }'
```

Or use the helper:
```bash
bash scripts/post-event.sh task.started system '{"agent":"gemini","description":"analyzing codebase"}'
```

## Event types you'll see

| Type | Source | Meaning |
|---|---|---|
| `task.started` | any | An agent started a task |
| `task.completed` | any | An agent finished a task |
| `task.failed` | any | An agent's task failed |
| `worker.status` | multiclaude | A multiclaude worker status update |
| `review.requested` | any | Someone wants a code review |
| `review.completed` | any | Review finished |
| `loop.start` | ralph | Ralph started an orchestration loop |
| `system.error` | system | Something went wrong |

## Setup

ws-mcp is already configured as a project-level MCP server. To verify:
```bash
gemini mcp list
```

You should see `ws-mcp` in the list. If not:
```bash
gemini mcp add ws-mcp ws-mcp-bridge -- --stdio
```

Or add it to `.gemini/settings.json`:
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

3. For the HTTP server (event posting and WebSocket):
   ```bash
   bash scripts/bridge-start.sh
   ```

## Coordination patterns

**Check recent activity before starting:**
Call `events_latest` to see what other agents are working on. Avoid duplicating work.

**Announce your work:**
Post a `task.started` event so others know what you're doing.

**Dialectical review:**
When another agent posts `review.requested`, use `events_latest` to find it, review the work, and post `review.completed` with your findings.
