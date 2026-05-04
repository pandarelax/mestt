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

	modelPath := filepath.Join(tempDir, "ggml-custom.bin")
	if err := os.WriteFile(modelPath, []byte("model"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	scriptPath := filepath.Join(tempDir, "whisper-cli")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\nout=''\nwhile [ $# -gt 0 ]; do\n  case \"$1\" in\n    -of) out=\"$2\"; shift 2 ;;\n    *) shift ;;\n  esac\ndone\nprintf '{\"transcription\":[{\"text\":\" hello\"},{\"text\":\" local\"}]}' > \"${out}.json\"\n"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	client := LocalClient{}
	result, err := client.Transcribe(context.Background(), LocalRequest{
		AudioPath: audioPath,
		Model:     Model{ID: ModelLargeV3TurboQ5Local, Provider: ProviderLocal, APIName: "custom"},
		Command:   scriptPath,
		ModelPath: modelPath,
	})
	if err != nil {
		t.Fatalf("Transcribe() error = %v", err)
	}
	if result.Text != "hello local" {
		t.Fatalf("Text = %q, want %q", result.Text, "hello local")
	}
}

func TestLocalClientIgnoresStderrWarnings(t *testing.T) {
	tempDir := t.TempDir()
	audioPath := filepath.Join(tempDir, "sample.wav")
	if err := os.WriteFile(audioPath, []byte("fake"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	modelPath := filepath.Join(tempDir, "ggml-custom.bin")
	if err := os.WriteFile(modelPath, []byte("model"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	scriptPath := filepath.Join(tempDir, "whisper-cli")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\nout=''\nwhile [ $# -gt 0 ]; do\n  case \"$1\" in\n    -of) out=\"$2\"; shift 2 ;;\n    *) shift ;;\n  esac\ndone\nprintf 'warning\\n' >&2\nprintf '{\"transcription\":[{\"text\":\"hello local\"}]}' > \"${out}.json\"\n"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	client := LocalClient{}
	result, err := client.Transcribe(context.Background(), LocalRequest{
		AudioPath: audioPath,
		Model:     Model{ID: ModelLargeV3TurboQ5Local, Provider: ProviderLocal, APIName: "custom"},
		Command:   scriptPath,
		ModelPath: modelPath,
	})
	if err != nil {
		t.Fatalf("Transcribe() error = %v", err)
	}
	if result.Text != "hello local" {
		t.Fatalf("Text = %q, want %q", result.Text, "hello local")
	}
}

func TestLocalClientDownloadsDefaultModel(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(t.TempDir(), "state"))

	tempDir := t.TempDir()
	audioPath := filepath.Join(tempDir, "sample.wav")
	if err := os.WriteFile(audioPath, []byte("fake"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	downloaderPath := filepath.Join(tempDir, "whisper-cpp-download-ggml-model")
	if err := os.WriteFile(downloaderPath, []byte("#!/bin/sh\nmodel=\"$1\"\nout=\"$2\"\nmkdir -p \"$out\"\nprintf 'model' > \"$out/ggml-${model}.bin\"\n"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	commandPath := filepath.Join(tempDir, "whisper-cli")
	if err := os.WriteFile(commandPath, []byte("#!/bin/sh\nout=''\nmodel=''\nwhile [ $# -gt 0 ]; do\n  case \"$1\" in\n    -of) out=\"$2\"; shift 2 ;;\n    -m) model=\"$2\"; shift 2 ;;\n    *) shift ;;\n  esac\ndone\n[ -f \"$model\" ] || exit 1\nprintf '{\"transcription\":[{\"text\":\"downloaded\"}]}' > \"${out}.json\"\n"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	client := LocalClient{}
	result, err := client.Transcribe(context.Background(), LocalRequest{
		AudioPath:       audioPath,
		Model:           Model{ID: ModelLargeV3TurboQ5Local, Provider: ProviderLocal, APIName: "custom"},
		Command:         commandPath,
		DownloadCommand: downloaderPath,
	})
	if err != nil {
		t.Fatalf("Transcribe() error = %v", err)
	}
	if result.Text != "downloaded" {
		t.Fatalf("Text = %q, want %q", result.Text, "downloaded")
	}
}

func TestValidateKnownModelFileRejectsCorruptFile(t *testing.T) {
	tempDir := t.TempDir()
	modelPath := filepath.Join(tempDir, "ggml-large-v3-turbo-q5_0.bin")
	if err := os.WriteFile(modelPath, []byte("bad"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	err := validateModelFile(modelPath, knownModel{Size: 5, SHA256: "deadbeef"})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestLocalClientPrepareReportsMissingWhisperCLI(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	client := LocalClient{}
	err := client.Prepare(context.Background(), LocalRequest{
		Model: Model{ID: ModelLargeV3TurboQ5Local, Provider: ProviderLocal, APIName: "custom"},
	})
	if err == nil {
		t.Fatal("expected prepare error")
	}
	if got := err.Error(); got != "whisper-cli not found in PATH; run inside 'nix develop' or set [local].command to an absolute command path" {
		t.Fatalf("Prepare() error = %q", got)
	}
}
