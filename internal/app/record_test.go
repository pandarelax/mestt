package app

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"pandarelax/mestt/internal/audio"
	"pandarelax/mestt/internal/config"
	"pandarelax/mestt/internal/output"
	"pandarelax/mestt/internal/transcribe"
	"pandarelax/mestt/internal/tui/record"
)

type fakeRecorder struct{ called bool }

func (f *fakeRecorder) Start(context.Context, audio.RecordOptions) (audio.SessionHandle, error) {
	f.called = true
	return fakeSession{}, nil
}

type fakeSession struct{}

func (fakeSession) Stop(context.Context) (audio.Recording, error) {
	return audio.Recording{Path: filepath.Join("/tmp", "recording.wav"), CreatedAt: time.Now()}, nil
}

func (fakeSession) Cancel() error                     { return nil }
func (fakeSession) Duration() time.Duration           { return 0 }
func (fakeSession) Levels() (float64, float64, error) { return 0, 0, nil }

type fakeRecordingRunner struct{}

func (fakeRecordingRunner) RunPrepare(_ record.PrepareOptions, action func(context.Context) error) error {
	if action == nil {
		return nil
	}
	return action(context.Background())
}

func (fakeRecordingRunner) Run(_ audio.SessionHandle, _ output.Target, opts record.Options) error {
	return opts.Submit(audio.Recording{Path: filepath.Join("/tmp", "recording.wav"), CreatedAt: time.Now()})
}

type fakeLocalPreparer struct{ called bool }

func (f *fakeLocalPreparer) PrepareLocal(context.Context) error {
	f.called = true
	return nil
}

type fakeLocalPreparerError struct{ called bool }

func (f *fakeLocalPreparerError) PrepareLocal(context.Context) error {
	f.called = true
	return context.DeadlineExceeded
}

type fakeLocalPreparerCanceled struct{ ctx context.Context }

func (f *fakeLocalPreparerCanceled) PrepareLocal(ctx context.Context) error {
	f.ctx = ctx
	return ctx.Err()
}

type fakeCancelPrepareRunner struct{}

func (fakeCancelPrepareRunner) RunPrepare(_ record.PrepareOptions, action func(context.Context) error) error {
	if action == nil {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return action(ctx)
}

func (fakeCancelPrepareRunner) Run(_ audio.SessionHandle, _ output.Target, _ record.Options) error {
	return nil
}

type fakeTranscribeRunner struct{ called bool }

func (f *fakeTranscribeRunner) Run(context.Context, TranscribeInput) (transcribe.Result, error) {
	f.called = true
	return transcribe.Result{Text: "hello", ModelID: "large-v3-turbo-q5_0", Provider: "local"}, nil
}

func TestRecordServicePreparesLocalModelBeforeRecording(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(t.TempDir(), "state"))

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	cfg.Transcription.Provider = "local"
	cfg.Transcription.Model = "large-v3-turbo-q5_0"
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	preparer := &fakeLocalPreparer{}
	transcriber := &fakeTranscribeRunner{}
	recorder := &fakeRecorder{}
	service := RecordService{
		Recorder:   recorder,
		RunUI:      fakeRecordingRunner{},
		Transcribe: transcriber,
		Prepare:    preparer,
	}

	if err := service.Run(context.Background(), RecordInput{Target: output.Target{Kind: output.TargetStdout}}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !preparer.called {
		t.Fatal("expected PrepareLocal to be called")
	}
	if !transcriber.called {
		t.Fatal("expected transcription to be called")
	}
	if !recorder.called {
		t.Fatal("expected recorder to start")
	}
}

func TestRecordServiceStopsBeforeRecordingWhenPreflightFails(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(t.TempDir(), "state"))

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	cfg.Transcription.Provider = "local"
	cfg.Transcription.Model = "large-v3-turbo-q5_0"
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	preparer := &fakeLocalPreparerError{}
	transcriber := &fakeTranscribeRunner{}
	recorder := &fakeRecorder{}
	service := RecordService{
		Recorder:   recorder,
		RunUI:      fakeRecordingRunner{},
		Transcribe: transcriber,
		Prepare:    preparer,
	}

	err = service.Run(context.Background(), RecordInput{Target: output.Target{Kind: output.TargetStdout}})
	if err == nil {
		t.Fatal("expected Run() error")
	}
	if !preparer.called {
		t.Fatal("expected PrepareLocal to be called")
	}
	if recorder.called {
		t.Fatal("expected recorder not to start")
	}
	if transcriber.called {
		t.Fatal("expected transcription not to be called")
	}
}

func TestRecordServiceReturnsNilWhenPreflightIsCanceled(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(t.TempDir(), "state"))

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	cfg.Transcription.Provider = "local"
	cfg.Transcription.Model = "large-v3-turbo-q5_0"
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	preparer := &fakeLocalPreparerCanceled{}
	recorder := &fakeRecorder{}
	service := RecordService{
		Recorder:   recorder,
		RunUI:      fakeCancelPrepareRunner{},
		Transcribe: &fakeTranscribeRunner{},
		Prepare:    preparer,
	}

	if err := service.Run(context.Background(), RecordInput{Target: output.Target{Kind: output.TargetStdout}}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if preparer.ctx == nil {
		t.Fatal("expected PrepareLocal context")
	}
	if preparer.ctx.Err() == nil {
		t.Fatal("expected canceled prepare context")
	}
	if recorder.called {
		t.Fatal("expected recorder not to start")
	}
}
