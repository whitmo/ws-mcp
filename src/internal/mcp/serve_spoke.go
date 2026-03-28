package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
)

// ServeSpoke runs an MCP stdio server that forwards all requests to the hub
// via the ProxyClient. Notifications (no ID) are swallowed locally.
func ServeSpoke(ctx context.Context, proxy *ProxyClient, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)

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

		// Peek at the request to detect notifications (no ID = no response expected)
		var peek struct {
			Method string `json:"method"`
			ID     any    `json:"id"`
		}
		if err := json.Unmarshal(line, &peek); err != nil {
			// Malformed — write parse error locally
			errResp, _ := json.Marshal(Response{
				JSONRPC: "2.0",
				Error:   &Error{Code: ErrCodeParse, Message: "parse error"},
			})
			out.Write(errResp)
			out.Write([]byte("\n"))
			continue
		}

		// Notifications have no ID — don't expect a response
		if peek.ID == nil {
			// Forward to hub (fire-and-forget) but don't write response
			proxy.Forward(ctx, line)
			continue
		}

		// Forward to hub and relay response
		respBytes, err := proxy.Forward(ctx, line)
		if err != nil {
			// Hub error — return JSON-RPC internal error
			errResp, _ := json.Marshal(Response{
				JSONRPC: "2.0",
				Error:   &Error{Code: ErrCodeInternal, Message: "hub unreachable: " + err.Error()},
				ID:      peek.ID,
			})
			out.Write(errResp)
			out.Write([]byte("\n"))
			continue
		}

		out.Write(respBytes)
		// Ensure newline termination
		if len(respBytes) > 0 && respBytes[len(respBytes)-1] != '\n' {
			out.Write([]byte("\n"))
		}
	}

	return scanner.Err()
}
