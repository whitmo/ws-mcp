package types

import "testing"

func TestIsKnownEventType(t *testing.T) {
	known := []string{
		"task.started", "task.completed", "task.failed",
		"commit.pushed",
		"pr.opened", "pr.merged", "pr.reviewed",
		"review.requested", "review.completed",
		"agent.started", "agent.stopped",
		"system.healthcheck", "system.error",
	}
	for _, et := range known {
		if !IsKnownEventType(et) {
			t.Errorf("expected %q to be known", et)
		}
	}
}

func TestIsKnownEventType_Unknown(t *testing.T) {
	unknown := []string{"", "foo", "task.unknown", "custom.event"}
	for _, et := range unknown {
		if IsKnownEventType(et) {
			t.Errorf("expected %q to be unknown", et)
		}
	}
}

func TestKnownEventTypesCount(t *testing.T) {
	if len(KnownEventTypes) != 13 {
		t.Errorf("expected 13 known event types, got %d", len(KnownEventTypes))
	}
}
