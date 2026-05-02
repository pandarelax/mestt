package transcribe

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type LocalRequest struct {
	AudioPath     string
	Model         Model
	PythonCommand string
	Device        string
	ComputeType   string
	Language      string
	ScriptPath    string
}

type LocalClient struct{}

type localResponse struct {
	Text string `json:"text"`
}

func (c LocalClient) Transcribe(ctx context.Context, req LocalRequest) (Result, error) {
	pythonCommand := strings.TrimSpace(req.PythonCommand)
	if pythonCommand == "" {
		pythonCommand = "python3"
	}

	scriptPath := req.ScriptPath
	if scriptPath == "" {
		scriptPath = helperScriptPath()
	}

	args := []string{
		scriptPath,
		"--audio", req.AudioPath,
		"--model", req.Model.APIName,
		"--device", defaultString(req.Device, "cuda"),
		"--compute-type", defaultString(req.ComputeType, "float16"),
	}
	if strings.TrimSpace(req.Language) != "" {
		args = append(args, "--language", req.Language)
	}

	cmd := exec.CommandContext(ctx, pythonCommand, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return Result{}, fmt.Errorf("run local faster-whisper helper: %w: %s", err, strings.TrimSpace(string(output)))
	}

	var parsed localResponse
	if err := json.Unmarshal(output, &parsed); err != nil {
		return Result{}, fmt.Errorf("parse local transcription response: %w", err)
	}

	return Result{Text: parsed.Text, ModelID: string(req.Model.ID), Provider: string(req.Model.Provider)}, nil
}

func helperScriptPath() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.Join("scripts", "faster_whisper_transcribe.py")
	}
	return filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(file))), "scripts", "faster_whisper_transcribe.py")
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
