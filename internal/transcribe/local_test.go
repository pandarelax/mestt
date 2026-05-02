package transcribe

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalClientTranscribe(t *testing.T) {
	tempDir := t.TempDir()
	audioPath := filepath.Join(tempDir, "sample.wav")
	if err := os.WriteFile(audioPath, []byte("fake"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	scriptPath := filepath.Join(tempDir, "helper.py")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\nprintf '{\"text\":\"hello local\"}\\n'\n"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	client := LocalClient{}
	result, err := client.Transcribe(context.Background(), LocalRequest{
		AudioPath:     audioPath,
		Model:         Model{ID: ModelLargeV3TurboLocal, Provider: ProviderLocal, APIName: "large-v3-turbo"},
		PythonCommand: "sh",
		ScriptPath:    scriptPath,
	})
	if err != nil {
		t.Fatalf("Transcribe() error = %v", err)
	}
	if result.Text != "hello local" {
		t.Fatalf("Text = %q, want %q", result.Text, "hello local")
	}
}
