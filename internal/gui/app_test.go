package gui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	appsvc "pandarelax/mestt/internal/app"
	"pandarelax/mestt/internal/audio"
	"pandarelax/mestt/internal/output"
	"pandarelax/mestt/internal/transcribe"
)

type fakeRecorder struct {
	session audio.SessionHandle
	err     error
	called  bool
}

func (f *fakeRecorder) Start(context.Context, audio.RecordOptions) (audio.SessionHandle, error) {
	f.called = true
	if f.err != nil {
		return nil, f.err
	}
	return f.session, nil
}

type fakeSession struct {
	stopRecording audio.Recording
	stopErr       error
	cancelErr     error
	level         float64
	peak          float64
	duration      time.Duration
	canceled      bool
}

func (f *fakeSession) Stop(context.Context) (audio.Recording, error) {
	return f.stopRecording, f.stopErr
}

func (f *fakeSession) Cancel() error {
	f.canceled = true
	return f.cancelErr
}

func (f *fakeSession) Duration() time.Duration { return f.duration }

func (f *fakeSession) Levels() (float64, float64, error) {
	return f.level, f.peak, nil
}

type fakeTranscriber struct {
	result transcribe.Result
	err    error
	called bool
	input  appsvc.TranscribeInput
}

func (f *fakeTranscriber) Run(_ context.Context, input appsvc.TranscribeInput) (transcribe.Result, error) {
	f.called = true
	f.input = input
	return f.result, f.err
}

type fakePreparer struct {
	err error
	ctx context.Context
	fn  func(context.Context) error
}

func (f *fakePreparer) PrepareLocal(ctx context.Context) error {
	f.ctx = ctx
	if f.fn != nil {
		return f.fn(ctx)
	}
	return f.err
}

func TestStartRecordingTransitionsToRecording(t *testing.T) {
	app := &App{
		state:      GUIState{Status: string(StatusIdle), Message: "Press Enter to record", Ready: true},
		recorder:   &fakeRecorder{session: &fakeSession{}},
		transcribe: &fakeTranscriber{},
		prepare:    &fakePreparer{},
	}

	state := app.StartRecording()
	if state.Status != string(StatusPreparing) {
		t.Fatalf("StartRecording() status = %q, want %q", state.Status, StatusPreparing)
	}

	waitForStatus(t, app, StatusRecording)
	if app.session == nil {
		t.Fatal("expected active session")
	}
}

func TestCancelRecordingCancelsPrepareAndReturnsIdle(t *testing.T) {
	preparer := &fakePreparer{fn: func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}}
	app := &App{
		state:      GUIState{Status: string(StatusIdle), Message: "Press Enter to record", Ready: true},
		recorder:   &fakeRecorder{session: &fakeSession{}},
		transcribe: &fakeTranscriber{},
		prepare:    preparer,
	}

	app.StartRecording()
	waitForStatus(t, app, StatusPreparing)
	state := app.CancelRecording()
	if state.Status != string(StatusIdle) {
		t.Fatalf("CancelRecording() status = %q, want %q", state.Status, StatusIdle)
	}
	waitForCondition(t, func() bool { return preparer.ctx != nil }, "prepare context")
	waitForStatus(t, app, StatusIdle)
	if preparer.ctx.Err() == nil {
		t.Fatal("expected canceled prepare context")
	}
}

func TestStopAndTranscribeTransitionsToCopied(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "recording.wav")
	if err := os.WriteFile(path, []byte("fake"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	session := &fakeSession{stopRecording: audio.Recording{Path: path, CreatedAt: time.Now()}}
	transcriber := &fakeTranscriber{result: transcribe.Result{Text: "hello world"}}
	app := &App{
		state:      GUIState{Status: string(StatusRecording), Message: "Recording...", Ready: true},
		session:    session,
		transcribe: transcriber,
	}

	state := app.StopAndTranscribe()
	if state.Status != string(StatusTranscribing) {
		t.Fatalf("StopAndTranscribe() status = %q, want %q", state.Status, StatusTranscribing)
	}

	waitForStatus(t, app, StatusCopied)
	final := app.Status()
	if final.Text != "hello world" {
		t.Fatalf("Text = %q, want %q", final.Text, "hello world")
	}
	if !transcriber.called {
		t.Fatal("expected transcriber to be called")
	}
	if transcriber.input.Target.Kind != output.TargetClipboard {
		t.Fatalf("Target.Kind = %q, want %q", transcriber.input.Target.Kind, output.TargetClipboard)
	}
}

func TestStopAndTranscribeErrorBecomesVisible(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "recording.wav")
	if err := os.WriteFile(path, []byte("fake"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	session := &fakeSession{stopRecording: audio.Recording{Path: path, CreatedAt: time.Now()}}
	app := &App{
		state:      GUIState{Status: string(StatusRecording), Message: "Recording...", Ready: true},
		session:    session,
		transcribe: &fakeTranscriber{err: errors.New("clipboard failed")},
	}

	app.StopAndTranscribe()
	waitForStatus(t, app, StatusError)
	if app.Status().Error == "" {
		t.Fatal("expected visible error")
	}
}

func waitForStatus(t *testing.T, app *App, want Status) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if got := app.Status().Status; got == string(want) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for status %q; last status %q", want, app.Status().Status)
}

func waitForCondition(t *testing.T, check func() bool, description string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", description)
}
