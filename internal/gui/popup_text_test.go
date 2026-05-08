package gui

import "testing"

func TestCompactStatusTextShowsRecordingDuration(t *testing.T) {
	got := compactStatusText(
		GUIState{Status: string(StatusRecording)},
		LevelState{DurationSeconds: 75},
	)
	if got != "Listening  01:15" {
		t.Fatalf("compactStatusText() = %q, want %q", got, "Listening  01:15")
	}
}

func TestCompactStatusTextPrefersErrors(t *testing.T) {
	got := compactStatusText(
		GUIState{Status: string(StatusError), Error: "clipboard failed"},
		LevelState{},
	)
	if got != "clipboard failed" {
		t.Fatalf("compactStatusText() = %q, want %q", got, "clipboard failed")
	}
}
