package hub

import (
	"testing"
	"time"

	"github.com/whitmo/ws-mcp/src/internal/types"
)

func TestHub_Broadcast(t *testing.T) {
	h := NewHub()
	go h.Run()
	defer h.Stop()

	// Connect a dummy client
	clientReceived := make(chan types.Event, 1)
	client := &Client{
		hub:  h,
		send: make(chan types.Event, 256),
	}
	h.register <- client

	go func() {
		for event := range client.send {
			clientReceived <- event
		}
	}()

	// Broadcast event
	testEvent := types.Event{
		ID:     "test-123",
		Source: types.SourceSystem,
		Ts:     time.Now(),
	}
	h.Broadcast(testEvent)

	// Verify receipt
	select {
	case received := <-clientReceived:
		if received.ID != "test-123" {
			t.Fatalf("expected event id test-123, got %s", received.ID)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for broadcast event")
	}
}
