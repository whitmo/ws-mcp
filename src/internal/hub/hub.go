package hub

import (
	"log"
	"sync"
	"github.com/whitmo/ws-mcp/src/internal/types"
	"github.com/gorilla/websocket"
)

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan types.Event
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan types.Event
	register   chan *Client
	unregister chan *Client
	mu         sync.Mutex
	done       chan struct{}
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan types.Event),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		done:       make(chan struct{}),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
		case event := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				select {
				case client.send <- event:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
		case <-h.done:
			return
		}
	}
}

func (h *Hub) Stop() {
	close(h.done)
}

func (h *Hub) Broadcast(event types.Event) {
	h.broadcast <- event
}

func (h *Hub) RegisterClient(conn *websocket.Conn) *Client {
	client := &Client{hub: h, conn: conn, send: make(chan types.Event, 256)}
	h.register <- client
	
	// Start pump to client
	go func() {
		defer func() {
			h.unregister <- client
			conn.Close()
		}()
		for event := range client.send {
			if err := conn.WriteJSON(event); err != nil {
				log.Printf("WS write error: %v", err)
				return
			}
		}
	}()
	
	return client
}
