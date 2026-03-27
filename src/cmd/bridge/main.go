package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/whitmo/ws-mcp/src/internal/hub"
	"github.com/whitmo/ws-mcp/src/internal/store"
	"github.com/whitmo/ws-mcp/src/pkg/api"
)

func main() {
	fmt.Println("MCP Bridge Service initializing...")

	// Handle graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Initialize components
	buf := store.NewRingBuffer(2000)
	h := hub.NewHub()
	go h.Run()

	router := api.NewRouter(buf)
	router.SetHub(h)
	mux := router.SetupRoutes()

	go func() {
		<-stop
		fmt.Println("\nShutting down gracefully...")
		h.Stop()
		os.Exit(0)
	}()

	port := "8080"
	fmt.Printf("Listening on :%s\n", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
