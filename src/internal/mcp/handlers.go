package mcp

import (
	"context"

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
	
func (h *Handler) HandleAck(ctx context.Context, id string, ackedBy string) error {
	return h.store.Ack(id, ackedBy)
	// Note: Broadcasting the acked state over WS (T026) would require h.hub.Broadcast(event)
	// We'll leave the WS broadcast out of this minimal handler for now, or assume 
	// clients query for updated state.
}
