# Quickstart: MCP Bridge Service

## Prerequisites
- Go 1.21+
- `gh` CLI (for remote repository access)
- `multiclaude` CLI (for multi-agent orchestration)

## Initial Setup
1. **Initialize GitHub Repository**:
   ```bash
   gh repo create whitmo/ws-mcp --public --source=. --remote=origin --push
   ```
2. **Initialize Multiclaude**:
   ```bash
   multiclaude init github.com/whitmo/ws-mcp
   ```

## Running the Bridge
1. **Build and Run**:
   ```bash
   go build -o bridge ./src/cmd/bridge
   ./bridge --port 8080
   ```
2. **Post an Event (HTTP)**:
   ```bash
   curl -X POST http://localhost:8080/event \
     -H "Content-Type: application/json" \
     -d '{
       "id": "550e8400-e29b-41d4-a716-446655440000",
       "source": "system",
       "type": "bridge_ready",
       "ts": "2026-03-08T12:00:00Z",
       "payload": { "status": "online" }
     }'
   ```

## Connecting as an Observer
- **WebSocket (Real-time)**: Connect to `ws://localhost:8080/ws` using a tool like `wscat` or a custom client.
- **MCP (Structured Query)**: Configure your MCP-compatible agent to use the bridge via standard streams or HTTP endpoints.

## Testing (TDD)
Run all tests using the standard Go test runner:
```bash
go test ./tests/...
```
