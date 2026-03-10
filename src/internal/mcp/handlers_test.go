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
