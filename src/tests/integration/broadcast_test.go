package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/whitmo/ws-mcp/src/internal/hub"
	"github.com/whitmo/ws-mcp/src/internal/store"
	"github.com/whitmo/ws-mcp/src/pkg/api"
	"github.com/whitmo/ws-mcp/src/internal/types"
)

func TestIngestToBroadcast(t *testing.T) {
	rb := store.NewRingBuffer(10)
	h := hub.NewHub()
	go h.Run()
	defer h.Stop()

	router := api.NewRouter(rb)
	router.SetHub(h)

	server := httptest.NewServer(router.SetupRoutes())
	defer server.Close()

	// 1. Post Event
	event := types.Event{
		ID:     "int-1",
		Source: types.SourceRalph,
		Type:   "test_started",
		Ts:     time.Now(),
		Payload: map[string]any{"foo": "bar"},
	}
	body, _ := json.Marshal(event)
	
	resp, err := http.Post(server.URL+"/event", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to post event: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %v", resp.StatusCode)
	}

	// 2. Verify stored
	latest := rb.Latest(1)
	if len(latest) == 0 || latest[0].ID != "int-1" {
		t.Fatalf("event not found in ring buffer")
	}

	// TODO: Connect WS client and verify broadcast over network
}
