# Contract: WebSocket Stream (Real-time)

## Endpoint: `GET /ws`
Establishes a WebSocket connection for real-time event broadcasting.

### Connection Handshake
- **Upgrade Header**: `Upgrade: websocket`
- **Sec-WebSocket-Key**: Standard handshake key.

### Message Envelope (JSON)
All messages sent from the server to the client are JSON events.

```json
{
  "id": "uuid",
  "source": "ralph|multiclaude|system",
  "type": "string",
  "ts": "date-time",
  "payload": { "key": "value" },
  "acked": false
}
```

### Client Actions (Optional/MVP)
- **Ping/Pong**: Standard heartbeats.
- **Initial Connection**: Server MAY send the last 10 events as a "catch-up" upon connection.

### Failure Handling
- **Backpressure**: If a client is slow, the server will buffer up to a certain limit and then drop the client connection if it falls behind.
- **Heartbeat**: Clients should send Pings regularly to maintain the connection.
