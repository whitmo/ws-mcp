package mcp

import (
	"context"
	"time"

	"github.com/whitmo/ws-mcp/src/internal/store"
	"github.com/whitmo/ws-mcp/src/internal/types"
)

type Handler struct {
	store *store.RingBuffer
}

func NewHandler(s *store.RingBuffer) *Handler {
	return &Handler{store: s}
}

func (h *Handler) HandleLatest(ctx context.Context, limit int) ([]types.Event, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	events := h.store.Latest(limit)
	return events, nil
}

func (h *Handler) HandleFilter(ctx context.Context, source string) ([]types.Event, error) {
	// For MVP, we fetch latest 100 and filter in memory
	all := h.store.Latest(100)
	var filtered []types.Event
	
	for _, e := range all {
		if string(e.Source) == source {
			filtered = append(filtered, e)
		}
	}

	return filtered, nil
}

func (h *Handler) HandleAck(ctx context.Context, id string, ackedBy string) error {
	return h.store.Ack(id, ackedBy)
	// Note: Broadcasting the acked state over WS (T026) would require h.hub.Broadcast(event)
	// We'll leave the WS broadcast out of this minimal handler for now, or assume
	// clients query for updated state.
}

// SummaryResult holds the report.summary response.
type SummaryResult struct {
	Window       int                       `json:"window"`
	TotalEvents  int                       `json:"total_events"`
	BySource     map[string]int            `json:"by_source"`
	ByType       map[string]int            `json:"by_type"`
	AckedCount   int                       `json:"acked_count"`
	UnackedCount int                       `json:"unacked_count"`
	Alerts       []string                  `json:"alerts"`
}

func (h *Handler) HandleSummary(ctx context.Context, windowMinutes int) (*SummaryResult, error) {
	if windowMinutes <= 0 {
		windowMinutes = 60
	}

	cutoff := time.Now().Add(-time.Duration(windowMinutes) * time.Minute)
	all := h.store.Latest(100)

	result := &SummaryResult{
		Window:   windowMinutes,
		BySource: make(map[string]int),
		ByType:   make(map[string]int),
	}

	for _, e := range all {
		if e.Ts.Before(cutoff) {
			continue
		}
		result.TotalEvents++
		result.BySource[string(e.Source)]++
		result.ByType[e.Type]++
		if e.Acked {
			result.AckedCount++
		} else {
			result.UnackedCount++
		}
	}

	if result.UnackedCount > 10 {
		result.Alerts = append(result.Alerts, "high number of unacknowledged events")
	}
	if result.ByType["error"] > 0 {
		result.Alerts = append(result.Alerts, "errors detected in window")
	}
	if result.Alerts == nil {
		result.Alerts = []string{}
	}

	return result, nil
}
