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

func TestRingBuffer_FindByID(t *testing.T) {
	rb := NewRingBuffer(10)
	rb.Push(types.Event{ID: "abc", Source: types.SourceSystem})
	rb.Push(types.Event{ID: "def", Source: types.SourceRalph})

	e, found := rb.FindByID("abc")
	if !found {
		t.Fatal("expected to find event abc")
	}
	if e.ID != "abc" {
		t.Fatalf("expected abc, got %s", e.ID)
	}

	_, found = rb.FindByID("nonexistent")
	if found {
		t.Fatal("expected not to find nonexistent event")
	}
}

func TestRingBuffer_FindByInReplyTo(t *testing.T) {
	rb := NewRingBuffer(10)
	rb.Push(types.Event{ID: "req-1", Source: types.SourceSystem, Type: "request"})
	rb.Push(types.Event{ID: "reply-1", Source: types.SourceRalph, InReplyTo: "req-1"})

	reply, found := rb.FindByInReplyTo("req-1")
	if !found {
		t.Fatal("expected to find reply")
	}
	if reply.ID != "reply-1" {
		t.Fatalf("expected reply-1, got %s", reply.ID)
	}

	_, found = rb.FindByInReplyTo("nonexistent")
	if found {
		t.Fatal("expected not to find reply for nonexistent request")
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
