package integration

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/whitmo/ws-mcp/src/internal/store"
	"github.com/whitmo/ws-mcp/src/internal/types"
	"github.com/whitmo/ws-mcp/src/pkg/api"
)

func TestIngest_KnownEventType_NoWarning(t *testing.T) {
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	defer log.SetOutput(nil)

	rb := store.NewRingBuffer(10)
	router := api.NewRouter(rb)
	server := httptest.NewServer(router.SetupRoutes())
	defer server.Close()

	event := types.Event{
		ID:     "test-1",
		Source: types.SourceMultiClaude,
		Type:   "task.started",
		Ts:     time.Now(),
		Payload: map[string]any{
			"task_id": "abc",
			"agent":   "worker-1",
		},
	}

	body, _ := json.Marshal(event)
	resp, err := http.Post(server.URL+"/event", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}

	if strings.Contains(logBuf.String(), "WARN: unknown event type") {
		t.Error("should not warn for known event type")
	}
}

func TestIngest_UnknownEventType_WarnsButAccepts(t *testing.T) {
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	defer log.SetOutput(nil)

	rb := store.NewRingBuffer(10)
	router := api.NewRouter(rb)
	server := httptest.NewServer(router.SetupRoutes())
	defer server.Close()

	event := types.Event{
		ID:     "test-2",
		Source: types.SourceSystem,
		Type:   "custom.unknown",
		Ts:     time.Now(),
		Payload: map[string]any{
			"detail": "something custom",
		},
	}

	body, _ := json.Marshal(event)
	resp, err := http.Post(server.URL+"/event", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	// Should still accept the event
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 for unknown event type, got %d", resp.StatusCode)
	}

	// Should log a warning
	if !strings.Contains(logBuf.String(), "WARN: unknown event type") {
		t.Error("expected warning log for unknown event type")
	}
	if !strings.Contains(logBuf.String(), "custom.unknown") {
		t.Error("expected warning to contain the event type")
	}
}

func TestAllStandardEventTypes_Accepted(t *testing.T) {
	rb := store.NewRingBuffer(100)
	router := api.NewRouter(rb)
	server := httptest.NewServer(router.SetupRoutes())
	defer server.Close()

	standardTypes := []string{
		"task.started", "task.completed", "task.failed",
		"commit.pushed",
		"pr.opened", "pr.merged", "pr.reviewed",
		"review.requested", "review.completed",
		"agent.started", "agent.stopped",
		"system.healthcheck", "system.error",
	}

	for _, et := range standardTypes {
		event := types.Event{
			ID:      "test-" + et,
			Source:  types.SourceMultiClaude,
			Type:    et,
			Ts:      time.Now(),
			Payload: map[string]any{},
		}

		body, _ := json.Marshal(event)
		resp, err := http.Post(server.URL+"/event", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("event type %s: %v", et, err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusAccepted {
			t.Errorf("event type %s: expected 202, got %d", et, resp.StatusCode)
		}
	}
}
