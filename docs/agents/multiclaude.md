# Using ws-mcp with Multiclaude

Multiclaude workers are Claude Code sessions running in isolated git worktrees. Each worker inherits Claude Code's MCP configuration, so they automatically have access to ws-mcp tools.

## How workers communicate through the bridge

### Workers → Bridge (posting events)

Workers can post events using the helper scripts:

```bash
# Announce starting work
bash scripts/post-event.sh task.started multiclaude '{"worker":"clever-wolf","task":"implement auth"}'

# Announce completion
bash scripts/post-event.sh task.completed multiclaude '{"worker":"clever-wolf","pr":15}'

# Request a review
bash scripts/post-event.sh review.requested multiclaude '{"pr":15,"reviewer":"gemini"}'
```

Or use the multiclaude-specific hook:
```bash
bash scripts/multiclaude-hook.sh worker.started clever-wolf
bash scripts/multiclaude-hook.sh worker.completed clever-wolf '{"pr":15}'
```

### Bridge → Workers (reading events)

Workers have MCP tools available:

- **events_latest** — See what's happening across all agents
- **events_filter** — Filter by source to see ralph or system events
- **events_ack** — Acknowledge events
- **report_summary** — Get activity summary

### Supervisor → Bridge (monitoring)

The supervisor session can monitor all workers through the bridge:

```bash
# Post status for all workers
bash scripts/check-workers.sh ws-mcp

# Query the bridge for worker events
bash scripts/query-events.sh events.filter '{"source":"multiclaude"}'

# Get overall activity summary
bash scripts/query-events.sh report.summary '{"window":30}'
```

## Lifecycle hook integration

Use `scripts/multiclaude-hook.sh` to post worker lifecycle events:

| Event | Usage |
|---|---|
| `worker.started` | `multiclaude-hook.sh worker.started <name>` |
| `worker.committed` | `multiclaude-hook.sh worker.committed <name> '{"sha":"abc123"}'` |
| `worker.pushed` | `multiclaude-hook.sh worker.pushed <name> '{"branch":"work/name"}'` |
| `worker.completed` | `multiclaude-hook.sh worker.completed <name> '{"pr":15}'` |

### Wiring into Claude Code hooks

To automatically fire events, add to `~/.claude/settings.json`:

```json
{
  "hooks": {
    "SessionStart": [{
      "matcher": "",
      "hooks": [{
        "type": "command",
        "command": "HOOK_EVENT=session.start \"$HOME/.claude/hooks/ws-mcp-notify.sh\""
      }]
    }],
    "Stop": [{
      "matcher": "",
      "hooks": [{
        "type": "command",
        "command": "HOOK_EVENT=session.stop \"$HOME/.claude/hooks/ws-mcp-notify.sh\""
      }]
    }]
  }
}
```

## Setup

### Prerequisites

1. Bridge binary installed at `~/bin/ws-mcp-bridge`
2. Bridge HTTP server running:
   ```bash
   bash scripts/bridge-start.sh
   ```
3. ws-mcp in `.mcp.json` (already configured):
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

### Verifying

```bash
# Spawn a worker
multiclaude work "test task" --repo ws-mcp

# Check bridge sees worker activity
bash scripts/query-events.sh events.latest '{"limit":5}'
```

## Coordination patterns

**Parallel workers checking for conflicts:**
Each worker calls `events_latest` before touching shared files to see if another worker is already modifying them.

**Worker requesting review:**
Worker posts `review.requested` event → reactor dispatches to `multiclaude review $PR` → review agent picks it up.

**Supervisor monitoring swarm:**
```bash
# Continuous monitoring
watch -n 10 'bash scripts/check-workers.sh ws-mcp && bash scripts/query-events.sh report.summary'
```

## Architecture

```
Supervisor (this session)
  ├── spawns workers via: multiclaude work "task"
  ├── monitors via: scripts/check-workers.sh → bridge
  └── queries via: scripts/query-events.sh

Worker (isolated worktree)
  ├── MCP tools: events_latest, events_filter, events_ack
  ├── posts events: scripts/post-event.sh, scripts/multiclaude-hook.sh
  └── inherits Claude Code hooks (SessionStart/Stop → bridge)

Bridge (localhost:8080)
  ├── HTTP ingest: POST /event
  ├── WebSocket broadcast: GET /ws
  ├── JSON-RPC: POST /rpc
  └── MCP stdio: ws-mcp-bridge --stdio
```
