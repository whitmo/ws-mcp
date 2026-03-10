# Feature Specification: MCP Bridge Service (WebSocket + HTTP Ingest)

**Feature Branch**: `001-websocket-mcp-bridge`  
**Created**: 2026-03-08  
**Status**: Draft  
**Input**: User description: "Go MCP bridge with HTTP ingest + WebSocket broadcast and MCP query tools, accepting events from Ralph and MultiClaude"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Real-time Event Monitoring (Priority: P1)

As a developer using Ralph or MultiClaude, I want to see a live stream of events as they happen so I can monitor the progress of my autonomous agents without manually polling or checking logs.

**Why this priority**: This is the core value proposition of the bridge - real-time observability across multiple agent instances.

**Independent Test**: Can be tested by connecting a WebSocket client to `/ws`, sending a `POST /event` from any source, and verifying the event appears in the WS stream immediately.

**Acceptance Scenarios**:

1. **Given** the bridge is running and a WebSocket client is connected to `/ws`, **When** a JSON event is posted to `/event`, **Then** the WebSocket client receives the exact same event payload in real-time.
2. **Given** multiple WebSocket clients are connected, **When** an event is ingested, **Then** ALL connected clients receive the broadcast.

---

### User Story 2 - Structured Event Querying (Priority: P2)

As an MCP-compatible client (like Gemini or Claude), I want to query the bridge for the latest events or filter them by source/type so I can programmatically analyze the agent activity history.

**Why this priority**: Enables higher-level agents to "see" what other agents have done, which is critical for multi-agent coordination.

**Independent Test**: Can be tested by ingesting several events and then using the MCP `events.latest` tool to retrieve them in the correct order.

**Acceptance Scenarios**:

1. **Given** 5 events have been ingested, **When** I call the MCP tool `events.latest(limit=3)`, **Then** I receive the 3 most recent events in descending chronological order.
2. **Given** events from both 'ralph' and 'multiclaude' sources, **When** I call `events.filter(source='ralph')`, **Then** only events originating from Ralph are returned.

---

### User Story 3 - Event Acknowledgment (Priority: P3)

As a supervisor agent, I want to mark specific events as "acknowledged" so that other agents know a task or notification has been seen and handled.

**Why this priority**: Reduces redundant work and provides a basic mechanism for coordination/handoff.

**Independent Test**: Can be tested by calling `events.ack(id)` and then verifying the event state via `events.latest`.

**Acceptance Scenarios**:

1. **Given** an unacknowledged event exists in the store, **When** I call `events.ack(id)`, **Then** the event's `acked` property becomes `true` and `acked_ts` is recorded.

---

### Edge Cases

- **Ring Buffer Overflow**: When the number of events exceeds the limit (default 2000), the oldest events must be dropped to make room for new ones without crashing the service.
- **Malformed Payloads**: If a `POST /event` contains invalid JSON or is missing required fields (id, source, type, ts), the service must return a 400 Bad Request and not broadcast or store the event.
- **Slow WS Consumers**: If a WebSocket client is slow to read, the server should not block other clients or the ingest process (using buffered channels or dropping slow clients).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide a `POST /event` endpoint that accepts JSON payloads with `id`, `source`, `type`, `ts`, and `payload`.
- **FR-002**: System MUST validate that `source` is one of `ralph`, `multiclaude`, or `system`.
- **FR-003**: System MUST provide a `GET /ws` endpoint for real-time event broadcasting.
- **FR-004**: System MUST maintain an in-memory ring buffer of events (default size 2000).
- **FR-005**: System MUST expose MCP tools: `events.latest`, `events.filter`, `events.ack`, and `report.summary`.
- **FR-006**: System MUST support optional local system notifications (e.g., audio alerts) for specific event types.
- **FR-007**: System SHOULD support a generic webhook URL for external event fanout.

### Key Entities *(include if feature involves data)*

- **Event**: Represents a single point-in-time occurrence from an agent or the system. Includes identity, source, timestamp, and a flexible data payload.
- **RingBuffer**: The internal storage mechanism that manages the bounded set of recent events in memory.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Event ingest latency (from POST request to storage) is under 100ms for 95% of requests.
- **SC-002**: WebSocket broadcast latency (from ingest to send) is under 150ms for 95% of clients.
- **SC-003**: System maintains a stable memory footprint under 50MB when the ring buffer is full (at default settings).
- **SC-004**: 100% of events sent via `POST /event` are either stored/broadcasted or rejected with an error message.
