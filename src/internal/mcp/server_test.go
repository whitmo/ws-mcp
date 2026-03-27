package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/whitmo/ws-mcp/src/internal/store"
	"github.com/whitmo/ws-mcp/src/internal/types"
)

func newTestServer() (*Server, *store.RingBuffer) {
	rb := store.NewRingBuffer(100)
	h := NewHandler(rb)
	s := NewServer(h)
	return s, rb
}

func seedEvents(rb *store.RingBuffer) {
	rb.Push(types.Event{ID: "e1", Source: types.SourceRalph, Type: "task.start", Ts: time.Now()})
	rb.Push(types.Event{ID: "e2", Source: types.SourceSystem, Type: "error", Ts: time.Now()})
	rb.Push(types.Event{ID: "e3", Source: types.SourceMultiClaude, Type: "task.complete", Ts: time.Now()})
}

func doRPC(t *testing.T, srv *Server, method string, params any) Response {
	t.Helper()
	var raw json.RawMessage
	if params != nil {
		b, _ := json.Marshal(params)
		raw = b
	}
	req := &Request{JSONRPC: "2.0", Method: method, Params: raw, ID: 1}
	resp := srv.Dispatch(context.Background(), req)
	return *resp
}

func TestDispatch_EventsLatest(t *testing.T) {
	srv, rb := newTestServer()
	seedEvents(rb)

	resp := doRPC(t, srv, "events.latest", map[string]int{"limit": 2})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	b, _ := json.Marshal(resp.Result)
	var events []types.Event
	json.Unmarshal(b, &events)
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].ID != "e3" {
		t.Fatalf("expected e3 first (descending), got %s", events[0].ID)
	}
}

func TestDispatch_EventsLatest_DefaultLimit(t *testing.T) {
	srv, rb := newTestServer()
	seedEvents(rb)

	resp := doRPC(t, srv, "events.latest", nil)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	b, _ := json.Marshal(resp.Result)
	var events []types.Event
	json.Unmarshal(b, &events)
	if len(events) != 3 {
		t.Fatalf("expected 3 events (all), got %d", len(events))
	}
}

func TestDispatch_EventsFilter(t *testing.T) {
	srv, rb := newTestServer()
	seedEvents(rb)

	resp := doRPC(t, srv, "events.filter", map[string]string{"source": "ralph"})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	b, _ := json.Marshal(resp.Result)
	var events []types.Event
	json.Unmarshal(b, &events)
	if len(events) != 1 || events[0].ID != "e1" {
		t.Fatalf("expected 1 ralph event, got %v", events)
	}
}

func TestDispatch_EventsAck(t *testing.T) {
	srv, rb := newTestServer()
	seedEvents(rb)

	resp := doRPC(t, srv, "events.ack", map[string]string{"id": "e1", "acked_by": "agent-x"})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	// Verify ack took effect
	events := rb.Latest(100)
	for _, e := range events {
		if e.ID == "e1" {
			if !e.Acked {
				t.Fatal("event e1 should be acked")
			}
			if e.AckedBy != "agent-x" {
				t.Fatalf("expected acked_by agent-x, got %s", e.AckedBy)
			}
			return
		}
	}
	t.Fatal("event e1 not found")
}

func TestDispatch_EventsAck_MissingID(t *testing.T) {
	srv, _ := newTestServer()

	resp := doRPC(t, srv, "events.ack", map[string]string{"acked_by": "x"})
	if resp.Error == nil {
		t.Fatal("expected error for missing id")
	}
	if resp.Error.Code != ErrCodeBadParams {
		t.Fatalf("expected bad params error, got %d", resp.Error.Code)
	}
}

func TestDispatch_EventsAck_NotFound(t *testing.T) {
	srv, _ := newTestServer()

	resp := doRPC(t, srv, "events.ack", map[string]string{"id": "nonexistent"})
	if resp.Error == nil {
		t.Fatal("expected error for nonexistent event")
	}
}

func TestDispatch_ReportSummary(t *testing.T) {
	srv, rb := newTestServer()
	seedEvents(rb)

	resp := doRPC(t, srv, "report.summary", map[string]int{"window": 60})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	b, _ := json.Marshal(resp.Result)
	var summary SummaryResult
	json.Unmarshal(b, &summary)
	if summary.TotalEvents != 3 {
		t.Fatalf("expected 3 total events, got %d", summary.TotalEvents)
	}
	if summary.BySource["ralph"] != 1 {
		t.Fatalf("expected 1 ralph event, got %d", summary.BySource["ralph"])
	}
	if summary.ByType["error"] != 1 {
		t.Fatalf("expected 1 error event, got %d", summary.ByType["error"])
	}
	if len(summary.Alerts) == 0 || summary.Alerts[0] != "errors detected in window" {
		t.Fatalf("expected error alert, got %v", summary.Alerts)
	}
}

func TestMCP_Initialize(t *testing.T) {
	srv, _ := newTestServer()

	resp := doRPC(t, srv, "initialize", nil)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	b, _ := json.Marshal(resp.Result)
	var result map[string]any
	json.Unmarshal(b, &result)

	if result["protocolVersion"] != "2024-11-05" {
		t.Fatalf("expected protocolVersion 2024-11-05, got %v", result["protocolVersion"])
	}
	serverInfo, ok := result["serverInfo"].(map[string]any)
	if !ok {
		t.Fatal("missing serverInfo")
	}
	if serverInfo["name"] != "ws-mcp" {
		t.Fatalf("expected server name ws-mcp, got %v", serverInfo["name"])
	}
	if serverInfo["version"] != "0.1.0" {
		t.Fatalf("expected version 0.1.0, got %v", serverInfo["version"])
	}
	caps, ok := result["capabilities"].(map[string]any)
	if !ok {
		t.Fatal("missing capabilities")
	}
	if _, ok := caps["tools"]; !ok {
		t.Fatal("missing tools capability")
	}
}

func TestMCP_NotificationsInitialized(t *testing.T) {
	srv, _ := newTestServer()

	req := &Request{JSONRPC: "2.0", Method: "notifications/initialized", ID: nil}
	resp := srv.Dispatch(context.Background(), req)
	if resp != nil {
		t.Fatalf("expected nil response for notification, got %+v", resp)
	}
}

func TestMCP_ToolsList(t *testing.T) {
	srv, _ := newTestServer()

	resp := doRPC(t, srv, "tools/list", nil)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	b, _ := json.Marshal(resp.Result)
	var result map[string]any
	json.Unmarshal(b, &result)

	tools, ok := result["tools"].([]any)
	if !ok {
		t.Fatal("missing tools array")
	}
	if len(tools) != 6 {
		t.Fatalf("expected 6 tools, got %d", len(tools))
	}

	// Verify each tool has required fields
	expectedNames := map[string]bool{
		"events_latest": false, "events_filter": false,
		"events_ack": false, "report_summary": false,
		"events_request": false, "events_await_reply": false,
	}
	for _, raw := range tools {
		tool := raw.(map[string]any)
		name := tool["name"].(string)
		if _, ok := expectedNames[name]; !ok {
			t.Fatalf("unexpected tool: %s", name)
		}
		expectedNames[name] = true
		if tool["description"] == nil || tool["description"] == "" {
			t.Fatalf("tool %s missing description", name)
		}
		if tool["inputSchema"] == nil {
			t.Fatalf("tool %s missing inputSchema", name)
		}
	}
	for name, found := range expectedNames {
		if !found {
			t.Fatalf("tool %s not found in list", name)
		}
	}
}

func TestDispatch_EventsRequest(t *testing.T) {
	srv, _ := newTestServer()

	resp := doRPC(t, srv, "events.request", map[string]any{
		"id":     "req-1",
		"source": "ralph",
	})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	b, _ := json.Marshal(resp.Result)
	var result map[string]string
	json.Unmarshal(b, &result)
	if result["request_id"] != "req-1" {
		t.Fatalf("expected request_id req-1, got %s", result["request_id"])
	}
}

func TestDispatch_EventsRequest_MissingID(t *testing.T) {
	srv, _ := newTestServer()

	resp := doRPC(t, srv, "events.request", map[string]any{
		"source": "ralph",
	})
	if resp.Error == nil {
		t.Fatal("expected error for missing id")
	}
}

func TestDispatch_EventsAwaitReply(t *testing.T) {
	srv, rb := newTestServer()

	// Store request event
	rb.Push(types.Event{ID: "req-1", Source: types.SourceRalph, Type: "request", Ts: time.Now()})

	// Store reply event
	rb.Push(types.Event{ID: "reply-1", Source: types.SourceSystem, Type: "response", InReplyTo: "req-1", Ts: time.Now()})

	resp := doRPC(t, srv, "events.await_reply", map[string]any{
		"request_id": "req-1",
		"timeout_ms": 1000,
	})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	b, _ := json.Marshal(resp.Result)
	var event types.Event
	json.Unmarshal(b, &event)
	if event.ID != "reply-1" {
		t.Fatalf("expected reply-1, got %s", event.ID)
	}
}

func TestDispatch_EventsAwaitReply_MissingRequestID(t *testing.T) {
	srv, _ := newTestServer()

	resp := doRPC(t, srv, "events.await_reply", map[string]any{
		"timeout_ms": 100,
	})
	if resp.Error == nil {
		t.Fatal("expected error for missing request_id")
	}
}

func TestMCP_ToolsCall_EventsLatest(t *testing.T) {
	srv, rb := newTestServer()
	seedEvents(rb)

	resp := doRPC(t, srv, "tools/call", map[string]any{
		"name":      "events_latest",
		"arguments": map[string]any{"limit": 2},
	})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	b, _ := json.Marshal(resp.Result)
	var result map[string]any
	json.Unmarshal(b, &result)

	content, ok := result["content"].([]any)
	if !ok || len(content) != 1 {
		t.Fatalf("expected content array with 1 element, got %v", result["content"])
	}
	item := content[0].(map[string]any)
	if item["type"] != "text" {
		t.Fatalf("expected type text, got %v", item["type"])
	}

	// The text should be valid JSON containing events
	var events []types.Event
	if err := json.Unmarshal([]byte(item["text"].(string)), &events); err != nil {
		t.Fatalf("failed to parse text as events: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

func TestMCP_ToolsCall_UnknownTool(t *testing.T) {
	srv, _ := newTestServer()

	resp := doRPC(t, srv, "tools/call", map[string]any{
		"name":      "nonexistent",
		"arguments": map[string]any{},
	})
	if resp.Error == nil {
		t.Fatal("expected error for unknown tool")
	}
	if resp.Error.Code != ErrCodeNoMethod {
		t.Fatalf("expected method not found code, got %d", resp.Error.Code)
	}
}

func TestMCP_ToolsCall_EventsAck(t *testing.T) {
	srv, rb := newTestServer()
	seedEvents(rb)

	resp := doRPC(t, srv, "tools/call", map[string]any{
		"name":      "events_ack",
		"arguments": map[string]any{"id": "e1", "acked_by": "mcp-client"},
	})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	// Verify ack took effect
	events := rb.Latest(100)
	for _, e := range events {
		if e.ID == "e1" {
			if !e.Acked || e.AckedBy != "mcp-client" {
				t.Fatalf("expected e1 acked by mcp-client, got acked=%v by=%s", e.Acked, e.AckedBy)
			}
			return
		}
	}
	t.Fatal("event e1 not found")
}

func TestMCP_Stdio_FullHandshake(t *testing.T) {
	srv, rb := newTestServer()
	seedEvents(rb)

	// Simulate a full MCP handshake: initialize -> notifications/initialized -> tools/list -> tools/call
	input := `{"jsonrpc":"2.0","method":"initialize","id":1}
{"jsonrpc":"2.0","method":"notifications/initialized"}
{"jsonrpc":"2.0","method":"tools/list","id":2}
{"jsonrpc":"2.0","method":"tools/call","params":{"name":"events_latest","arguments":{"limit":1}},"id":3}
`
	in := strings.NewReader(input)
	var out bytes.Buffer

	err := srv.ServeIO(context.Background(), in, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	// Should get 3 responses (notifications/initialized produces no output)
	if len(lines) != 3 {
		t.Fatalf("expected 3 response lines, got %d: %s", len(lines), out.String())
	}

	// Verify initialize response
	var resp1 Response
	json.Unmarshal([]byte(lines[0]), &resp1)
	if resp1.Error != nil {
		t.Fatalf("initialize error: %v", resp1.Error)
	}

	// Verify tools/list response
	var resp2 Response
	json.Unmarshal([]byte(lines[1]), &resp2)
	if resp2.Error != nil {
		t.Fatalf("tools/list error: %v", resp2.Error)
	}

	// Verify tools/call response
	var resp3 Response
	json.Unmarshal([]byte(lines[2]), &resp3)
	if resp3.Error != nil {
		t.Fatalf("tools/call error: %v", resp3.Error)
	}
}

func TestMCP_HTTP_NotificationNoContent(t *testing.T) {
	srv, _ := newTestServer()

	body := `{"jsonrpc":"2.0","method":"notifications/initialized"}`
	req := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for notification, got %d", w.Code)
	}
}

func TestDispatch_MethodNotFound(t *testing.T) {
	srv, _ := newTestServer()

	resp := doRPC(t, srv, "nonexistent.method", nil)
	if resp.Error == nil {
		t.Fatal("expected error for unknown method")
	}
	if resp.Error.Code != ErrCodeNoMethod {
		t.Fatalf("expected method not found code, got %d", resp.Error.Code)
	}
}

func TestHTTP_RPC(t *testing.T) {
	srv, rb := newTestServer()
	seedEvents(rb)

	body := `{"jsonrpc":"2.0","method":"events.latest","params":{"limit":1},"id":42}`
	req := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp Response
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	if resp.ID != float64(42) {
		t.Fatalf("expected id 42, got %v", resp.ID)
	}
}

func TestHTTP_InvalidJSON(t *testing.T) {
	srv, _ := newTestServer()

	req := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader("not json"))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var resp Response
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Error == nil || resp.Error.Code != ErrCodeParse {
		t.Fatalf("expected parse error, got %v", resp.Error)
	}
}

func TestHTTP_MethodNotAllowed(t *testing.T) {
	srv, _ := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/rpc", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestStdio_Transport(t *testing.T) {
	srv, rb := newTestServer()
	seedEvents(rb)

	input := `{"jsonrpc":"2.0","method":"events.latest","params":{"limit":1},"id":1}
{"jsonrpc":"2.0","method":"report.summary","params":{"window":60},"id":2}
`
	in := strings.NewReader(input)
	var out bytes.Buffer

	err := srv.ServeIO(context.Background(), in, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 response lines, got %d: %s", len(lines), out.String())
	}

	var resp1 Response
	json.Unmarshal([]byte(lines[0]), &resp1)
	if resp1.Error != nil {
		t.Fatalf("resp1 error: %v", resp1.Error)
	}

	var resp2 Response
	json.Unmarshal([]byte(lines[1]), &resp2)
	if resp2.Error != nil {
		t.Fatalf("resp2 error: %v", resp2.Error)
	}
}
