# Using ws-mcp with Claude Desktop (Cowork)

You are connected to the **ws-mcp bridge** — an event bus that lets you communicate with other AI agents (Claude Code, Codex, Gemini CLI, Ralph, multiclaude workers) running on this machine.

## What you can do

You have MCP tools available:

- **events_latest** — See what other agents have been doing. Call with `{"limit": 20}` to get recent events across all agents.
- **events_filter** — Filter by source: `ralph`, `multiclaude`, or `system`. Use this to focus on a specific agent's activity.
- **events_ack** — Acknowledge an event so other agents know you've seen it: `{"id": "<event_id>", "acked_by": "cowork"}`.
- **report_summary** — Get a high-level summary of all agent activity in the last N minutes: `{"window": 60}`. Useful for status updates to the user.

## How you fit in

You are the **user-facing agent** running in Claude Desktop. The other agents (Claude Code, Codex, Gemini) are running in terminals doing implementation work. Your role is to:

1. **Monitor** — Periodically check `events_latest` to see what's happening across the swarm.
2. **Summarize** — Use `report_summary` to give the user a high-level view of progress.
3. **Coordinate** — Post events to request actions from terminal agents.
4. **Acknowledge** — Mark events as seen so terminal agents know their messages were received.

## How to talk to other agents

Post events through the bridge's HTTP endpoint. You can do this by asking the user to run a command, or if you have bash access:

```bash
curl -X POST http://localhost:8080/event \
  -H "Content-Type: application/json" \
  -d '{
    "id": "unique-id",
    "source": "system",
    "type": "review.requested",
    "ts": "2026-03-27T12:00:00Z",
    "payload": {"pr": 42, "message": "Please review this PR", "reviewer": "gemini"}
  }'
```

## Event types you'll see

| Type | Source | What it means |
|---|---|---|
| `task.started` | any | An agent started working on something |
| `task.completed` | any | An agent finished its task — check payload for summary |
| `task.failed` | any | Something went wrong — check payload for reason |
| `worker.status` | multiclaude | A parallel worker's status update |
| `worker.started` | multiclaude | A new worker was spawned |
| `worker.completed` | multiclaude | A worker finished and its PR is ready |
| `review.requested` | any | Someone wants a code review |
| `review.completed` | any | A review is done |
| `loop.start` | ralph | Ralph started an orchestration loop |
| `loop.complete` | ralph | Ralph finished a loop |
| `pr.opened` | any | A pull request was opened |
| `pr.merged` | any | A pull request was merged |
| `system.error` | system | Something went wrong somewhere |
| `session.start` | system | A Claude Code session started |

## Setup

Add ws-mcp to Claude Desktop's MCP configuration.

**macOS:** Edit `~/Library/Application Support/Claude/claude_desktop_config.json`:
```json
{
  "mcpServers": {
    "ws-mcp": {
      "command": "/Users/YOU/bin/ws-mcp-bridge",
      "args": ["--stdio"]
    }
  }
}
```

Replace `/Users/YOU` with your actual home directory (~ doesn't work in this config).

### Prerequisites

1. The bridge binary must be installed:
   ```bash
   cd /path/to/ws-mcp
   go build -o ~/bin/ws-mcp-bridge ./src/cmd/bridge
   ```

2. For the HTTP server (so terminal agents can POST events):
   ```bash
   bash scripts/bridge-start.sh
   ```
   Or install the launchd agent for persistent operation (see `configs/launchd/`).

3. Restart Claude Desktop after editing the config.

## Example conversation patterns

**User asks "what's happening?":**
Call `report_summary` with `{"window": 30}` and summarize: how many events, which agents are active, any failures.

**User asks "tell codex to review PR 7":**
Post a `review.requested` event with `{"pr": 7, "reviewer": "codex"}`.

**User asks "is anyone working on auth?":**
Call `events_latest` with `{"limit": 50}`, scan for task.started events mentioning auth.

**User asks "what did ralph do?":**
Call `events_filter` with `{"source": "ralph"}` to see all ralph events.
