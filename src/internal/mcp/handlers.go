package mcp

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/whitmo/ws-mcp/src/internal/store"
	"github.com/whitmo/ws-mcp/src/internal/types"
)

type Handler struct {
	store     *store.RingBuffer
	taskStore *store.TaskStore
}

func NewHandler(s *store.RingBuffer) *Handler {
	return &Handler{store: s, taskStore: store.NewTaskStore()}
}

// NewHandlerWithTaskStore creates a Handler with an explicit TaskStore (for testing).
func NewHandlerWithTaskStore(s *store.RingBuffer, ts *store.TaskStore) *Handler {
	return &Handler{store: s, taskStore: ts}
}

func (h *Handler) HandleLatest(ctx context.Context, limit int) ([]types.Event, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	events := h.store.Latest(limit)
	return events, nil
}

func (h *Handler) HandleFilter(ctx context.Context, source string, excludeType string) ([]types.Event, error) {
	// Fetch latest 100 and filter in memory
	all := h.store.Latest(100)
	filtered := make([]types.Event, 0)

	for _, e := range all {
		if source != "" && string(e.Source) != source {
			continue
		}
		if excludeType != "" && e.Type == excludeType {
			continue
		}
		filtered = append(filtered, e)
	}

	return filtered, nil
}

func (h *Handler) HandleAck(ctx context.Context, id string, ackedBy string) error {
	return h.store.Ack(id, ackedBy)
	// Note: Broadcasting the acked state over WS (T026) would require h.hub.Broadcast(event)
	// We'll leave the WS broadcast out of this minimal handler for now, or assume
	// clients query for updated state.
}

func (h *Handler) HandleRequest(ctx context.Context, event types.Event) (string, error) {
	if event.Type != "request" {
		return "", errors.New("event type must be 'request'")
	}
	if event.ID == "" {
		return "", errors.New("event ID is required")
	}
	h.store.Push(event)
	return event.ID, nil
}

func (h *Handler) HandleAwaitReply(ctx context.Context, requestID string, timeoutMs int) (*types.Event, error) {
	if requestID == "" {
		return nil, errors.New("request_id is required")
	}

	// Verify the request event exists
	if _, found := h.store.FindByID(requestID); !found {
		return nil, fmt.Errorf("request event %q not found", requestID)
	}

	if timeoutMs <= 0 {
		timeoutMs = 30000
	}

	deadline := time.After(time.Duration(timeoutMs) * time.Millisecond)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline:
			return nil, fmt.Errorf("timeout waiting for reply to %q after %dms", requestID, timeoutMs)
		case <-ticker.C:
			if reply, found := h.store.FindByInReplyTo(requestID); found {
				return &reply, nil
			}
		}
	}
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

// HandleTaskDelegate creates a new task delegation.
func (h *Handler) HandleTaskDelegate(ctx context.Context, fromAgent, toAgent, description string) (string, error) {
	if fromAgent == "" {
		return "", errors.New("from_agent is required")
	}
	if toAgent == "" {
		return "", errors.New("to_agent is required")
	}
	if description == "" {
		return "", errors.New("description is required")
	}

	id := fmt.Sprintf("task-%d", time.Now().UnixNano())
	task := &types.TaskDelegation{
		ID:          id,
		FromAgent:   fromAgent,
		ToAgent:     toAgent,
		Description: description,
		Status:      types.TaskPending,
		CreatedAt:   time.Now(),
	}
	h.taskStore.Put(task)
	return id, nil
}

// HandleTaskAccept marks a task as accepted.
func (h *Handler) HandleTaskAccept(ctx context.Context, taskID string) error {
	if taskID == "" {
		return errors.New("task_id is required")
	}
	return h.taskStore.Accept(taskID)
}

// HandleTaskComplete marks a task as completed with a result.
func (h *Handler) HandleTaskComplete(ctx context.Context, taskID string, result map[string]any) error {
	if taskID == "" {
		return errors.New("task_id is required")
	}
	return h.taskStore.Complete(taskID, result)
}

// HandleTaskPending returns pending tasks for the given agent.
func (h *Handler) HandleTaskPending(ctx context.Context, agent string) ([]*types.TaskDelegation, error) {
	if agent == "" {
		return nil, errors.New("agent is required")
	}
	tasks := h.taskStore.PendingFor(agent)
	if tasks == nil {
		tasks = []*types.TaskDelegation{}
	}
	return tasks, nil
}

// HandleTaskStatus returns the status of a task by ID.
func (h *Handler) HandleTaskStatus(ctx context.Context, taskID string) (*types.TaskDelegation, error) {
	if taskID == "" {
		return nil, errors.New("task_id is required")
	}
	task, ok := h.taskStore.Get(taskID)
	if !ok {
		return nil, fmt.Errorf("task %q not found", taskID)
	}
	return task, nil
}
