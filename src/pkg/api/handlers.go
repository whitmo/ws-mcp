package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/whitmo/ws-mcp/src/internal/hub"
	"github.com/whitmo/ws-mcp/src/internal/store"
	"github.com/whitmo/ws-mcp/src/internal/types"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // Allow all for MVP
}

func (r *Router) SetHub(h *hub.Hub) {
	r.hub = h
}

func (r *Router) SetFileStore(fs *store.FileStore) {
	r.fileStore = fs
}

// Add hub field to Router struct
// Ensure this works with the existing router.go

func (r *Router) handleIngest() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var event types.Event
		if err := json.NewDecoder(req.Body).Decode(&event); err != nil {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}

		// Validation (T016)
		if event.ID == "" || event.Type == "" || event.Ts.IsZero() {
			http.Error(w, "Missing required fields", http.StatusBadRequest)
			return
		}
		if event.Source != types.SourceRalph && event.Source != types.SourceMultiClaude && event.Source != types.SourceSystem {
			http.Error(w, "Invalid source", http.StatusBadRequest)
			return
		}

		// Warn on unknown event types (accept but log)
		if !types.IsKnownEventType(event.Type) {
			log.Printf("WARN: unknown event type %q from source %q (id=%s)", event.Type, event.Source, event.ID)
		}

		// Store Event
		if r.store != nil {
			r.store.Push(event)
		}

		// Persist to durable file store
		if r.fileStore != nil {
			if err := r.fileStore.Append(event); err != nil {
				log.Printf("file store append error: %v", err)
			}
		}

		// Broadcast Event
		if r.hub != nil {
			r.hub.Broadcast(event)
		}

		// Local system notification (T028)
		if event.Type == "error" {
			go func() {
				// Fire-and-forget local notification (e.g. macOS say)
				// cmd := exec.Command("say", "Agent Error Detected")
				// _ = cmd.Run()
			}()
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

func (r *Router) handleWebSocket() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		conn, err := upgrader.Upgrade(w, req, nil)
		if err != nil {
			return // Upgrade handles writing the error
		}

		if r.hub != nil {
			r.hub.RegisterClient(conn)
		} else {
			conn.Close()
		}
	}
}
