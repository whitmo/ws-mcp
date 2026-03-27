package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/whitmo/ws-mcp/src/internal/mcp"
	"github.com/whitmo/ws-mcp/src/internal/store"
	"github.com/whitmo/ws-mcp/src/internal/types"
)

// TestMCPStdio_FullProtocolFlow exercises the complete MCP stdio protocol:
// initialize → tools/list → ingest event via store → tools/call events_latest → verify event.
func TestMCPStdio_FullProtocolFlow(t *testing.T) {
	buf := store.NewRingBuffer(100)
	handler := mcp.NewHandler(buf)
	srv := mcp.NewServer(handler)

	// Build the request sequence
	requests := []string{
		`{"jsonrpc":"2.0","method":"initialize","id":1}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","method":"tools/list","id":2}`,
	}

	// Ingest a test event directly into the store (simulating HTTP ingest)
	testEvent := types.Event{
		ID:      "integ-001",
		Source:  types.SourceRalph,
		Type:    "task.start",
		Ts:      time.Now(),
		Payload: map[string]any{"message": "integration test event"},
	}
	buf.Push(testEvent)

	// Now query for it via tools/call
	requests = append(requests,
		`{"jsonrpc":"2.0","method":"tools/call","params":{"name":"events_latest","arguments":{"limit":10}},"id":3}`,
		`{"jsonrpc":"2.0","method":"tools/call","params":{"name":"events_ack","arguments":{"id":"integ-001","acked_by":"test-agent"}},"id":4}`,
		`{"jsonrpc":"2.0","method":"tools/call","params":{"name":"events_latest","arguments":{"limit":10}},"id":5}`,
	)

	input := strings.Join(requests, "\n") + "\n"
	in := strings.NewReader(input)
	var out bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := srv.ServeIO(ctx, in, &out)
	if err != nil {
		t.Fatalf("ServeIO error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	// Expect 5 responses: initialize, tools/list, events_latest, events_ack, events_latest
	// (notifications/initialized produces no response)
	if len(lines) != 5 {
		t.Fatalf("expected 5 response lines, got %d:\n%s", len(lines), out.String())
	}

	// --- 1. Verify initialize ---
	var initResp mcp.Response
	mustUnmarshal(t, lines[0], &initResp)
	if initResp.Error != nil {
		t.Fatalf("initialize error: %+v", initResp.Error)
	}
	initResult := marshalToMap(t, initResp.Result)
	if initResult["protocolVersion"] != "2024-11-05" {
		t.Fatalf("expected protocolVersion 2024-11-05, got %v", initResult["protocolVersion"])
	}

	// --- 2. Verify tools/list ---
	var listResp mcp.Response
	mustUnmarshal(t, lines[1], &listResp)
	if listResp.Error != nil {
		t.Fatalf("tools/list error: %+v", listResp.Error)
	}
	listResult := marshalToMap(t, listResp.Result)
	tools, ok := listResult["tools"].([]any)
	if !ok {
		t.Fatal("tools/list missing tools array")
	}
	if len(tools) < 4 {
		t.Fatalf("expected at least 4 tools, got %d", len(tools))
	}

	// --- 3. Verify events_latest contains our event ---
	var latestResp mcp.Response
	mustUnmarshal(t, lines[2], &latestResp)
	if latestResp.Error != nil {
		t.Fatalf("events_latest error: %+v", latestResp.Error)
	}
	events := extractEventsFromToolsCall(t, latestResp)
	found := false
	for _, e := range events {
		if e.ID == "integ-001" && e.Source == types.SourceRalph && e.Type == "task.start" {
			found = true
			if e.Acked {
				t.Fatal("event should not be acked yet")
			}
		}
	}
	if !found {
		t.Fatalf("test event integ-001 not found in events_latest response")
	}

	// --- 4. Verify events_ack succeeded ---
	var ackResp mcp.Response
	mustUnmarshal(t, lines[3], &ackResp)
	if ackResp.Error != nil {
		t.Fatalf("events_ack error: %+v", ackResp.Error)
	}

	// --- 5. Verify event is now acked ---
	var postAckResp mcp.Response
	mustUnmarshal(t, lines[4], &postAckResp)
	if postAckResp.Error != nil {
		t.Fatalf("post-ack events_latest error: %+v", postAckResp.Error)
	}
	postAckEvents := extractEventsFromToolsCall(t, postAckResp)
	for _, e := range postAckEvents {
		if e.ID == "integ-001" {
			if !e.Acked {
				t.Fatal("event integ-001 should be acked after events_ack")
			}
			if e.AckedBy != "test-agent" {
				t.Fatalf("expected acked_by test-agent, got %s", e.AckedBy)
			}
			return
		}
	}
	t.Fatal("event integ-001 not found in post-ack query")
}

// Helper: unmarshal JSON or fail
func mustUnmarshal(t *testing.T, data string, v any) {
	t.Helper()
	if err := json.Unmarshal([]byte(data), v); err != nil {
		t.Fatalf("failed to unmarshal %q: %v", data, err)
	}
}

// Helper: marshal result to map
func marshalToMap(t *testing.T, v any) map[string]any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}
	return m
}

// Helper: extract events from a tools/call response (content[0].text → []Event)
func extractEventsFromToolsCall(t *testing.T, resp mcp.Response) []types.Event {
	t.Helper()
	result := marshalToMap(t, resp.Result)
	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatal("missing content in tools/call response")
	}
	item := content[0].(map[string]any)
	text, ok := item["text"].(string)
	if !ok {
		t.Fatal("content[0].text not a string")
	}
	var events []types.Event
	if err := json.Unmarshal([]byte(text), &events); err != nil {
		t.Fatalf("failed to parse events from text: %v", err)
	}
	return events
}
