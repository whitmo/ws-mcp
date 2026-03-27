package integration

import (
	"context"
	"testing"
	"time"

	"github.com/whitmo/ws-mcp/src/internal/mcp"
	"github.com/whitmo/ws-mcp/src/internal/store"
	"github.com/whitmo/ws-mcp/src/internal/types"
)

func TestAckIntegration(t *testing.T) {
	rb := store.NewRingBuffer(10)
	handler := mcp.NewHandler(rb)

	// Simulate HTTP Ingest
	rb.Push(types.Event{
		ID:     "ack-1",
		Source: types.SourceMultiClaude,
		Type:   "test",
		Ts:     time.Now(),
	})

	// 1. Call MCP Ack
	err := handler.HandleAck(context.Background(), "ack-1", "supervisor-agent")
	if err != nil {
		t.Fatalf("expected no error on ack, got %v", err)
	}

	// 2. Verify state
	events, _ := handler.HandleLatest(context.Background(), 1)
	if !events[0].Acked {
		t.Fatalf("expected event to be acked")
	}
	if events[0].AckedBy != "supervisor-agent" {
		t.Fatalf("expected supervisor-agent, got %v", events[0].AckedBy)
	}
}
