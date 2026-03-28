package types

import "time"

// EventSource defines the valid origins for an event.
type EventSource string

const (
	SourceRalph       EventSource = "ralph"
	SourceMultiClaude EventSource = "multiclaude"
	SourceSystem      EventSource = "system"
)

// Event represents an atomic occurrence from an agent or system.
type Event struct {
	ID        string         `json:"id"`
	Source    EventSource    `json:"source"`
	Type      string         `json:"type"`
	Ts        time.Time      `json:"ts"`
	Payload   map[string]any `json:"payload"`
	Acked     bool           `json:"acked"`
	AckedBy   string         `json:"acked_by,omitempty"`
	AckedTs   *time.Time     `json:"acked_ts,omitempty"`
	ReplyTo   string         `json:"reply_to,omitempty"`
	InReplyTo string         `json:"in_reply_to,omitempty"`
	// Tracing: links events into causal chains across agents
	TraceID  string `json:"trace_id,omitempty"`
	ParentID string `json:"parent_id,omitempty"`
	// Cross-repo: identifies which repo an event originated from
	Repo string `json:"repo,omitempty"`
	// Agent identity
	AgentID string `json:"agent_id,omitempty"`
}
