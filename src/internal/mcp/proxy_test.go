package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/whitmo/ws-mcp/src/internal/store"
	"github.com/whitmo/ws-mcp/src/internal/types"
)

func newTestHub(t *testing.T) (*httptest.Server, *Server) {
	t.Helper()
	rb := store.NewRingBuffer(100)
	rb.Push(types.Event{
		ID:     "hub-event-1",
		Source: types.SourceRalph,
		Type:   "task.started",
		Ts:     time.Now(),
	})
	handler := NewHandler(rb)
	server := NewServer(handler)
	ts := httptest.NewServer(server)
	return ts, server
}

func TestProxyClient_Ping(t *testing.T) {
	ts, _ := newTestHub(t)
	defer ts.Close()

	proxy := NewProxyClient(ts.URL)
	if !proxy.Ping(2 * time.Second) {
		t.Fatal("expected ping to succeed against test hub")
	}

	deadProxy := NewProxyClient("http://127.0.0.1:1")
	if deadProxy.Ping(200 * time.Millisecond) {
		t.Fatal("expected ping to fail against dead address")
	}
}

func TestProxyClient_Forward(t *testing.T) {
	ts, _ := newTestHub(t)
	defer ts.Close()

	proxy := NewProxyClient(ts.URL)
	req := `{"jsonrpc":"2.0","method":"events.latest","params":{"limit":5},"id":1}`
	resp, err := proxy.Forward(context.Background(), []byte(req))
	if err != nil {
		t.Fatalf("forward failed: %v", err)
	}

	var rpcResp Response
	if err := json.Unmarshal(resp, &rpcResp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if rpcResp.Error != nil {
		t.Fatalf("unexpected error: %+v", rpcResp.Error)
	}

	events, ok := rpcResp.Result.([]any)
	if !ok {
		t.Fatalf("expected array result, got %T", rpcResp.Result)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event from hub, got %d", len(events))
	}
}

func TestServeSpoke_EndToEnd(t *testing.T) {
	ts, _ := newTestHub(t)
	defer ts.Close()

	proxy := NewProxyClient(ts.URL)

	// Pipe requests through ServeSpoke
	requests := strings.Join([]string{
		`{"jsonrpc":"2.0","method":"initialize","params":{},"id":1}`,
		`{"jsonrpc":"2.0","method":"events.latest","params":{"limit":5},"id":2}`,
	}, "\n") + "\n"

	in := strings.NewReader(requests)
	var out bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := ServeSpoke(ctx, proxy, in, &out)
	if err != nil {
		t.Fatalf("ServeSpoke error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 response lines, got %d: %s", len(lines), out.String())
	}

	// Verify initialize response
	var initResp Response
	if err := json.Unmarshal([]byte(lines[0]), &initResp); err != nil {
		t.Fatalf("unmarshal init response: %v", err)
	}
	if initResp.Error != nil {
		t.Fatalf("init error: %+v", initResp.Error)
	}

	// Verify events.latest response
	var latestResp Response
	if err := json.Unmarshal([]byte(lines[1]), &latestResp); err != nil {
		t.Fatalf("unmarshal latest response: %v", err)
	}
	if latestResp.Error != nil {
		t.Fatalf("latest error: %+v", latestResp.Error)
	}
}

func TestServeSpoke_Notification(t *testing.T) {
	ts, _ := newTestHub(t)
	defer ts.Close()

	proxy := NewProxyClient(ts.URL)

	// Notification has no ID — should produce no output
	requests := `{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n"
	in := strings.NewReader(requests)
	var out bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ServeSpoke(ctx, proxy, in, &out)

	if out.Len() != 0 {
		t.Fatalf("expected no output for notification, got: %s", out.String())
	}
}
