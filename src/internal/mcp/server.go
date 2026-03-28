package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/whitmo/ws-mcp/src/internal/types"
)

// JSON-RPC 2.0 types

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      any             `json:"id"`
}

type Response struct {
	JSONRPC string `json:"jsonrpc"`
	Result  any    `json:"result,omitempty"`
	Error   *Error `json:"error,omitempty"`
	ID      any    `json:"id"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

const (
	ErrCodeParse      = -32700
	ErrCodeInvalidReq = -32600
	ErrCodeNoMethod   = -32601
	ErrCodeBadParams  = -32602
	ErrCodeInternal   = -32603
)

// ToolDef describes an MCP tool exposed via tools/list.
type ToolDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

// toolDefs returns the MCP tool definitions for all 4 tools.
func toolDefs() []ToolDef {
	return []ToolDef{
		{
			Name:        "events_latest",
			Description: "Get the most recent events from the event buffer",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum number of events to return (1-100, default 10)",
					},
				},
			},
		},
		{
			Name:        "events_filter",
			Description: "Filter events by source, repo, and/or exclude by type",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"source": map[string]any{
						"type":        "string",
						"description": "Event source to filter by (ralph, multiclaude, system)",
					},
					"exclude_type": map[string]any{
						"type":        "string",
						"description": "Event type to exclude (e.g. agent.activity to filter noise)",
					},
					"repo": map[string]any{
						"type":        "string",
						"description": "Filter by repo name (e.g. ws-mcp, enriched-alert)",
					},
				},
			},
		},
		{
			Name:        "events_ack",
			Description: "Acknowledge an event by ID",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{
						"type":        "string",
						"description": "Event ID to acknowledge",
					},
					"acked_by": map[string]any{
						"type":        "string",
						"description": "Identifier of the acknowledging agent",
					},
				},
				"required": []string{"id"},
			},
		},
		{
			Name:        "report_summary",
			Description: "Get a summary report of events within a time window",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"window": map[string]any{
						"type":        "integer",
						"description": "Time window in minutes (default 60)",
					},
				},
			},
		},
		{
			Name:        "events_request",
			Description: "Send a request event and get its ID for awaiting a reply",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{
						"type":        "string",
						"description": "Unique event ID",
					},
					"source": map[string]any{
						"type":        "string",
						"description": "Event source (e.g. ralph, multiclaude, system)",
					},
					"payload": map[string]any{
						"type":        "object",
						"description": "Event payload",
					},
					"reply_to": map[string]any{
						"type":        "string",
						"description": "Optional: ID of the event this request is directed to",
					},
				},
				"required": []string{"id", "source"},
			},
		},
		{
			Name:        "events_await_reply",
			Description: "Poll for a reply to a previously sent request event",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"request_id": map[string]any{
						"type":        "string",
						"description": "ID of the request event to await a reply for",
					},
					"timeout_ms": map[string]any{
						"type":        "integer",
						"description": "Timeout in milliseconds (default 30000)",
					},
				},
				"required": []string{"request_id"},
			},
		},
		{
			Name:        "events_trace",
			Description: "Get all events in a trace — follow a causal chain across agents",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"trace_id": map[string]any{
						"type":        "string",
						"description": "Trace ID to look up",
					},
				},
				"required": []string{"trace_id"},
			},
		},
	}
}

// Server dispatches JSON-RPC 2.0 calls to MCP handlers.
type Server struct {
	handler *Handler
}

func NewServer(h *Handler) *Server {
	return &Server{handler: h}
}

// Dispatch routes a JSON-RPC request to the appropriate handler.
// Supports both MCP protocol methods and direct tool methods.
func (s *Server) Dispatch(ctx context.Context, req *Request) *Response {
	switch req.Method {
	// MCP protocol methods
	case "initialize":
		return s.handleInitialize(req)
	case "notifications/initialized":
		return nil // notification, no response
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(ctx, req)

	// Direct tool methods (existing)
	case "events.latest":
		return s.handleEventsLatest(ctx, req)
	case "events.filter":
		return s.handleEventsFilter(ctx, req)
	case "events.ack":
		return s.handleEventsAck(ctx, req)
	case "report.summary":
		return s.handleReportSummary(ctx, req)
	case "events.request":
		return s.handleEventsRequest(ctx, req)
	case "events.await_reply":
		return s.handleEventsAwaitReply(ctx, req)
	case "events.trace":
		return s.handleEventsTrace(ctx, req)
	default:
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &Error{Code: ErrCodeNoMethod, Message: fmt.Sprintf("method not found: %s", req.Method)},
		}
	}
}

func (s *Server) handleInitialize(req *Request) *Response {
	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "ws-mcp", "version": "0.1.0"},
		},
	}
}

func (s *Server) handleToolsList(req *Request) *Response {
	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  map[string]any{"tools": toolDefs()},
	}
}

func (s *Server) handleToolsCall(ctx context.Context, req *Request) *Response {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if req.Params != nil {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return &Response{JSONRPC: "2.0", ID: req.ID, Error: &Error{Code: ErrCodeBadParams, Message: "invalid params"}}
		}
	}

	// Build an inner request with the arguments as params, dispatching by tool name
	inner := &Request{JSONRPC: "2.0", Params: params.Arguments, ID: req.ID}

	var innerResp *Response
	switch params.Name {
	case "events_latest":
		innerResp = s.handleEventsLatest(ctx, inner)
	case "events_filter":
		innerResp = s.handleEventsFilter(ctx, inner)
	case "events_ack":
		innerResp = s.handleEventsAck(ctx, inner)
	case "report_summary":
		innerResp = s.handleReportSummary(ctx, inner)
	case "events_request":
		innerResp = s.handleEventsRequest(ctx, inner)
	case "events_await_reply":
		innerResp = s.handleEventsAwaitReply(ctx, inner)
	case "events_trace":
		innerResp = s.handleEventsTrace(ctx, inner)
	default:
		return &Response{JSONRPC: "2.0", ID: req.ID, Error: &Error{Code: ErrCodeNoMethod, Message: fmt.Sprintf("unknown tool: %s", params.Name)}}
	}

	// If the inner handler returned an error, propagate it
	if innerResp.Error != nil {
		return innerResp
	}

	// Wrap result in MCP tools/call content format
	resultJSON, err := json.Marshal(innerResp.Result)
	if err != nil {
		return &Response{JSONRPC: "2.0", ID: req.ID, Error: &Error{Code: ErrCodeInternal, Message: "failed to marshal result"}}
	}

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]any{
			"content": []map[string]string{
				{"type": "text", "text": string(resultJSON)},
			},
		},
	}
}

func (s *Server) handleEventsLatest(ctx context.Context, req *Request) *Response {
	var params struct {
		Limit int `json:"limit"`
	}
	if req.Params != nil {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return &Response{JSONRPC: "2.0", ID: req.ID, Error: &Error{Code: ErrCodeBadParams, Message: "invalid params"}}
		}
	}

	events, err := s.handler.HandleLatest(ctx, params.Limit)
	if err != nil {
		return &Response{JSONRPC: "2.0", ID: req.ID, Error: &Error{Code: ErrCodeInternal, Message: err.Error()}}
	}
	return &Response{JSONRPC: "2.0", ID: req.ID, Result: events}
}

func (s *Server) handleEventsFilter(ctx context.Context, req *Request) *Response {
	var params struct {
		Source      string `json:"source"`
		ExcludeType string `json:"exclude_type"`
		Repo        string `json:"repo"`
	}
	if req.Params != nil {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return &Response{JSONRPC: "2.0", ID: req.ID, Error: &Error{Code: ErrCodeBadParams, Message: "invalid params"}}
		}
	}

	events, err := s.handler.HandleFilter(ctx, params.Source, params.ExcludeType, params.Repo)
	if err != nil {
		return &Response{JSONRPC: "2.0", ID: req.ID, Error: &Error{Code: ErrCodeInternal, Message: err.Error()}}
	}
	return &Response{JSONRPC: "2.0", ID: req.ID, Result: events}
}

func (s *Server) handleEventsAck(ctx context.Context, req *Request) *Response {
	var params struct {
		ID      string `json:"id"`
		AckedBy string `json:"acked_by"`
	}
	if req.Params != nil {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return &Response{JSONRPC: "2.0", ID: req.ID, Error: &Error{Code: ErrCodeBadParams, Message: "invalid params"}}
		}
	}
	if params.ID == "" {
		return &Response{JSONRPC: "2.0", ID: req.ID, Error: &Error{Code: ErrCodeBadParams, Message: "id is required"}}
	}

	err := s.handler.HandleAck(ctx, params.ID, params.AckedBy)
	if err != nil {
		return &Response{JSONRPC: "2.0", ID: req.ID, Error: &Error{Code: ErrCodeInternal, Message: err.Error()}}
	}
	return &Response{JSONRPC: "2.0", ID: req.ID, Result: map[string]string{"status": "acked"}}
}

func (s *Server) handleReportSummary(ctx context.Context, req *Request) *Response {
	var params struct {
		Window int `json:"window"`
	}
	if req.Params != nil {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return &Response{JSONRPC: "2.0", ID: req.ID, Error: &Error{Code: ErrCodeBadParams, Message: "invalid params"}}
		}
	}

	summary, err := s.handler.HandleSummary(ctx, params.Window)
	if err != nil {
		return &Response{JSONRPC: "2.0", ID: req.ID, Error: &Error{Code: ErrCodeInternal, Message: err.Error()}}
	}
	return &Response{JSONRPC: "2.0", ID: req.ID, Result: summary}
}

func (s *Server) handleEventsRequest(ctx context.Context, req *Request) *Response {
	var params struct {
		ID      string         `json:"id"`
		Source  string         `json:"source"`
		Payload map[string]any `json:"payload"`
		ReplyTo string         `json:"reply_to"`
	}
	if req.Params != nil {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return &Response{JSONRPC: "2.0", ID: req.ID, Error: &Error{Code: ErrCodeBadParams, Message: "invalid params"}}
		}
	}
	if params.ID == "" {
		return &Response{JSONRPC: "2.0", ID: req.ID, Error: &Error{Code: ErrCodeBadParams, Message: "id is required"}}
	}
	if params.Source == "" {
		return &Response{JSONRPC: "2.0", ID: req.ID, Error: &Error{Code: ErrCodeBadParams, Message: "source is required"}}
	}

	event := types.Event{
		ID:      params.ID,
		Source:  types.EventSource(params.Source),
		Type:    "request",
		Ts:      time.Now(),
		Payload: params.Payload,
		ReplyTo: params.ReplyTo,
	}

	eventID, err := s.handler.HandleRequest(ctx, event)
	if err != nil {
		return &Response{JSONRPC: "2.0", ID: req.ID, Error: &Error{Code: ErrCodeInternal, Message: err.Error()}}
	}
	return &Response{JSONRPC: "2.0", ID: req.ID, Result: map[string]string{"request_id": eventID}}
}

func (s *Server) handleEventsAwaitReply(ctx context.Context, req *Request) *Response {
	var params struct {
		RequestID string `json:"request_id"`
		TimeoutMs int    `json:"timeout_ms"`
	}
	if req.Params != nil {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return &Response{JSONRPC: "2.0", ID: req.ID, Error: &Error{Code: ErrCodeBadParams, Message: "invalid params"}}
		}
	}
	if params.RequestID == "" {
		return &Response{JSONRPC: "2.0", ID: req.ID, Error: &Error{Code: ErrCodeBadParams, Message: "request_id is required"}}
	}

	reply, err := s.handler.HandleAwaitReply(ctx, params.RequestID, params.TimeoutMs)
	if err != nil {
		return &Response{JSONRPC: "2.0", ID: req.ID, Error: &Error{Code: ErrCodeInternal, Message: err.Error()}}
	}
	return &Response{JSONRPC: "2.0", ID: req.ID, Result: reply}
}

func (s *Server) handleEventsTrace(ctx context.Context, req *Request) *Response {
	var params struct {
		TraceID string `json:"trace_id"`
	}
	if req.Params != nil {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return &Response{JSONRPC: "2.0", ID: req.ID, Error: &Error{Code: ErrCodeBadParams, Message: "invalid params"}}
		}
	}
	if params.TraceID == "" {
		return &Response{JSONRPC: "2.0", ID: req.ID, Error: &Error{Code: ErrCodeBadParams, Message: "trace_id is required"}}
	}

	events, err := s.handler.HandleTrace(ctx, params.TraceID)
	if err != nil {
		return &Response{JSONRPC: "2.0", ID: req.ID, Error: &Error{Code: ErrCodeInternal, Message: err.Error()}}
	}
	return &Response{JSONRPC: "2.0", ID: req.ID, Result: events}
}

// ServeHTTP handles JSON-RPC over HTTP POST.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Response{
			JSONRPC: "2.0",
			Error:   &Error{Code: ErrCodeParse, Message: "parse error"},
			ID:      nil,
		})
		return
	}

	if req.JSONRPC != "2.0" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Response{
			JSONRPC: "2.0",
			Error:   &Error{Code: ErrCodeInvalidReq, Message: "invalid request: jsonrpc must be \"2.0\""},
			ID:      req.ID,
		})
		return
	}

	resp := s.Dispatch(r.Context(), &req)

	if resp == nil {
		// Notification — no response per JSON-RPC 2.0
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// ServeStdio runs the JSON-RPC server over stdin/stdout (one request per line).
func (s *Server) ServeStdio(ctx context.Context) error {
	return s.ServeIO(ctx, os.Stdin, os.Stdout)
}

// ServeIO runs the JSON-RPC server over arbitrary reader/writer (for testing).
func (s *Server) ServeIO(ctx context.Context, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	encoder := json.NewEncoder(out)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			encoder.Encode(Response{
				JSONRPC: "2.0",
				Error:   &Error{Code: ErrCodeParse, Message: "parse error"},
				ID:      nil,
			})
			continue
		}

		if req.JSONRPC != "2.0" {
			encoder.Encode(Response{
				JSONRPC: "2.0",
				Error:   &Error{Code: ErrCodeInvalidReq, Message: "invalid request: jsonrpc must be \"2.0\""},
				ID:      req.ID,
			})
			continue
		}

		resp := s.Dispatch(ctx, &req)
		if resp != nil {
			encoder.Encode(resp)
		}
	}

	return scanner.Err()
}
