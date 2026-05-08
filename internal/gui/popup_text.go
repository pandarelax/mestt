package gui

import "fmt"

func compactStatusText(state GUIState, levels LevelState) string {
	if state.Error != "" {
		return state.Error
	}

	switch state.Status {
	case string(StatusPreparing):
		return "Preparing..."
	case string(StatusRecording):
		return fmt.Sprintf("Listening  %s", formatDuration(levels.DurationSeconds))
	case string(StatusTranscribing):
		return "Transcribing..."
	case string(StatusCopied):
		return "Copied"
	default:
		return "Ready"
	}
}

func formatDuration(seconds int) string {
	if seconds < 0 {
		seconds = 0
	}
	return fmt.Sprintf("%02d:%02d", seconds/60, seconds%60)
}
