package integration

import (
	"context"
	"testing"
	"time"

	"github.com/whitmo/ws-mcp/src/internal/mcp"
	"github.com/whitmo/ws-mcp/src/internal/store"
	"github.com/whitmo/ws-mcp/src/internal/types"
)

func TestMCPQueryEndToEnd(t *testing.T) {
	rb := store.NewRingBuffer(10)
	handler := mcp.NewHandler(rb)

	// Simulate HTTP Ingest
	rb.Push(types.Event{
		ID:     "e2e-1",
		Source: types.SourceMultiClaude,
		Type:   "test",
		Ts:     time.Now(),
	})

	// Query via MCP
	events, err := handler.HandleLatest(context.Background(), 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(events) != 1 || events[0].ID != "e2e-1" {
		t.Fatalf("failed to query event via MCP handler")
	}
}
