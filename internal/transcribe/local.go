package transcribe

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

//go:embed faster_whisper_helper.py
var embeddedHelperScript string

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

	args := []string{
		"--audio", req.AudioPath,
		"--model", req.Model.APIName,
		"--device", defaultString(req.Device, "cpu"),
		"--compute-type", defaultString(req.ComputeType, "int8"),
	}
	if strings.TrimSpace(req.Language) != "" {
		args = append(args, "--language", req.Language)
	}

	cmdArgs := args
	if strings.TrimSpace(req.ScriptPath) != "" {
		cmdArgs = append([]string{req.ScriptPath}, args...)
	} else {
		cmdArgs = append([]string{"-c", embeddedHelperScript}, args...)
	}

	cmd := exec.CommandContext(ctx, pythonCommand, cmdArgs...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		stderrText := strings.TrimSpace(stderr.String())
		if stderrText != "" {
			return Result{}, fmt.Errorf("run local faster-whisper helper: %w: %s", err, stderrText)
		}
		return Result{}, fmt.Errorf("run local faster-whisper helper: %w", err)
	}

	var parsed localResponse
	if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
		stderrText := strings.TrimSpace(stderr.String())
		if stderrText != "" {
			return Result{}, fmt.Errorf("parse local transcription response: %w: %s", err, stderrText)
		}
		return Result{}, fmt.Errorf("parse local transcription response: %w", err)
	}

	return Result{Text: parsed.Text, ModelID: string(req.Model.ID), Provider: string(req.Model.Provider)}, nil
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
