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
