package store

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/whitmo/ws-mcp/src/internal/types"
)

// FileStore is an append-only JSONL file store for event durability.
type FileStore struct {
	mu   sync.Mutex
	path string
	file *os.File
}

// NewFileStore opens (or creates) a JSONL file at the given path.
func NewFileStore(path string) (*FileStore, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return &FileStore{path: path, file: f}, nil
}

// Append writes a single event as a JSON line.
func (fs *FileStore) Append(event types.Event) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = fs.file.Write(data)
	return err
}

// ReadAll reads every event from the JSONL file.
func (fs *FileStore) ReadAll() ([]types.Event, error) {
	return ReadEventsFromFile(fs.path)
}

// ReadEventsFromFile reads events from a JSONL file path (usable without an open FileStore).
func ReadEventsFromFile(path string) ([]types.Event, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var events []types.Event
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var ev types.Event
		if err := json.Unmarshal(line, &ev); err != nil {
			continue // skip corrupt lines
		}
		events = append(events, ev)
	}
	return events, scanner.Err()
}

// Close flushes and closes the underlying file.
func (fs *FileStore) Close() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.file.Close()
}

// Rotate renames the current file to path.bak and opens a fresh file.
func (fs *FileStore) Rotate() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if err := fs.file.Close(); err != nil {
		return err
	}
	bak := fs.path + ".bak"
	if err := os.Rename(fs.path, bak); err != nil {
		return err
	}
	f, err := os.OpenFile(fs.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	fs.file = f
	return nil
}
