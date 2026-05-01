package secret

import (
	"context"
	"path/filepath"
	"testing"
)

func TestFileStoreSetAndGet(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(t.TempDir(), "state"))

	store := NewFileStore()
	ctx := context.Background()

	if err := store.Set(ctx, "openai", "secret-key"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	got, err := store.Get(ctx, "openai")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got != "secret-key" {
		t.Fatalf("Get() = %q, want %q", got, "secret-key")
	}
}
