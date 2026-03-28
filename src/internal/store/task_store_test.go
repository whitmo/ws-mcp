package store

import (
	"testing"
	"time"

	"github.com/whitmo/ws-mcp/src/internal/types"
)

func TestTaskStore_PutGet(t *testing.T) {
	ts := NewTaskStore()
	task := &types.TaskDelegation{
		ID:        "t1",
		FromAgent: "alice",
		ToAgent:   "bob",
		Status:    types.TaskPending,
		CreatedAt: time.Now(),
	}
	ts.Put(task)

	got, ok := ts.Get("t1")
	if !ok {
		t.Fatal("expected to find task t1")
	}
	if got.FromAgent != "alice" {
		t.Fatalf("expected from_agent alice, got %s", got.FromAgent)
	}
}

func TestTaskStore_AcceptComplete(t *testing.T) {
	ts := NewTaskStore()
	ts.Put(&types.TaskDelegation{
		ID: "t1", FromAgent: "a", ToAgent: "b",
		Status: types.TaskPending, CreatedAt: time.Now(),
	})

	if err := ts.Accept("t1"); err != nil {
		t.Fatalf("accept: %v", err)
	}
	got, _ := ts.Get("t1")
	if got.Status != types.TaskAccepted {
		t.Fatalf("expected accepted, got %s", got.Status)
	}
	if got.AcceptedAt == nil {
		t.Fatal("expected accepted_at to be set")
	}

	result := map[string]any{"output": "done"}
	if err := ts.Complete("t1", result); err != nil {
		t.Fatalf("complete: %v", err)
	}
	got, _ = ts.Get("t1")
	if got.Status != types.TaskCompleted {
		t.Fatalf("expected completed, got %s", got.Status)
	}
	if got.Result["output"] != "done" {
		t.Fatalf("expected result output=done, got %v", got.Result)
	}
}

func TestTaskStore_AcceptNotPending(t *testing.T) {
	ts := NewTaskStore()
	ts.Put(&types.TaskDelegation{
		ID: "t1", Status: types.TaskAccepted, CreatedAt: time.Now(),
	})
	if err := ts.Accept("t1"); err == nil {
		t.Fatal("expected error accepting non-pending task")
	}
}

func TestTaskStore_CompleteNotAccepted(t *testing.T) {
	ts := NewTaskStore()
	ts.Put(&types.TaskDelegation{
		ID: "t1", Status: types.TaskPending, CreatedAt: time.Now(),
	})
	if err := ts.Complete("t1", nil); err == nil {
		t.Fatal("expected error completing non-accepted task")
	}
}

func TestTaskStore_PendingFor(t *testing.T) {
	ts := NewTaskStore()
	ts.Put(&types.TaskDelegation{ID: "t1", ToAgent: "bob", Status: types.TaskPending, CreatedAt: time.Now()})
	ts.Put(&types.TaskDelegation{ID: "t2", ToAgent: "bob", Status: types.TaskAccepted, CreatedAt: time.Now()})
	ts.Put(&types.TaskDelegation{ID: "t3", ToAgent: "alice", Status: types.TaskPending, CreatedAt: time.Now()})

	pending := ts.PendingFor("bob")
	if len(pending) != 1 || pending[0].ID != "t1" {
		t.Fatalf("expected 1 pending task for bob (t1), got %v", pending)
	}
}

func TestTaskStore_NotFound(t *testing.T) {
	ts := NewTaskStore()
	if _, ok := ts.Get("nope"); ok {
		t.Fatal("expected not found")
	}
	if err := ts.Accept("nope"); err == nil {
		t.Fatal("expected error")
	}
	if err := ts.Complete("nope", nil); err == nil {
		t.Fatal("expected error")
	}
}
