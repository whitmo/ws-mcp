# Research Report: MCP Bridge Service (WebSocket + HTTP Ingest)

## Technical Decisions

### Decision: GitHub Repository Initialization
**Decision**: Use `gh` CLI to create the remote repository and push the current state.
**Rationale**: Simplifies the path to a shared source of truth, enabling `multiclaude` and `ralph` to interact with a canonical repository.
**Alternatives Considered**: Manual creation (too slow), skip for MVP (blocks `multiclaude init`).

### Decision: Multiclaude Integration
**Decision**: Run `multiclaude init` for the newly created repository.
**Rationale**: Enables the supervisor agent to manage worker agents for the project as per the user's directive.
**Implementation**: `multiclaude init github.com/whitmo/ws-mcp` (after repo creation).

### Decision: Go MCP Implementation
**Decision**: Use a custom JSON-RPC implementation or `mcp-go-sdk` if available and stable.
**Rationale**: Standard JSON-RPC 2.0 is highly stable and well-suited for Go. Avoiding complex dependencies for the MVP aligns with the "Simple Protocols" and "Disposable Modularity" principles.
**Research Finding**: The MCP protocol specifies JSON-RPC 2.0 over standard streams or HTTP. Go's `encoding/json` and `net/rpc` or standard handlers are sufficient.

### Decision: Ingest Mechanism (Ralph/Multiclaude)
**Decision**: Standard HTTP `POST /event` with JSON payload.
**Rationale**: Simplest interoperability. `ralph` can be configured to post events via basic `curl` or internal `net/http` calls.
**Format**:
```json
{
  "id": "uuid",
  "source": "ralph|multiclaude|system",
  "type": "agent_started|agent_finished|error|notification",
  "ts": "RFC3339",
  "payload": {}
}
```

### Decision: WebSocket Hub
**Decision**: Use `gorilla/websocket` with a fan-out hub pattern.
**Rationale**: `gorilla/websocket` is the industry standard for Go. Hub pattern ensures efficient broadcast without blocking ingest.

## Remaining Unknowns
- **MCP Go SDK Maturity**: Need to verify if there's a community-standard Go SDK; otherwise, proceed with custom JSON-RPC handlers for `events.latest`, `events.filter`, and `events.ack`.
