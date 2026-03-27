package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/whitmo/ws-mcp/src/internal/types"
)

func makeEvent(id string) types.Event {
	return types.Event{
		ID:     id,
		Source: types.SourceSystem,
		Type:   "test",
		Ts:     time.Now(),
		Payload: map[string]any{
			"msg": "hello",
		},
	}
}

func TestFileStore_AppendAndReadBack(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")

	fs, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	defer fs.Close()

	for i := 0; i < 5; i++ {
		if err := fs.Append(makeEvent("evt-" + string(rune('a'+i)))); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	events, err := fs.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(events) != 5 {
		t.Fatalf("expected 5 events, got %d", len(events))
	}
	if events[0].ID != "evt-a" {
		t.Fatalf("expected first event id evt-a, got %s", events[0].ID)
	}
}

func TestFileStore_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "deep", "events.jsonl")

	fs, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	fs.Close()

	if _, err := os.Stat(filepath.Dir(path)); os.IsNotExist(err) {
		t.Fatal("expected directory to be created")
	}
}

func TestFileStore_Rotate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")

	fs, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	// Write some events before rotation
	for i := 0; i < 3; i++ {
		fs.Append(makeEvent("pre-rotate"))
	}

	if err := fs.Rotate(); err != nil {
		t.Fatalf("Rotate: %v", err)
	}

	// Old events should be in .bak
	bakEvents, err := ReadEventsFromFile(path + ".bak")
	if err != nil {
		t.Fatalf("read bak: %v", err)
	}
	if len(bakEvents) != 3 {
		t.Fatalf("expected 3 events in bak, got %d", len(bakEvents))
	}

	// New file should be empty
	fs.Append(makeEvent("post-rotate"))
	events, err := fs.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event after rotation, got %d", len(events))
	}

	fs.Close()
}

func TestReadEventsFromFile_MissingFile(t *testing.T) {
	_, err := ReadEventsFromFile("/nonexistent/path.jsonl")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestReadEventsFromFile_CorruptLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")

	// Write a mix of valid and invalid lines
	f, _ := os.Create(path)
	f.WriteString(`{"id":"good","source":"system","type":"test","ts":"2025-01-01T00:00:00Z"}` + "\n")
	f.WriteString("not json\n")
	f.WriteString(`{"id":"also-good","source":"system","type":"test","ts":"2025-01-01T00:00:00Z"}` + "\n")
	f.Close()

	events, err := ReadEventsFromFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 valid events, got %d", len(events))
	}
}
