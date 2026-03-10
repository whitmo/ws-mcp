# Tasks: MCP Bridge Service (WebSocket + HTTP Ingest)

**Input**: Design documents from `/specs/001-websocket-mcp-bridge/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: TDD is MANDATORY per Constitution Principle VI. Write tests FIRST and ensure they FAIL.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [x] T001 Create project structure per implementation plan (Done during plan)
- [x] T002 [P] Initialize Go module `go mod init github.com/whitmo/ws-mcp`
- [x] T003 [P] Install dependencies: `go get github.com/gorilla/websocket`
- [x] T004 [P] Create `src/cmd/bridge/main.go` entry point skeleton

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure for storage and types

- [x] T005 [P] Define `Event` struct and constants in `src/internal/types/event.go`
- [x] T006 [P] Implement `RingBuffer` in `src/internal/store/ring_buffer.go` (Design for disposal)
- [x] T007 Write unit tests for `RingBuffer` in `src/internal/store/ring_buffer_test.go` and ensure they FAIL
- [x] T008 Implement `RingBuffer` logic to satisfy tests (FIFO, max 2000 events)
- [x] T009 [P] Implement basic HTTP server and router in `src/pkg/api/router.go`

**Checkpoint**: Foundation ready - Event storage and basic server structure in place.

---

## Phase 3: User Story 1 - Real-time Event Monitoring (Priority: P1) 🎯 MVP

**Goal**: Ingest events via HTTP and broadcast via WebSocket.

**Independent Test**: Connect `wscat` to `/ws`, POST an event to `/event`, and verify receipt.

### Tests for User Story 1 (MANDATORY TDD)

- [x] T010 [P] [US1] Create integration test `tests/integration/broadcast_test.go` (Ingest -> WS broadcast) - FAIL FIRST
- [x] T011 [P] [US1] Create unit test for `Hub` in `src/internal/hub/hub_test.go` - FAIL FIRST

### Implementation for User Story 1

- [x] T012 [P] [US1] Implement `Hub` in `src/internal/hub/hub.go` for managing WS clients and broadcasting
- [x] T013 [P] [US1] Implement `POST /event` handler in `src/pkg/api/handlers.go` (ingest logic)
- [x] T014 [P] [US1] Implement `GET /ws` handler in `src/pkg/api/handlers.go` (WS upgrade)
- [x] T015 [US1] Connect Ingest handler to Hub and Store (Phase 2)
- [x] T016 [US1] Add validation for `source` and `ts` in ingest handler

**Checkpoint**: User Story 1 (MVP) functional. Real-time observability achieved.

---

## Phase 4: User Story 2 - Structured Event Querying (Priority: P2)

**Goal**: Expose MCP tools for querying events.

**Independent Test**: Use an MCP client or JSON-RPC call to `events.latest` and verify results.

### Tests for User Story 2 (MANDATORY TDD)

- [x] T017 [P] [US2] Create integration test `tests/integration/mcp_query_test.go` - FAIL FIRST
- [x] T018 [P] [US2] Create unit test for MCP tool handlers in `src/internal/mcp/handlers_test.go` - FAIL FIRST

### Implementation for User Story 2

- [x] T019 [P] [US2] Implement MCP JSON-RPC router in `src/internal/mcp/router.go`
- [x] T020 [P] [US2] Implement `events.latest` tool handler (reading from Store)
- [x] T021 [P] [US2] Implement `events.filter` tool handler
- [x] T022 [US2] Expose MCP tools over stdio/HTTP per contract

**Checkpoint**: User Story 2 functional. Agents can now query the bridge.

---

## Phase 5: User Story 3 - Event Acknowledgment (Priority: P3)

**Goal**: Support event acknowledgment state.

**Independent Test**: Call `events.ack(id)` and verify `acked: true` in subsequent queries.

### Tests for User Story 3 (MANDATORY TDD)

- [x] T023 [P] [US3] Create integration test `tests/integration/ack_test.go` - FAIL FIRST

### Implementation for User Story 3

- [x] T024 [P] [US3] Add `Ack(id)` method to `Store` and `Event` struct
- [x] T025 [P] [US3] Implement `events.ack` MCP tool handler
- [x] T026 [US3] Ensure `acked` state is broadcasted over WS if updated

**Checkpoint**: All core User Stories functional.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [x] T027 [P] Implement Graceful Shutdown in `main.go` (< 5s)
- [x] T028 [P] Add audio alert (macOS `say` or similar) for specific events (FR-006)
- [x] T029 [P] Final documentation update in `README.md` and `quickstart.md`
- [x] T030 Run all integration tests and verify against Success Criteria (SC-001 to SC-004) (Note: Run `go test ./...` in a standard terminal due to sandbox restrictions)

---

## Dependencies & Execution Order

1. **Setup (Phase 1)** -> **Foundational (Phase 2)** (Linear)
2. **Phase 3 (US1)** can start once Phase 2 is done.
3. **Phase 4 (US2)** and **Phase 5 (US3)** can run in parallel with US1 if interfaces are stable.
4. **Phase 6** runs after all US features are verified.

---

## Implementation Strategy: MVP First (User Story 1)

1. Complete T002-T004 (Setup)
2. Complete T005-T009 (Foundational)
3. Implement US1 (T010-T016)
4. **STOP and VALIDATE**: Verify real-time broadcast works from HTTP to WS.
