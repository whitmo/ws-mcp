# WS-MCP Bridge

A Go-based service acting as an event bridge between autonomous agents (like Ralph and MultiClaude) and WebSocket clients. It exposes an HTTP ingest endpoint, a real-time WebSocket broadcast hub, and an MCP interface for structured querying.

## Features
- **Event Ingest**: `POST /event` accepts standard JSON events.
- **Real-time Broadcast**: `GET /ws` streams ingested events to observers.
- **MCP Tool Surface**: Query recent events (`events.latest`) or acknowledge tasks (`events.ack`).
- **Bounded Resilience**: In-memory ring buffer prevents unbounded memory growth.

## Quickstart
See `specs/001-websocket-mcp-bridge/quickstart.md` for full setup instructions.

```bash
go mod tidy
go build -o bridge ./src/cmd/bridge
./bridge
```

## MCP Server Configuration

ws-mcp can be used as an MCP server by AI agents over stdio transport. The repo includes a `.mcp.json` config that tools like Claude Code auto-detect.

### Claude Code

Add to your Claude Code settings (`~/.claude/settings.json` or project `.claude/settings.json`):

```json
{
  "mcpServers": {
    "ws-mcp": {
      "command": "go",
      "args": ["run", "./src/cmd/bridge", "--mode", "mcp"],
      "cwd": "/path/to/ws-mcp",
      "env": { "WS_MCP_PORT": "8080" }
    }
  }
}
```

Or place a `.mcp.json` in the repo root (already included).

### Gemini CLI

Add to your Gemini settings (`~/.gemini/settings.json`):

```json
{
  "mcpServers": {
    "ws-mcp": {
      "command": "go",
      "args": ["run", "./src/cmd/bridge", "--mode", "mcp"],
      "cwd": "/path/to/ws-mcp"
    }
  }
}
```

### Codex

Add to your Codex MCP config:

```json
{
  "mcpServers": {
    "ws-mcp": {
      "command": "/path/to/ws-mcp/scripts/mcp-serve.sh"
    }
  }
}
```

### Start manually

```bash
./scripts/mcp-serve.sh
```

## Ralph Integration

The bridge integrates with [Ralph Orchestrator](https://github.com/mikeyobrien/ralph-orchestrator) via lifecycle hooks. When ralph runs, hook events (loop start/complete, iteration start, errors, plan creation) are automatically POSTed to the bridge.

### Setup

The `ralph.yml` in this repo already has hooks configured. Just ensure the bridge is running:

```bash
./bridge &
ralph run
```

### Standalone usage

You can also fire events manually:

```bash
# Basic event
./scripts/ralph-hook.sh loop.start

# With custom source and payload
./scripts/ralph-hook.sh iteration.complete ralph '{"iteration": 3, "hat": "builder"}'
```

Environment variables:
- `WS_MCP_URL` — Bridge base URL (default: `http://localhost:8080`)

### Test the integration

```bash
./scripts/test-ralph-hook.sh
```

This builds the bridge, starts it, posts a test event via `ralph-hook.sh`, and verifies it arrives via the `/rpc` `events.latest` query.

## Smoke Test

Run the round-trip smoke test (requires the bridge port to be free):

```bash
./scripts/smoke-test.sh
```

This builds the bridge, starts it, verifies the healthcheck, posts an event via HTTP, and optionally checks WebSocket delivery (if `websocat` or `wscat` is installed).

## Running Tests
Tests are designed under a Test-Driven Development (TDD) philosophy.
```bash
go test ./...
```
