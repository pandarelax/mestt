package app

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"pandarelax/mestt/internal/config"
	"pandarelax/mestt/internal/history"
	"pandarelax/mestt/internal/output"
	"pandarelax/mestt/internal/transcribe"
)

type fakeSecretStore struct {
	values map[string]string
}

func (f fakeSecretStore) Get(_ context.Context, key string) (string, error) {
	return f.values[key], nil
}

func (f fakeSecretStore) Set(_ context.Context, key, value string) error {
	if f.values == nil {
		f.values = map[string]string{}
	}
	f.values[key] = value
	return nil
}

type fakeHistorySaver struct {
	entries []history.Entry
}

func (f *fakeHistorySaver) Save(_ context.Context, entry history.Entry) error {
	f.entries = append(f.entries, entry)
	return nil
}

type fakeOutputWriter struct{}

func (fakeOutputWriter) Write(_ context.Context, _ string, _ output.Target) error {
	return nil
}

type fakeOpenAIClient struct{}

func (fakeOpenAIClient) Transcribe(_ context.Context, _ transcribe.Request) (transcribe.Result, error) {
	return transcribe.Result{Text: "hello", ModelID: "whisper-1", Provider: "openai"}, nil
}

type fakeLocalClient struct{}

func (fakeLocalClient) Transcribe(_ context.Context, _ transcribe.LocalRequest) (transcribe.Result, error) {
	return transcribe.Result{Text: "hello", ModelID: "large-v3-turbo", Provider: "local"}, nil
}

func TestTranscribeServiceRejectsProviderModelMismatch(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(t.TempDir(), "state"))

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	cfg.Transcription.Provider = "local"
	cfg.Transcription.Model = "whisper-1"
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	service := TranscribeService{
		Secrets: fakeSecretStore{values: map[string]string{"openai": "secret"}},
		History: &fakeHistorySaver{},
		Output:  fakeOutputWriter{},
		Client:  fakeOpenAIClient{},
		Local:   fakeLocalClient{},
	}

	_, err = service.Run(context.Background(), TranscribeInput{AudioPath: "sample.wav", Target: output.Target{Kind: output.TargetStdout}})
	if err == nil {
		t.Fatal("expected provider/model mismatch error")
	}
}

func TestTranscribeServiceOmitsRecordingSourcePath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(t.TempDir(), "state"))

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	cfg.Transcription.Provider = "local"
	cfg.Transcription.Model = "large-v3-turbo"
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	historySaver := &fakeHistorySaver{}
	service := TranscribeService{
		Secrets: fakeSecretStore{},
		History: historySaver,
		Output:  fakeOutputWriter{},
		Client:  fakeOpenAIClient{},
		Local:   fakeLocalClient{},
	}

	_, err = service.Run(context.Background(), TranscribeInput{
		AudioPath:  "temp.wav",
		Target:     output.Target{Kind: output.TargetStdout},
		SourceKind: "recording",
		SourcePath: "",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(historySaver.entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(historySaver.entries))
	}
	if historySaver.entries[0].SourcePath != "" {
		t.Fatalf("SourcePath = %q, want empty", historySaver.entries[0].SourcePath)
	}
	if time.Since(historySaver.entries[0].CreatedAt) > time.Minute {
		t.Fatalf("CreatedAt looks wrong: %v", historySaver.entries[0].CreatedAt)
	}
}
