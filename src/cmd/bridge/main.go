package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/whitmo/ws-mcp/src/internal/hub"
	"github.com/whitmo/ws-mcp/src/internal/mcp"
	"github.com/whitmo/ws-mcp/src/internal/store"
	"github.com/whitmo/ws-mcp/src/pkg/api"
)

func main() {
	stdio := flag.Bool("stdio", false, "Run MCP server on stdio transport")
	dataDir := flag.String("data-dir", ".bridge", "Directory for persistent data files")
	flag.Parse()

	fmt.Fprintln(os.Stderr, "MCP Bridge Service initializing...")

	// Handle graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	const bufCap = 2000
	buf := store.NewRingBuffer(bufCap)

	// Open durable file store
	eventsPath := filepath.Join(*dataDir, "events.jsonl")
	fileStore, err := store.NewFileStore(eventsPath)
	if err != nil {
		log.Fatalf("Failed to open file store at %s: %v", eventsPath, err)
	}

	// Replay persisted events into ring buffer (most recent bufCap)
	if persisted, err := store.ReadEventsFromFile(eventsPath); err == nil && len(persisted) > 0 {
		start := 0
		if len(persisted) > bufCap {
			start = len(persisted) - bufCap
		}
		for _, ev := range persisted[start:] {
			buf.Push(ev)
		}
		fmt.Fprintf(os.Stderr, "Replayed %d events from %s\n", len(persisted)-start, eventsPath)
	}

	h := hub.NewHub()
	go h.Run()

	// MCP JSON-RPC server
	mcpHandler := mcp.NewHandler(buf)
	mcpServer := mcp.NewServer(mcpHandler)

	if *stdio {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			<-stop
			fileStore.Close()
			cancel()
		}()

		// Spoke mode: try to proxy through a running hub
		hubURL := "http://localhost:8080/rpc"
		if env := os.Getenv("WS_MCP_HUB"); env != "" {
			hubURL = env
		}
		proxy := mcp.NewProxyClient(hubURL)
		if proxy.Ping(400 * time.Millisecond) {
			fmt.Fprintln(os.Stderr, "Hub detected at", hubURL, "— running as spoke")
			if err := mcp.ServeSpoke(ctx, proxy, os.Stdin, os.Stdout); err != nil {
				log.Fatalf("Spoke error: %v", err)
			}
			return
		}

		// No hub — standalone mode with local ring buffer
		fmt.Fprintln(os.Stderr, "No hub detected — running standalone on stdio")
		if err := mcpServer.ServeStdio(ctx); err != nil {
			log.Fatalf("Stdio server error: %v", err)
		}
		return
	}

	router := api.NewRouter(buf)
	router.SetHub(h)
	router.SetFileStore(fileStore)
	mux := router.SetupRoutes()
	mux.Handle("/rpc", mcpServer)

	go func() {
		<-stop
		fmt.Fprintln(os.Stderr, "\nShutting down gracefully...")
		fileStore.Close()
		h.Stop()
		os.Exit(0)
	}()

	port := "8080"
	fmt.Fprintf(os.Stderr, "Listening on :%s (JSON-RPC at /rpc)\n", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
