package mcp

import (
	"context"
	"testing"
	"time"

	"github.com/whitmo/ws-mcp/src/internal/store"
	"github.com/whitmo/ws-mcp/src/internal/types"
)

func TestMCPHandlers_Latest(t *testing.T) {
	rb := store.NewRingBuffer(10)
	rb.Push(types.Event{ID: "event-1", Source: types.SourceRalph, Ts: time.Now()})
	rb.Push(types.Event{ID: "event-2", Source: types.SourceSystem, Ts: time.Now()})

	handler := NewHandler(rb)
	
	result, err := handler.HandleLatest(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 || result[0].ID != "event-2" {
		t.Fatalf("expected event-2, got %v", result)
	}
}

func TestMCPHandlers_Request(t *testing.T) {
	rb := store.NewRingBuffer(10)
	handler := NewHandler(rb)

	event := types.Event{
		ID:     "req-1",
		Source: types.SourceRalph,
		Type:   "request",
		Ts:     time.Now(),
	}

	id, err := handler.HandleRequest(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "req-1" {
		t.Fatalf("expected req-1, got %s", id)
	}

	// Verify it was stored
	found, ok := rb.FindByID("req-1")
	if !ok {
		t.Fatal("request event not found in store")
	}
	if found.Type != "request" {
		t.Fatalf("expected type request, got %s", found.Type)
	}
}

func TestMCPHandlers_Request_WrongType(t *testing.T) {
	rb := store.NewRingBuffer(10)
	handler := NewHandler(rb)

	event := types.Event{ID: "e1", Source: types.SourceRalph, Type: "task.started", Ts: time.Now()}
	_, err := handler.HandleRequest(context.Background(), event)
	if err == nil {
		t.Fatal("expected error for non-request type")
	}
}

func TestMCPHandlers_AwaitReply(t *testing.T) {
	rb := store.NewRingBuffer(10)
	handler := NewHandler(rb)

	// Store a request
	rb.Push(types.Event{ID: "req-1", Source: types.SourceRalph, Type: "request", Ts: time.Now()})

	// Simulate a reply arriving after a short delay
	go func() {
		time.Sleep(200 * time.Millisecond)
		rb.Push(types.Event{ID: "reply-1", Source: types.SourceSystem, Type: "response", InReplyTo: "req-1", Ts: time.Now()})
	}()

	reply, err := handler.HandleAwaitReply(context.Background(), "req-1", 5000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reply.ID != "reply-1" {
		t.Fatalf("expected reply-1, got %s", reply.ID)
	}
	if reply.InReplyTo != "req-1" {
		t.Fatalf("expected in_reply_to req-1, got %s", reply.InReplyTo)
	}
}

func TestMCPHandlers_AwaitReply_Timeout(t *testing.T) {
	rb := store.NewRingBuffer(10)
	handler := NewHandler(rb)

	rb.Push(types.Event{ID: "req-1", Source: types.SourceRalph, Type: "request", Ts: time.Now()})

	_, err := handler.HandleAwaitReply(context.Background(), "req-1", 200)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestMCPHandlers_AwaitReply_NotFound(t *testing.T) {
	rb := store.NewRingBuffer(10)
	handler := NewHandler(rb)

	_, err := handler.HandleAwaitReply(context.Background(), "nonexistent", 200)
	if err == nil {
		t.Fatal("expected error for nonexistent request")
	}
}

func TestMCPHandlers_Filter(t *testing.T) {
	rb := store.NewRingBuffer(10)
	rb.Push(types.Event{ID: "event-1", Source: types.SourceRalph, Ts: time.Now()})
	rb.Push(types.Event{ID: "event-2", Source: types.SourceSystem, Ts: time.Now()})

	handler := NewHandler(rb)
	
	result, err := handler.HandleFilter(context.Background(), string(types.SourceRalph))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 || result[0].ID != "event-1" {
		t.Fatalf("expected event-1, got %v", result)
	}
}
