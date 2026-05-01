package history

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreSaveAndList(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(t.TempDir(), "state"))

	store, err := Open()
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	if err := store.Save(ctx, Entry{
		CreatedAt:  time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
		SourceKind: "file",
		SourcePath: "sample.wav",
		ModelID:    "whisper-1",
		Transcript: "hello world",
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	entries, err := store.List(ctx, 10)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
	if entries[0].Transcript != "hello world" {
		t.Fatalf("Transcript = %q, want %q", entries[0].Transcript, "hello world")
	}
}
