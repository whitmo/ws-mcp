package types

// EventType represents a standard event type in the ws-mcp vocabulary.
type EventType string

const (
	EventTaskStarted   EventType = "task.started"
	EventTaskCompleted EventType = "task.completed"
	EventTaskFailed    EventType = "task.failed"

	EventCommitPushed EventType = "commit.pushed"

	EventPROpened   EventType = "pr.opened"
	EventPRMerged   EventType = "pr.merged"
	EventPRReviewed EventType = "pr.reviewed"

	EventReviewRequested EventType = "review.requested"
	EventReviewCompleted EventType = "review.completed"

	EventAgentStarted EventType = "agent.started"
	EventAgentStopped EventType = "agent.stopped"

	EventSystemHealthcheck EventType = "system.healthcheck"
	EventSystemError       EventType = "system.error"
)

// KnownEventTypes is the set of all standard event types.
var KnownEventTypes = map[EventType]bool{
	EventTaskStarted:       true,
	EventTaskCompleted:     true,
	EventTaskFailed:        true,
	EventCommitPushed:      true,
	EventPROpened:          true,
	EventPRMerged:          true,
	EventPRReviewed:        true,
	EventReviewRequested:   true,
	EventReviewCompleted:   true,
	EventAgentStarted:      true,
	EventAgentStopped:      true,
	EventSystemHealthcheck: true,
	EventSystemError:       true,
}

// IsKnownEventType reports whether the given type string is in the standard vocabulary.
func IsKnownEventType(t string) bool {
	return KnownEventTypes[EventType(t)]
}
