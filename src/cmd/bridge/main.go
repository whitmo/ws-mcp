package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/whitmo/ws-mcp/src/internal/hub"
	"github.com/whitmo/ws-mcp/src/internal/mcp"
	"github.com/whitmo/ws-mcp/src/internal/store"
	"github.com/whitmo/ws-mcp/src/pkg/api"
)

func main() {
	fmt.Fprintln(os.Stderr, "MCP Bridge Service initializing...")

	// Handle graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Initialize components
	buf := store.NewRingBuffer(2000)
	h := hub.NewHub()
	go h.Run()

	// MCP JSON-RPC server
	mcpHandler := mcp.NewHandler(buf)
	mcpServer := mcp.NewServer(mcpHandler)

	// Check for stdio transport mode
	if len(os.Args) > 1 && os.Args[1] == "--stdio" {
		fmt.Fprintln(os.Stderr, "MCP server running on stdio")
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			<-stop
			cancel()
		}()
		if err := mcpServer.ServeStdio(ctx); err != nil {
			log.Fatalf("Stdio server error: %v", err)
		}
		return
	}

	router := api.NewRouter(buf)
	router.SetHub(h)
	mux := router.SetupRoutes()
	mux.Handle("/rpc", mcpServer)

	go func() {
		<-stop
		fmt.Println("\nShutting down gracefully...")
		h.Stop()
		os.Exit(0)
	}()

	port := "8080"
	fmt.Printf("Listening on :%s (JSON-RPC at /rpc)\n", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
