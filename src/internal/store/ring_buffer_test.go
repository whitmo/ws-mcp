package store

import (
	"testing"
	"time"

	"github.com/whitmo/ws-mcp/src/internal/types"
)

func TestRingBuffer_PushAndLatest(t *testing.T) {
	rb := NewRingBuffer(3) // small capacity for test

	// Push 4 events
	for i := 1; i <= 4; i++ {
		rb.Push(types.Event{
			ID:     "id",
			Source: types.SourceSystem,
			Ts:     time.Now(),
		})
	}

	// Should only hold the last 3 events (evicted the first one)
	events := rb.Latest(10)
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
}

func TestRingBuffer_Ack(t *testing.T) {
	rb := NewRingBuffer(10)
	event := types.Event{
		ID:     "123-abc",
		Source: types.SourceRalph,
	}
	rb.Push(event)

	err := rb.Ack("123-abc", "supervisor")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	latest := rb.Latest(1)
	if !latest[0].Acked {
		t.Fatalf("expected event to be acked")
	}
	if latest[0].AckedBy != "supervisor" {
		t.Fatalf("expected acked_by to be 'supervisor', got '%s'", latest[0].AckedBy)
	}
}
