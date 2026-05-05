package gui

type Status string

const (
	StatusIdle         Status = "idle"
	StatusPreparing    Status = "preparing"
	StatusRecording    Status = "recording"
	StatusTranscribing Status = "transcribing"
	StatusCopied       Status = "copied"
	StatusError        Status = "error"
)

type GUIState struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Error   string `json:"error"`
	Text    string `json:"text"`
	Ready   bool   `json:"ready"`
}

type LevelState struct {
	Level           float64 `json:"level"`
	Peak            float64 `json:"peak"`
	DurationSeconds int     `json:"durationSeconds"`
}
