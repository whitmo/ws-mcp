# Contract: MCP Tools (Query & Control)

The MCP interface allows structured access to the bridge state via JSON-RPC 2.0.

## Tools Defined

### `events.latest`
Retrieve the most recent events from the ring buffer.

- **Inputs**:
  - `limit` (Integer, default=10, max=100)
- **Output**: Array of `Event` objects in descending chronological order.

### `events.filter`
Retrieve events filtered by source, type, or time.

- **Inputs**:
  - `source` (String, optional)
- **Output**: Array of `Event` objects matching the filters.

### `events.ack`
Acknowledge a specific event by ID.

- **Inputs**:
  - `id` (UUID, required)
  - `acked_by` (String, optional, agent identity)
- **Output**: The updated `Event` object or an error if the ID is not found.

### `report.summary`
Generate a high-level summary of agent activity over a window of time.

- **Inputs**:
  - `window` (Integer, minutes, default=60)
- **Output**: Object with event counts by source and type, and any system alerts.

## Protocol Detail
- **Transport**: Standard streams (stdio) or HTTP/SSE.
- **Message Format**: JSON-RPC 2.0 Request/Response.
