package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/whitmo/ws-mcp/src/internal/types"
)

func TestMatchEvent(t *testing.T) {
	ev := types.Event{
		ID:     "1",
		Source: types.SourceRalph,
		Type:   "task.started",
		Ts:     time.Now(),
	}

	tests := []struct {
		name       string
		source     string
		typeFilter string
		want       bool
	}{
		{"no filters", "", "", true},
		{"source match", "ralph", "", true},
		{"source mismatch", "multiclaude", "", false},
		{"type exact match", "", "task.started", true},
		{"type glob match", "", "task.*", true},
		{"type glob mismatch", "", "error.*", false},
		{"both match", "ralph", "task.*", true},
		{"source match type mismatch", "ralph", "error", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchEvent(ev, tt.source, tt.typeFilter)
			if got != tt.want {
				t.Errorf("matchEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGlobMatch(t *testing.T) {
	tests := []struct {
		pattern string
		value   string
		want    bool
	}{
		{"task.*", "task.started", true},
		{"task.*", "task.completed", true},
		{"task.*", "error", false},
		{"error", "error", true},
		{"*", "anything", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.value, func(t *testing.T) {
			if got := globMatch(tt.pattern, tt.value); got != tt.want {
				t.Errorf("globMatch(%q, %q) = %v, want %v", tt.pattern, tt.value, got, tt.want)
			}
		})
	}
}

func TestObserverReceivesEvents(t *testing.T) {
	// Set up a test WebSocket server that sends events
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

	events := []types.Event{
		{ID: "e1", Source: types.SourceRalph, Type: "task.started", Ts: time.Now(), Payload: map[string]any{"task": "build"}},
		{ID: "e2", Source: types.SourceSystem, Type: "error", Ts: time.Now(), Payload: map[string]any{"msg": "fail"}},
		{ID: "e3", Source: types.SourceRalph, Type: "task.completed", Ts: time.Now(), Payload: map[string]any{"task": "build"}},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer conn.Close()

		for _, ev := range events {
			if err := conn.WriteJSON(ev); err != nil {
				return
			}
		}
		// Give client time to read, then close
		time.Sleep(100 * time.Millisecond)
		conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/"

	// Connect as observer, collect received events
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	var received []types.Event
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		var ev types.Event
		if err := json.Unmarshal(msg, &ev); err != nil {
			t.Errorf("unmarshal: %v", err)
			continue
		}
		received = append(received, ev)
	}

	if len(received) != 3 {
		t.Fatalf("expected 3 events, got %d", len(received))
	}

	// Verify filtering works on received events
	var filtered []types.Event
	for _, ev := range received {
		if matchEvent(ev, "ralph", "task.*") {
			filtered = append(filtered, ev)
		}
	}
	if len(filtered) != 2 {
		t.Errorf("expected 2 filtered events (ralph+task.*), got %d", len(filtered))
	}
}
