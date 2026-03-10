# Implementation Plan: MCP Bridge Service (WebSocket + HTTP Ingest)

**Branch**: `001-websocket-mcp-bridge` | **Date**: 2026-03-08 | **Spec**: [specs/001-websocket-mcp-bridge/spec.md](specs/001-websocket-mcp-bridge/spec.md)
**Input**: Feature specification for a Go-based MCP bridge with HTTP ingest and WebSocket broadcast.

## Summary
The goal is to build a high-performance Go service that aggregates events from `ralph` and `multiclaude`, broadcasts them to WebSocket subscribers for real-time monitoring, and exposes an MCP interface for structured querying and acknowledgment. The implementation will follow TDD and modularity principles from the constitution.

## Technical Context

**Language/Version**: Go 1.21+  
**Primary Dependencies**: `gorilla/websocket`, standard `net/http`, MCP Go SDK (or custom JSON-RPC)  
**Storage**: In-memory Ring Buffer (bounded at 2000 events)  
**Testing**: `go test` with TDD workflow (Tests/Docs first)  
**Target Platform**: Linux / macOS (Local Network)
**Project Type**: Web Service / Bridge  
**Performance Goals**: Ingest Latency < 100ms (p95), WS Broadcast Latency < 150ms (p95)  
**Constraints**: < 50MB Memory Footprint, Graceful Shutdown < 5s  
**Scale/Scope**: 2000 events, support for multiple concurrent WS observers and MCP clients

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Compliance Note |
|-----------|--------|-----------------|
| I. Event-Centric | ✅ Pass | All inputs/outputs follow the Event schema. |
| II. Real-time Obs | ✅ Pass | WebSocket broadcast is a core requirement. |
| III. MCP-First | ✅ Pass | MCP tools are primary query interface. |
| IV. Bounded Resilience | ✅ Pass | Ring Buffer explicitly defined in spec. |
| V. Simple Protocols | ✅ Pass | Using HTTP and WebSocket. |
| VI. TDD | ✅ Pass | Planned in Phase 0/1 tasks. |
| VII. Modularity | ✅ Pass | Separation of Hub, Store, and MCP handlers. |
| VIII. Deep Modules | ✅ Pass | Internal ring buffer logic hidden behind Store interface. |
| IX. Incremental PRs | ✅ Pass | Implementation broken into atomic phases. |
| X. Wabi-sabi | ✅ Pass | MVP focuses on local trusted network first. |

## Project Structure

### Documentation (this feature)

```text
specs/001-websocket-mcp-bridge/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
└── tasks.md             # Phase 2 output (to be generated)
```

### Source Code (repository root)

```text
src/
├── cmd/
│   └── bridge/          # Main entry point
├── internal/
│   ├── hub/             # WebSocket hub and client management
│   ├── store/           # In-memory ring buffer implementation
│   ├── mcp/             # MCP tool handlers and JSON-RPC logic
│   └── types/           # Shared event schemas and interfaces
├── pkg/
│   └── api/             # HTTP handlers and routing
└── tests/
    ├── integration/     # End-to-end ingest/broadcast tests
    └── unit/            # Component-specific tests (Store, Hub)
```

**Structure Decision**: Single Go project with standard layout (`cmd/`, `internal/`, `pkg/`).

## Complexity Tracking

*No violations detected.*
