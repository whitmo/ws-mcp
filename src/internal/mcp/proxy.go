package mcp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ProxyClient forwards raw JSON-RPC requests to a hub's /rpc endpoint.
type ProxyClient struct {
	hubURL     string
	httpClient *http.Client
}

func NewProxyClient(hubURL string) *ProxyClient {
	return &ProxyClient{
		hubURL: hubURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Forward sends a raw JSON-RPC request to the hub and returns the raw response.
func (p *ProxyClient) Forward(ctx context.Context, reqBytes []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.hubURL, bytes.NewReader(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("proxy: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("proxy: hub unreachable: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("proxy: read response: %w", err)
	}

	return body, nil
}

// Ping checks if the hub is reachable within the given timeout.
func (p *ProxyClient) Ping(timeout time.Duration) bool {
	client := &http.Client{Timeout: timeout}
	probe := []byte(`{"jsonrpc":"2.0","method":"initialize","params":{},"id":"probe"}`)
	resp, err := client.Post(p.hubURL, "application/json", bytes.NewReader(probe))
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
