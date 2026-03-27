package api

import (
	"net/http"

	"github.com/whitmo/ws-mcp/src/internal/hub"
	"github.com/whitmo/ws-mcp/src/internal/store"
)

type Router struct {
	store     *store.RingBuffer
	hub       *hub.Hub
	fileStore *store.FileStore
}

func NewRouter(s *store.RingBuffer) *Router {
	return &Router{
		store: s,
	}
}

func (r *Router) SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mux.HandleFunc("/event", r.handleIngest())
	mux.HandleFunc("/ws", r.handleWebSocket())

	return mux
}
