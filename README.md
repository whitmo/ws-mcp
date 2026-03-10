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

## Running Tests
Tests are designed under a Test-Driven Development (TDD) philosophy.
```bash
go test ./...
```
