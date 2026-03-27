<!--
Sync Impact Report:
- Version change: 1.0.0 → 1.1.0
- List of modified principles (Added):
    - VI. Test-Driven Documentation (TDD)
    - VII. Disposable Modularity
    - VIII. Deep Modules & Shallow Interfaces
    - IX. Incremental Atomic Progress
    - X. Wabi-sabi (Imperfection & Flow)
- Added sections: None
- Removed sections: None
- Templates requiring updates:
    - .specify/templates/plan-template.md (✅ updated)
    - .specify/templates/spec-template.md (✅ updated)
    - .specify/templates/tasks-template.md (✅ updated)
- Follow-up TODOs: Ensure TDD workflow is reflected in Task generation.
-->
# WS-MCP Bridge Constitution

## Core Principles

### I. Event-Centric Architecture
All agent interactions (Ralph, MultiClaude, System) MUST be encapsulated as standard JSON events. Events are the single source of truth for system state and inter-agent coordination.

### II. Real-time Observability
The bridge MUST broadcast ingested events over WebSocket with minimal latency (p95 < 150ms). Developers and agents should have immediate visibility into the event stream without polling.

### III. MCP-First Interoperability
Every bridge capability MUST be exposed via a structured Model Context Protocol (MCP) tool interface. Querying, filtering, and acknowledging events should be as natural for an AI as calling a function.

### IV. Bounded Resilience
The system MUST protect its own stability through bounded in-memory storage (Ring Buffers). Older events are dropped to ensure current events flow freely. Performance targets for ingest and broadcast are non-negotiable.

### V. Simple Protocol Handshakes
Avoid proprietary or complex transports. Use standard HTTP for ingest and WebSocket for broadcast. Keep the handshake and message envelopes minimal, human-readable, and machine-parsable.

### VI. Test-Driven Documentation (TDD)
A feature is not ready for implementation until its tests and documentation exist and are failing. Documentation MUST define the intent, and tests MUST verify the contract before a single line of application logic is written.

### VII. Disposable Modularity
Design for the dumpster. Components MUST be highly modular with narrow, well-defined interfaces so they can be completely replaced or thrown away without ripple effects. Avoid "marriage" to any specific implementation.

### VIII. Deep Modules & Shallow Interfaces
Complexity is a tax. Follow *A Philosophy of Software Design*: build "deep" modules that hide significant internal complexity behind a small, simple, and intuitive surface area.

### IX. Incremental Atomic Progress
Large changes are risks. Deliver value through small, frequent Pull Requests that are atomic and self-contained. Every PR MUST leave the codebase in a deployable, passing state.

### X. Wabi-sabi (Imperfection & Flow)
Embrace the beauty of the imperfect, impermanent, and incomplete. Prioritize shipping functional, "good enough" code that solves the immediate problem over chasing theoretical perfection. Refactor when the friction demands it, not before.

## Operational Standards
The Go service must maintain a low memory footprint (< 50MB default) and support graceful shutdown within 5 seconds to ensure clean service rotation in development and production environments.

## Security & Performance
All event data is considered trusted within the local network (no auth required for MVP), but all inputs MUST be validated against the event schema before being broadcasted or stored to prevent system corruption.

## Governance
This constitution supersedes all other documentation. Amendments require a version increment and an update to the `Sync Impact Report` to maintain alignment across all project templates and artifacts.

**Version**: 1.1.0 | **Ratified**: 2026-03-08 | **Last Amended**: 2026-03-08
