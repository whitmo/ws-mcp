# Using ws-mcp with Ralph Orchestrator

Ralph is integrated with ws-mcp through two mechanisms: **hooks** (Ralph → bridge) and **MCP tools** (bridge → Ralph's agents).

## Hooks: Ralph → Bridge

Ralph automatically posts lifecycle events to the bridge via hooks configured in `ralph.yml`:

| Ralph Event | Bridge Event Type | When |
|---|---|---|
| `post.loop.start` | `loop.start` | Ralph starts an orchestration loop |
| `post.iteration.start` | `iteration.start` | Each iteration begins |
| `post.loop.complete` | `loop.complete` | Loop finishes successfully |
| `post.loop.error` | `loop.error` | Loop encounters an error |
| `post.plan.created` | `plan.created` | A plan is generated |

These are already wired in `ralph.yml`. Events are posted via `scripts/ralph-hook.sh`.

### Adding custom hooks

To post events for additional ralph lifecycle points, add to `ralph.yml`:

```yaml
hooks:
  events:
    post.hat.changed:
      - name: "ws-mcp-bridge"
        command: ["./scripts/ralph-hook.sh", "hat.changed"]
        on_error: warn
```

### Manual event posting from ralph agents

Ralph's backend agents (Claude, Gemini, etc.) can post events directly:

```bash
./scripts/ralph-hook.sh task.started ralph '{"task":"review code","hat":"reviewer"}'
```

## MCP Tools: Bridge → Ralph Agents

When Ralph runs with a Claude backend, the backend agent has access to ws-mcp as an MCP server (via `.mcp.json`). The agent can:

- **events_latest** — Check what other agents have done recently
- **events_filter** — Filter by source to see only multiclaude or system events
- **events_ack** — Acknowledge events to signal they've been handled
- **report_summary** — Get activity summary to inform decision-making

### Example: Ralph agent checking for pending reviews

During a ralph loop iteration, the backend agent can call `events_filter` with `{"source":"multiclaude"}` to see if any workers have completed, then decide whether to review their PRs.

## Setup

### Prerequisites

1. Bridge binary installed at `~/bin/ws-mcp-bridge`
2. Bridge HTTP server running on port 8080
3. `ralph.yml` in the repo root with hooks configured (already done)

### Verifying hooks work

```bash
# Start the bridge
bash scripts/bridge-start.sh

# Run ralph (hooks will fire automatically)
ralph run -p "test task" -H builtin:code-assist

# Check bridge for ralph events
bash scripts/query-events.sh events.filter '{"source":"ralph"}'
```

### Running ralph as MCP server alongside ws-mcp

Both ralph-orchestrator and ws-mcp can run as MCP servers simultaneously. The `.mcp.json` includes both:

```json
{
  "mcpServers": {
    "ralph-orchestrator": {
      "command": "ralph",
      "args": ["mcp", "serve"]
    },
    "ws-mcp": {
      "command": "ws-mcp-bridge",
      "args": ["--stdio"]
    }
  }
}
```

This gives backend agents access to both Ralph's control plane (loops, hats, tasks) and the ws-mcp event bus.

## Integration architecture

```
Ralph Loop
  ├── hook fires → ralph-hook.sh → POST /event → bridge → WS broadcast
  └── backend agent (Claude/Gemini)
        ├── ralph MCP tools (loops, hats, tasks)
        └── ws-mcp MCP tools (events, filtering, ack)
```
