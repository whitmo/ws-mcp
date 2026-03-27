package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/whitmo/ws-mcp/src/internal/types"
	"gopkg.in/yaml.v3"
)

// Rule defines a single event-matching rule and its action.
type Rule struct {
	Name    string `yaml:"name"`
	Source  string `yaml:"source"`
	Type    string `yaml:"type"`
	Command string `yaml:"command,omitempty"`
	PostURL string `yaml:"post_url,omitempty"`
}

// Config holds the reactor configuration.
type Config struct {
	Rules []Rule `yaml:"rules"`
}

func main() {
	url := flag.String("url", "ws://localhost:8080/ws", "WebSocket URL to connect to")
	configPath := flag.String("config", "reactor.yaml", "Path to rules config file")
	flag.Parse()

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	log.Printf("loaded %d rules from %s", len(cfg.Rules), *configPath)

	if err := Run(*url, cfg, DefaultExecutor{}); err != nil {
		log.Fatal(err)
	}
}

// LoadConfig reads and parses a YAML config file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if len(cfg.Rules) == 0 {
		return nil, fmt.Errorf("no rules defined in config")
	}
	return &cfg, nil
}

// Executor defines how actions are executed. Allows testing with mocks.
type Executor interface {
	ExecCommand(cmd string) error
	PostURL(url string, event types.Event) error
}

// DefaultExecutor runs real shell commands and HTTP posts.
type DefaultExecutor struct{}

func (DefaultExecutor) ExecCommand(cmd string) error {
	c := exec.Command("sh", "-c", cmd)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func (DefaultExecutor) PostURL(url string, event types.Event) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("POST %s returned %d", url, resp.StatusCode)
	}
	return nil
}

// MatchRule checks if an event matches a rule's source and type patterns.
func MatchRule(rule Rule, event types.Event) bool {
	if rule.Source != "" && rule.Source != string(event.Source) {
		return false
	}
	if rule.Type != "" && !globMatch(rule.Type, event.Type) {
		return false
	}
	return true
}

// globMatch does simple glob matching, same as the observer.
func globMatch(pattern, value string) bool {
	matched, err := filepath.Match(pattern, value)
	if err != nil {
		return false
	}
	if matched {
		return true
	}
	pattern = strings.ReplaceAll(pattern, ".", "/")
	value = strings.ReplaceAll(value, ".", "/")
	matched, _ = filepath.Match(pattern, value)
	return matched
}

// ExpandVars replaces $KEY and ${KEY} in cmd with values from the event payload.
// Also supports $SOURCE, $TYPE, $ID from the event itself.
func ExpandVars(cmd string, event types.Event) string {
	replacements := map[string]string{
		"SOURCE": string(event.Source),
		"TYPE":   event.Type,
		"ID":     event.ID,
	}
	for k, v := range event.Payload {
		if s, ok := v.(string); ok {
			replacements[strings.ToUpper(k)] = s
		}
	}

	result := cmd
	for k, v := range replacements {
		result = strings.ReplaceAll(result, "${"+k+"}", v)
		result = strings.ReplaceAll(result, "$"+k, v)
	}
	return result
}

// Dispatch finds matching rules for an event and executes their actions.
func Dispatch(rules []Rule, event types.Event, exec Executor) []error {
	var errs []error
	for _, rule := range rules {
		if !MatchRule(rule, event) {
			continue
		}
		log.Printf("rule %q matched event %s/%s", rule.Name, event.Source, event.Type)

		if rule.Command != "" {
			expanded := ExpandVars(rule.Command, event)
			log.Printf("exec: %s", expanded)
			if err := exec.ExecCommand(expanded); err != nil {
				log.Printf("command error: %v", err)
				errs = append(errs, fmt.Errorf("rule %q command: %w", rule.Name, err))
			}
		}
		if rule.PostURL != "" {
			expanded := ExpandVars(rule.PostURL, event)
			log.Printf("POST: %s", expanded)
			if err := exec.PostURL(expanded, event); err != nil {
				log.Printf("post error: %v", err)
				errs = append(errs, fmt.Errorf("rule %q post: %w", rule.Name, err))
			}
		}
	}
	return errs
}

// Run connects to the WebSocket and dispatches events.
func Run(url string, cfg *Config, exec Executor) error {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer conn.Close()

	log.Printf("connected to %s", url)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					return
				}
				log.Printf("read error: %v", err)
				return
			}

			var event types.Event
			if err := json.Unmarshal(msg, &event); err != nil {
				log.Printf("decode error: %v", err)
				continue
			}

			Dispatch(cfg.Rules, event, exec)
		}
	}()

	select {
	case <-done:
		return nil
	case <-interrupt:
		err := conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		)
		if err != nil {
			return err
		}
		select {
		case <-done:
		case <-time.After(time.Second):
		}
		return nil
	}
}
