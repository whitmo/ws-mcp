package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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

// Server dispatches JSON-RPC 2.0 calls to MCP handlers.
type Server struct {
	handler *Handler
}

func NewServer(h *Handler) *Server {
	return &Server{handler: h}
}

// Dispatch routes a JSON-RPC request to the appropriate handler.
func (s *Server) Dispatch(ctx context.Context, req *Request) *Response {
	resp := &Response{JSONRPC: "2.0", ID: req.ID}

	switch req.Method {
	case "events.latest":
		resp = s.handleEventsLatest(ctx, req)
	case "events.filter":
		resp = s.handleEventsFilter(ctx, req)
	case "events.ack":
		resp = s.handleEventsAck(ctx, req)
	case "report.summary":
		resp = s.handleReportSummary(ctx, req)
	default:
		resp.Error = &Error{Code: ErrCodeNoMethod, Message: fmt.Sprintf("method not found: %s", req.Method)}
	}

	return resp
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
		Source string `json:"source"`
	}
	if req.Params != nil {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return &Response{JSONRPC: "2.0", ID: req.ID, Error: &Error{Code: ErrCodeBadParams, Message: "invalid params"}}
		}
	}

	events, err := s.handler.HandleFilter(ctx, params.Source)
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
		encoder.Encode(resp)
	}

	return scanner.Err()
}
