package record

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestFormatDuration(t *testing.T) {
	if got := formatDuration(65 * time.Second); got != "01:05" {
		t.Fatalf("formatDuration() = %q, want %q", got, "01:05")
	}
}

func TestRenderVisualizationHasExpectedDimensions(t *testing.T) {
	history := []float64{10, 20, 40, 60, 35, 25, 50, 70}
	view := renderVisualization(history, 12, 4, 3)
	lines := strings.Split(strings.TrimSuffix(view, "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("len(lines) = %d, want 4", len(lines))
	}
	for i, line := range lines {
		if len(line) != 12 {
			t.Fatalf("line %d length = %d, want 12", i, len(line))
		}
	}
}

func TestPrepareModelCancelsActivePreflight(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	m := prepareModel{ctx: ctx, cancel: cancel, active: true, status: "preparing"}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	next := updated.(prepareModel)
	if !errors.Is(next.err, ErrCanceled) {
		t.Fatalf("err = %v, want %v", next.err, ErrCanceled)
	}
	if next.ctx.Err() == nil {
		t.Fatal("expected context cancellation")
	}
	if cmd == nil {
		t.Fatal("expected quit command")
	}
}

func TestRecordModelKeepsTranscriptionErrorsVisible(t *testing.T) {
	m := model{closing: true, status: "transcribing"}
	updated, cmd := m.Update(transcribeMsg{err: errors.New("boom")})
	next := updated.(model)
	if next.status != "error" {
		t.Fatalf("status = %q, want %q", next.status, "error")
	}
	if next.err == nil {
		t.Fatal("expected transcription error")
	}
	if next.quitting {
		t.Fatal("expected model to stay open")
	}
	if cmd != nil {
		t.Fatal("expected no quit command")
	}
}
