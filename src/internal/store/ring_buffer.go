package store

import (
	"errors"
	"sync"
	"time"

	"github.com/whitmo/ws-mcp/src/internal/types"
)

type RingBuffer struct {
	mu     sync.RWMutex
	events []types.Event
	max    int
}

func NewRingBuffer(max int) *RingBuffer {
	return &RingBuffer{
		events: make([]types.Event, 0, max),
		max:    max,
	}
}

func (r *RingBuffer) Push(event types.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.events) == r.max {
		// Evict oldest by shifting
		r.events = r.events[1:]
	}
	r.events = append(r.events, event)
}

func (r *RingBuffer) Latest(limit int) []types.Event {
	r.mu.RLock()
	defer r.mu.RUnlock()

	n := len(r.events)
	if limit > n {
		limit = n
	}

	result := make([]types.Event, limit)
	// Return in descending chronological order
	for i := 0; i < limit; i++ {
		result[i] = r.events[n-1-i]
	}
	return result
}

func (r *RingBuffer) FindByID(id string) (types.Event, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for i := len(r.events) - 1; i >= 0; i-- {
		if r.events[i].ID == id {
			return r.events[i], true
		}
	}
	return types.Event{}, false
}

func (r *RingBuffer) FindByInReplyTo(requestID string) (types.Event, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for i := len(r.events) - 1; i >= 0; i-- {
		if r.events[i].InReplyTo == requestID {
			return r.events[i], true
		}
	}
	return types.Event{}, false
}

func (r *RingBuffer) Ack(id string, by string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := len(r.events) - 1; i >= 0; i-- {
		if r.events[i].ID == id {
			if r.events[i].Acked {
				return errors.New("event already acked")
			}
			r.events[i].Acked = true
			r.events[i].AckedBy = by
			now := time.Now()
			r.events[i].AckedTs = &now
			return nil
		}
	}
	return errors.New("event not found")
}
