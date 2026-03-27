package main

import (
	"os"
	"testing"
	"time"

	"github.com/whitmo/ws-mcp/src/internal/types"
)

func TestMatchRule(t *testing.T) {
	tests := []struct {
		name  string
		rule  Rule
		event types.Event
		want  bool
	}{
		{
			name: "exact source and type match",
			rule: Rule{Source: "multiclaude", Type: "review.requested"},
			event: types.Event{
				Source: types.SourceMultiClaude,
				Type:   "review.requested",
			},
			want: true,
		},
		{
			name: "source mismatch",
			rule: Rule{Source: "ralph", Type: "review.requested"},
			event: types.Event{
				Source: types.SourceMultiClaude,
				Type:   "review.requested",
			},
			want: false,
		},
		{
			name: "type glob match",
			rule: Rule{Source: "ralph", Type: "task.*"},
			event: types.Event{
				Source: types.SourceRalph,
				Type:   "task.completed",
			},
			want: true,
		},
		{
			name: "type glob no match",
			rule: Rule{Source: "ralph", Type: "task.*"},
			event: types.Event{
				Source: types.SourceRalph,
				Type:   "error.fatal",
			},
			want: false,
		},
		{
			name: "empty source matches any",
			rule: Rule{Type: "error.*"},
			event: types.Event{
				Source: types.SourceSystem,
				Type:   "error.timeout",
			},
			want: true,
		},
		{
			name: "empty type matches any",
			rule: Rule{Source: "multiclaude"},
			event: types.Event{
				Source: types.SourceMultiClaude,
				Type:   "anything.here",
			},
			want: true,
		},
		{
			name: "both empty matches all",
			rule: Rule{},
			event: types.Event{
				Source: types.SourceRalph,
				Type:   "whatever",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchRule(tt.rule, tt.event)
			if got != tt.want {
				t.Errorf("MatchRule() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExpandVars(t *testing.T) {
	event := types.Event{
		ID:     "evt-123",
		Source: types.SourceMultiClaude,
		Type:   "review.requested",
		Payload: map[string]any{
			"pr":   "42",
			"repo": "ws-mcp",
		},
	}

	tests := []struct {
		name string
		cmd  string
		want string
	}{
		{
			name: "payload var",
			cmd:  "multiclaude review $PR",
			want: "multiclaude review 42",
		},
		{
			name: "braced var",
			cmd:  "echo ${REPO}#${PR}",
			want: "echo ws-mcp#42",
		},
		{
			name: "event metadata",
			cmd:  "echo $SOURCE $TYPE $ID",
			want: "echo multiclaude review.requested evt-123",
		},
		{
			name: "no vars",
			cmd:  "echo hello",
			want: "echo hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandVars(tt.cmd, event)
			if got != tt.want {
				t.Errorf("ExpandVars() = %q, want %q", got, tt.want)
			}
		})
	}
}

// MockExecutor records calls for testing.
type MockExecutor struct {
	Commands []string
	Posts    []string
	Err      error
}

func (m *MockExecutor) ExecCommand(cmd string) error {
	m.Commands = append(m.Commands, cmd)
	return m.Err
}

func (m *MockExecutor) PostURL(url string, event types.Event) error {
	m.Posts = append(m.Posts, url)
	return m.Err
}

func TestDispatch(t *testing.T) {
	rules := []Rule{
		{
			Name:    "review-trigger",
			Source:  "multiclaude",
			Type:    "review.requested",
			Command: "multiclaude review $PR",
		},
		{
			Name:    "error-webhook",
			Source:  "ralph",
			Type:    "error.*",
			PostURL: "http://hooks.example.com/alert",
		},
		{
			Name:    "catch-all-log",
			Command: "echo event: $TYPE from $SOURCE",
		},
	}

	t.Run("matching command rule", func(t *testing.T) {
		mock := &MockExecutor{}
		event := types.Event{
			ID:      "evt-1",
			Source:  types.SourceMultiClaude,
			Type:    "review.requested",
			Ts:      time.Now(),
			Payload: map[string]any{"pr": "99"},
		}
		errs := Dispatch(rules, event, mock)
		if len(errs) > 0 {
			t.Errorf("unexpected errors: %v", errs)
		}
		// Should match review-trigger and catch-all-log
		if len(mock.Commands) != 2 {
			t.Fatalf("expected 2 commands, got %d: %v", len(mock.Commands), mock.Commands)
		}
		if mock.Commands[0] != "multiclaude review 99" {
			t.Errorf("command[0] = %q, want %q", mock.Commands[0], "multiclaude review 99")
		}
		if mock.Commands[1] != "echo event: review.requested from multiclaude" {
			t.Errorf("command[1] = %q, want %q", mock.Commands[1], "echo event: review.requested from multiclaude")
		}
	})

	t.Run("matching post rule", func(t *testing.T) {
		mock := &MockExecutor{}
		event := types.Event{
			ID:     "evt-2",
			Source: types.SourceRalph,
			Type:   "error.fatal",
			Ts:     time.Now(),
		}
		errs := Dispatch(rules, event, mock)
		if len(errs) > 0 {
			t.Errorf("unexpected errors: %v", errs)
		}
		if len(mock.Posts) != 1 {
			t.Fatalf("expected 1 post, got %d", len(mock.Posts))
		}
		if mock.Posts[0] != "http://hooks.example.com/alert" {
			t.Errorf("post url = %q, want %q", mock.Posts[0], "http://hooks.example.com/alert")
		}
		// catch-all also fires
		if len(mock.Commands) != 1 {
			t.Fatalf("expected 1 command from catch-all, got %d", len(mock.Commands))
		}
	})

	t.Run("no matching rules", func(t *testing.T) {
		mock := &MockExecutor{}
		rules := []Rule{
			{Name: "specific", Source: "multiclaude", Type: "deploy.started"},
		}
		event := types.Event{
			Source: types.SourceRalph,
			Type:   "task.completed",
		}
		errs := Dispatch(rules, event, mock)
		if len(errs) > 0 {
			t.Errorf("unexpected errors: %v", errs)
		}
		if len(mock.Commands) != 0 || len(mock.Posts) != 0 {
			t.Error("expected no actions executed")
		}
	})
}

func TestLoadConfig(t *testing.T) {
	// Write a temp config
	dir := t.TempDir()
	path := dir + "/test-rules.yaml"
	data := `rules:
  - name: review
    source: multiclaude
    type: review.requested
    command: "multiclaude review $PR"
  - name: error-alert
    source: ralph
    type: "error.*"
    post_url: "http://hooks.example.com/alert"
`
	if err := writeFile(path, data); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if len(cfg.Rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(cfg.Rules))
	}
	if cfg.Rules[0].Name != "review" {
		t.Errorf("rule[0].Name = %q, want %q", cfg.Rules[0].Name, "review")
	}
	if cfg.Rules[1].PostURL != "http://hooks.example.com/alert" {
		t.Errorf("rule[1].PostURL = %q", cfg.Rules[1].PostURL)
	}
}

func TestLoadConfig_empty(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/empty.yaml"
	if err := writeFile(path, "rules: []\n"); err != nil {
		t.Fatal(err)
	}
	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for empty rules")
	}
}

func TestLoadReactorConfig(t *testing.T) {
	// Load the real configs/reactor.yaml
	cfg, err := LoadConfig("../../../configs/reactor.yaml")
	if err != nil {
		t.Fatalf("failed to load configs/reactor.yaml: %v", err)
	}
	if len(cfg.Rules) != 4 {
		t.Fatalf("expected 4 rules, got %d", len(cfg.Rules))
	}

	// Verify rule names
	expected := []string{"review-trigger", "task-failure-notify", "agent-started-log", "agent-stopped-log"}
	for i, name := range expected {
		if cfg.Rules[i].Name != name {
			t.Errorf("rule[%d].Name = %q, want %q", i, cfg.Rules[i].Name, name)
		}
	}
}

func TestReactorConfigMatchesEvents(t *testing.T) {
	cfg, err := LoadConfig("../../../configs/reactor.yaml")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	ts := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		event       types.Event
		wantMatches []string // expected rule names that match
		wantCmd     string   // expected expanded command from first matching rule
	}{
		{
			name: "review.requested matches review-trigger",
			event: types.Event{
				Source:  types.SourceMultiClaude,
				Type:    "review.requested",
				Ts:      ts,
				Payload: map[string]any{"pr_number": "42", "reviewer": "bot"},
			},
			wantMatches: []string{"review-trigger"},
			wantCmd:     "multiclaude review 42",
		},
		{
			name: "task.failed matches task-failure-notify",
			event: types.Event{
				Source:  types.SourceRalph,
				Type:    "task.failed",
				Ts:      ts,
				Payload: map[string]any{"task_id": "build-123", "agent": "worker-1", "reason": "timeout"},
			},
			wantMatches: []string{"task-failure-notify"},
			wantCmd:     `terminal-notifier -title ws-mcp -message "Task failed: build-123"`,
		},
		{
			name: "agent.started matches agent-started-log",
			event: types.Event{
				Source:  types.SourceMultiClaude,
				Type:    "agent.started",
				Ts:      ts,
				Payload: map[string]any{"agent": "lively-squirrel", "worktree": "/tmp/wt"},
			},
			wantMatches: []string{"agent-started-log"},
			wantCmd:     `echo "2026-03-27T12:00:00Z lively-squirrel started" >> ~/.bridge/logs/agents.log`,
		},
		{
			name: "agent.stopped matches agent-stopped-log",
			event: types.Event{
				Source:  types.SourceMultiClaude,
				Type:    "agent.stopped",
				Ts:      ts,
				Payload: map[string]any{"agent": "bold-lion", "reason": "completed"},
			},
			wantMatches: []string{"agent-stopped-log"},
			wantCmd:     `echo "2026-03-27T12:00:00Z bold-lion stopped" >> ~/.bridge/logs/agents.log`,
		},
		{
			name: "unrelated event matches nothing",
			event: types.Event{
				Source: types.SourceSystem,
				Type:   "system.healthcheck",
				Ts:     ts,
			},
			wantMatches: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var matched []string
			for _, rule := range cfg.Rules {
				if MatchRule(rule, tt.event) {
					matched = append(matched, rule.Name)
				}
			}

			if len(matched) != len(tt.wantMatches) {
				t.Fatalf("matched rules = %v, want %v", matched, tt.wantMatches)
			}
			for i := range matched {
				if matched[i] != tt.wantMatches[i] {
					t.Errorf("matched[%d] = %q, want %q", i, matched[i], tt.wantMatches[i])
				}
			}

			// Check command expansion for first matching rule
			if tt.wantCmd != "" && len(matched) > 0 {
				for _, rule := range cfg.Rules {
					if rule.Name == matched[0] {
						got := ExpandVars(rule.Command, tt.event)
						if got != tt.wantCmd {
							t.Errorf("expanded command = %q, want %q", got, tt.wantCmd)
						}
						break
					}
				}
			}
		})
	}
}

func TestExpandVarsTimestamp(t *testing.T) {
	ts := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)
	event := types.Event{
		Source:  types.SourceMultiClaude,
		Type:    "agent.started",
		Ts:      ts,
		Payload: map[string]any{"agent": "test-worker"},
	}
	got := ExpandVars("echo $TIMESTAMP $AGENT", event)
	want := "echo 2026-03-27T12:00:00Z test-worker"
	if got != want {
		t.Errorf("ExpandVars() = %q, want %q", got, want)
	}
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
