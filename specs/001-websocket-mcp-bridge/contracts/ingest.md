# Contract: Event Ingest (HTTP)

## Endpoint: `POST /event`
Accepts a JSON payload representing a single agent or system event.

### Request Body Schema
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "id": { "type": "string", "format": "uuid" },
    "source": { "enum": ["ralph", "multiclaude", "system"] },
    "type": { "type": "string" },
    "ts": { "type": "string", "format": "date-time" },
    "payload": { "type": "object" }
  },
  "required": ["id", "source", "type", "ts", "payload"]
}
```

### Responses
- **202 Accepted**: Event successfully parsed, stored, and queued for broadcast.
- **400 Bad Request**: Malformed JSON or validation failure.
- **500 Internal Server Error**: Ring buffer or server failure.

### Example
```bash
curl -X POST http://localhost:8080/event \
  -H "Content-Type: application/json" \
  -d '{
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "source": "ralph",
    "type": "task_started",
    "ts": "2026-03-08T12:00:00Z",
    "payload": { "task_name": "build_bridge" }
  }'
```
