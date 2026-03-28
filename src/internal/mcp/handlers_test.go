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

func TestMCPHandlers_TaskDelegateAcceptComplete(t *testing.T) {
	rb := store.NewRingBuffer(10)
	handler := NewHandler(rb)

	// Delegate
	taskID, err := handler.HandleTaskDelegate(context.Background(), "alice", "bob", "do the thing")
	if err != nil {
		t.Fatalf("delegate: %v", err)
	}
	if taskID == "" {
		t.Fatal("expected non-empty task_id")
	}

	// Check status
	task, err := handler.HandleTaskStatus(context.Background(), taskID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if task.Status != "pending" {
		t.Fatalf("expected pending, got %s", task.Status)
	}

	// Pending for bob
	pending, err := handler.HandleTaskPending(context.Background(), "bob")
	if err != nil {
		t.Fatalf("pending: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending task, got %d", len(pending))
	}

	// Accept
	if err := handler.HandleTaskAccept(context.Background(), taskID); err != nil {
		t.Fatalf("accept: %v", err)
	}

	// Pending should now be empty
	pending, _ = handler.HandleTaskPending(context.Background(), "bob")
	if len(pending) != 0 {
		t.Fatalf("expected 0 pending after accept, got %d", len(pending))
	}

	// Complete
	result := map[string]any{"output": "all done"}
	if err := handler.HandleTaskComplete(context.Background(), taskID, result); err != nil {
		t.Fatalf("complete: %v", err)
	}

	// Final status
	task, _ = handler.HandleTaskStatus(context.Background(), taskID)
	if task.Status != "completed" {
		t.Fatalf("expected completed, got %s", task.Status)
	}
	if task.Result["output"] != "all done" {
		t.Fatalf("expected result output, got %v", task.Result)
	}
}

func TestMCPHandlers_TaskDelegate_Validation(t *testing.T) {
	rb := store.NewRingBuffer(10)
	handler := NewHandler(rb)

	if _, err := handler.HandleTaskDelegate(context.Background(), "", "bob", "desc"); err == nil {
		t.Fatal("expected error for missing from_agent")
	}
	if _, err := handler.HandleTaskDelegate(context.Background(), "alice", "", "desc"); err == nil {
		t.Fatal("expected error for missing to_agent")
	}
	if _, err := handler.HandleTaskDelegate(context.Background(), "alice", "bob", ""); err == nil {
		t.Fatal("expected error for missing description")
	}
}

func TestMCPHandlers_TaskStatus_NotFound(t *testing.T) {
	rb := store.NewRingBuffer(10)
	handler := NewHandler(rb)

	_, err := handler.HandleTaskStatus(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent task")
	}
}

func TestMCPHandlers_Filter(t *testing.T) {
	rb := store.NewRingBuffer(10)
	rb.Push(types.Event{ID: "event-1", Source: types.SourceRalph, Ts: time.Now()})
	rb.Push(types.Event{ID: "event-2", Source: types.SourceSystem, Ts: time.Now()})

	handler := NewHandler(rb)

	result, err := handler.HandleFilter(context.Background(), string(types.SourceRalph), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 || result[0].ID != "event-1" {
		t.Fatalf("expected event-1, got %v", result)
	}
}
