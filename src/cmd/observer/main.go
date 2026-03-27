package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/whitmo/ws-mcp/src/internal/types"
)

func main() {
	url := flag.String("url", "ws://localhost:8080/ws", "WebSocket URL to connect to")
	source := flag.String("source", "", "Filter events by source (e.g. ralph, multiclaude, system)")
	typeFilter := flag.String("type", "", "Filter events by type glob (e.g. task.*, error)")
	ack := flag.Bool("ack", false, "Auto-acknowledge received events")
	pretty := flag.Bool("pretty", false, "Human-readable output instead of raw JSON")
	flag.Parse()

	if err := run(*url, *source, *typeFilter, *ack, *pretty); err != nil {
		log.Fatal(err)
	}
}

func run(url, source, typeFilter string, ack, pretty bool) error {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer conn.Close()

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

			if !matchEvent(event, source, typeFilter) {
				continue
			}

			if pretty {
				printPretty(event)
			} else {
				fmt.Println(string(msg))
			}

			if ack {
				ackMsg := map[string]string{
					"action":   "ack",
					"event_id": event.ID,
				}
				if err := conn.WriteJSON(ackMsg); err != nil {
					log.Printf("ack error: %v", err)
				}
			}
		}
	}()

	select {
	case <-done:
		return nil
	case <-interrupt:
		// Clean close
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

// matchEvent returns true if the event passes the source and type filters.
func matchEvent(e types.Event, source, typeFilter string) bool {
	if source != "" && string(e.Source) != source {
		return false
	}
	if typeFilter != "" && !globMatch(typeFilter, e.Type) {
		return false
	}
	return true
}

// globMatch does simple glob matching using filepath.Match semantics.
func globMatch(pattern, value string) bool {
	// Support "task.*" by converting to filepath.Match pattern
	// filepath.Match uses * to match any non-separator sequence
	matched, err := filepath.Match(pattern, value)
	if err != nil {
		return false
	}
	if matched {
		return true
	}
	// Also try matching with dots as path segments for nested types
	// e.g. "task.*" should match "task.started"
	pattern = strings.ReplaceAll(pattern, ".", "/")
	value = strings.ReplaceAll(value, ".", "/")
	matched, _ = filepath.Match(pattern, value)
	return matched
}

func printPretty(e types.Event) {
	ts := e.Ts.Format("15:04:05")
	payload, _ := json.Marshal(e.Payload)
	acked := ""
	if e.Acked {
		acked = " [acked]"
	}
	fmt.Printf("[%s] %s %s %s%s\n", ts, e.Source, e.Type, string(payload), acked)
}
