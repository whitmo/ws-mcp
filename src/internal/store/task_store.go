package store

import (
	"errors"
	"sync"
	"time"

	"github.com/whitmo/ws-mcp/src/internal/types"
)

// TaskStore holds delegated tasks in a thread-safe map.
type TaskStore struct {
	mu    sync.RWMutex
	tasks map[string]*types.TaskDelegation
}

func NewTaskStore() *TaskStore {
	return &TaskStore{
		tasks: make(map[string]*types.TaskDelegation),
	}
}

func (s *TaskStore) Put(task *types.TaskDelegation) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[task.ID] = task
}

func (s *TaskStore) Get(id string) (*types.TaskDelegation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tasks[id]
	return t, ok
}

func (s *TaskStore) Accept(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[id]
	if !ok {
		return errors.New("task not found")
	}
	if t.Status != types.TaskPending {
		return errors.New("task is not in pending status")
	}
	now := time.Now()
	t.Status = types.TaskAccepted
	t.AcceptedAt = &now
	return nil
}

func (s *TaskStore) Complete(id string, result map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[id]
	if !ok {
		return errors.New("task not found")
	}
	if t.Status != types.TaskAccepted {
		return errors.New("task is not in accepted status")
	}
	now := time.Now()
	t.Status = types.TaskCompleted
	t.CompletedAt = &now
	t.Result = result
	return nil
}

// PendingFor returns tasks delegated to the given agent that are still pending.
func (s *TaskStore) PendingFor(agent string) []*types.TaskDelegation {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*types.TaskDelegation
	for _, t := range s.tasks {
		if t.ToAgent == agent && t.Status == types.TaskPending {
			result = append(result, t)
		}
	}
	return result
}
