package record

import (
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	if got := formatDuration(65 * time.Second); got != "01:05" {
		t.Fatalf("formatDuration() = %q, want %q", got, "01:05")
	}
}
