package types

import "time"

// TaskStatus represents the lifecycle state of a delegated task.
type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskAccepted  TaskStatus = "accepted"
	TaskCompleted TaskStatus = "completed"
	TaskFailed    TaskStatus = "failed"
)

// TaskDelegation represents a task delegated from one agent to another.
type TaskDelegation struct {
	ID          string         `json:"id"`
	FromAgent   string         `json:"from_agent"`
	ToAgent     string         `json:"to_agent"`
	Description string         `json:"description"`
	Status      TaskStatus     `json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	AcceptedAt  *time.Time     `json:"accepted_at,omitempty"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	Result      map[string]any `json:"result,omitempty"`
}
