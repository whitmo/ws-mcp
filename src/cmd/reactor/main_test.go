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

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
