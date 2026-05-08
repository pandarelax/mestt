package daemon

import (
	"context"
	"strings"
	"testing"
)

func TestUnsupportedHotkeyBackendReturnsReason(t *testing.T) {
	err := UnsupportedHotkeyBackend{Reason: "linux backend not implemented"}.Start(context.Background(), func() {})
	if err == nil {
		t.Fatal("Start() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "linux backend not implemented") {
		t.Fatalf("Start() error = %q, want reason included", err.Error())
	}
}
