package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCreatesDefaultConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(t.TempDir(), "state"))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Transcription.Model != "gpt-4o-mini-transcribe" {
		t.Fatalf("unexpected default model %q", cfg.Transcription.Model)
	}
	if cfg.Local.Command != "whisper-cli" {
		t.Fatalf("unexpected default local command %q", cfg.Local.Command)
	}
	if !cfg.Local.UseGPU {
		t.Fatal("expected local GPU usage to default to true")
	}

	configFile := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "mestt", "config.toml")
	if _, err := os.Stat(configFile); err != nil {
		t.Fatalf("config file not created: %v", err)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(t.TempDir(), "state"))

	want := Default()
	want.Transcription.Model = "whisper-1"

	if err := Save(want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.Transcription.Model != want.Transcription.Model {
		t.Fatalf("model = %q, want %q", got.Transcription.Model, want.Transcription.Model)
	}
}

func TestLoadPreservesExplicitFullLocalModelSelection(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(t.TempDir(), "state"))

	cfg := Default()
	cfg.Transcription.Provider = "local"
	cfg.Transcription.Model = "large-v3-turbo"
	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Transcription.Model != "large-v3-turbo" {
		t.Fatalf("model = %q, want %q", got.Transcription.Model, "large-v3-turbo")
	}
}
