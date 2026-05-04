package gui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	appsvc "pandarelax/mestt/internal/app"
	"pandarelax/mestt/internal/audio"
	"pandarelax/mestt/internal/config"
	"pandarelax/mestt/internal/history"
	"pandarelax/mestt/internal/output"
	"pandarelax/mestt/internal/secret"
	"pandarelax/mestt/internal/transcribe"
)

type recorder interface {
	Start(ctx context.Context, opts audio.RecordOptions) (audio.SessionHandle, error)
}

type transcribeRunner interface {
	Run(ctx context.Context, input appsvc.TranscribeInput) (transcribe.Result, error)
}

type localPreparer interface {
	PrepareLocal(ctx context.Context) error
}

type App struct {
	mu              sync.RWMutex
	ctx             context.Context
	state           GUIState
	level           LevelState
	recorder        recorder
	transcribe      transcribeRunner
	prepare         localPreparer
	historyStore    *history.Store
	session         audio.SessionHandle
	recordingCancel context.CancelFunc
	operationCtx    context.Context
	operationCancel context.CancelFunc
}

func NewApp() (*App, error) {
	historyStore, err := history.Open()
	if err != nil {
		return nil, err
	}

	service := appsvc.TranscribeService{
		Secrets: secret.NewFileStore(),
		History: historyStore,
		Output:  output.Writer{},
		Client:  transcribe.OpenAIClient{},
		Local:   transcribe.LocalClient{},
	}

	return &App{
		state: GUIState{
			Status:  string(StatusIdle),
			Message: "Press Enter to record",
			Ready:   false,
		},
		recorder:     audio.NewRecorder(),
		transcribe:   service,
		prepare:      service,
		historyStore: historyStore,
	}, nil
}

func (a *App) Startup(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.ctx = ctx
	a.state.Ready = true
}

func (a *App) Shutdown(context.Context) {
	a.mu.Lock()
	session := a.session
	recordingCancel := a.recordingCancel
	operationCancel := a.operationCancel
	historyStore := a.historyStore
	a.session = nil
	a.recordingCancel = nil
	a.operationCtx = nil
	a.operationCancel = nil
	a.mu.Unlock()

	if operationCancel != nil {
		operationCancel()
	}
	if recordingCancel != nil {
		recordingCancel()
	}
	if session != nil {
		_ = session.Cancel()
	}
	if historyStore != nil {
		_ = historyStore.Close()
	}
}

func (a *App) Status() GUIState {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.state
}

func (a *App) Levels() LevelState {
	a.mu.Lock()
	session := a.session
	a.mu.Unlock()
	if session == nil {
		return LevelState{}
	}

	level, peak, err := session.Levels()
	if err != nil {
		a.mu.RLock()
		defer a.mu.RUnlock()
		return a.level
	}

	next := LevelState{
		Level:           level,
		Peak:            peak,
		DurationSeconds: int(session.Duration().Seconds()),
	}

	a.mu.Lock()
	a.level = next
	a.mu.Unlock()
	return next
}

func (a *App) StartRecording() GUIState {
	a.mu.Lock()
	if a.session != nil || a.operationCancel != nil {
		state := a.state
		a.mu.Unlock()
		return state
	}

	baseCtx := context.Background()
	if a.ctx != nil {
		baseCtx = a.ctx
	}
	ctx, cancel := context.WithCancel(baseCtx)
	a.operationCancel = cancel
	a.operationCtx = ctx
	a.level = LevelState{}
	state := a.setStateLocked(StatusPreparing, "Preparing local model...", "", "")
	a.mu.Unlock()

	go a.prepareAndStart(ctx)
	return state
}

func (a *App) StopAndTranscribe() GUIState {
	a.mu.Lock()
	if a.session == nil || a.operationCancel != nil {
		state := a.state
		a.mu.Unlock()
		return state
	}

	session := a.session
	a.session = nil
	a.recordingCancel = nil
	baseCtx := context.Background()
	if a.ctx != nil {
		baseCtx = a.ctx
	}
	ctx, cancel := context.WithCancel(baseCtx)
	a.operationCancel = cancel
	a.operationCtx = ctx
	state := a.setStateLocked(StatusTranscribing, "Transcribing...", "", "")
	a.mu.Unlock()

	go a.stopAndTranscribe(ctx, session)
	return state
}

func (a *App) CancelRecording() GUIState {
	a.mu.Lock()
	session := a.session
	recordingCancel := a.recordingCancel
	operationCancel := a.operationCancel
	a.session = nil
	a.recordingCancel = nil
	a.operationCtx = nil
	a.operationCancel = nil
	state := a.setStateLocked(StatusIdle, "Press Enter to record", "", "")
	a.level = LevelState{}
	a.mu.Unlock()

	if operationCancel != nil {
		operationCancel()
	}
	if recordingCancel != nil {
		recordingCancel()
	}
	if session != nil {
		_ = session.Cancel()
	}
	return state
}

func (a *App) Prepare() GUIState {
	a.mu.Lock()
	if a.session != nil || a.operationCancel != nil {
		state := a.state
		a.mu.Unlock()
		return state
	}

	baseCtx := context.Background()
	if a.ctx != nil {
		baseCtx = a.ctx
	}
	ctx, cancel := context.WithCancel(baseCtx)
	a.operationCancel = cancel
	a.operationCtx = ctx
	state := a.setStateLocked(StatusPreparing, "Preparing local model...", "", "")
	a.mu.Unlock()

	go a.prepareOnly(ctx)
	return state
}

func (a *App) DismissError() GUIState {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.state.Status == string(StatusError) || a.state.Status == string(StatusCopied) {
		a.level = LevelState{}
		return a.setStateLocked(StatusIdle, "Press Enter to record", "", "")
	}
	return a.state
}

func (a *App) prepareOnly(ctx context.Context) {
	err := a.prepare.PrepareLocal(ctx)
	if errors.Is(err, context.Canceled) {
		a.finishOperationWithState(ctx, func() {
			a.level = LevelState{}
			a.setStateLocked(StatusIdle, "Press Enter to record", "", "")
		})
		return
	}
	if err != nil {
		a.finishOperationWithState(ctx, func() {
			a.level = LevelState{}
			a.setStateLocked(StatusError, "Something went wrong", err.Error(), "")
		})
		return
	}
	a.finishOperationWithState(ctx, func() {
		a.level = LevelState{}
		a.setStateLocked(StatusIdle, "Press Enter to record", "", "")
	})
}

func (a *App) prepareAndStart(ctx context.Context) {
	err := a.prepare.PrepareLocal(ctx)
	if errors.Is(err, context.Canceled) {
		a.finishOperationWithState(ctx, func() {
			a.level = LevelState{}
			a.setStateLocked(StatusIdle, "Press Enter to record", "", "")
		})
		return
	}
	if err != nil {
		a.finishOperationWithState(ctx, func() {
			a.level = LevelState{}
			a.setStateLocked(StatusError, "Something went wrong", err.Error(), "")
		})
		return
	}

	cfg, err := config.Load()
	if err != nil {
		a.finishOperationWithState(ctx, func() {
			a.level = LevelState{}
			a.setStateLocked(StatusError, "Something went wrong", err.Error(), "")
		})
		return
	}

	recordCtx, recordCancel := context.WithCancel(ctx)
	session, err := a.recorder.Start(recordCtx, audio.RecordOptions{
		Device:     cfg.Audio.Device,
		Driver:     cfg.Audio.Driver,
		SampleRate: cfg.Audio.SampleRate,
		Format:     cfg.Audio.Format,
	})
	if err != nil {
		recordCancel()
		if errors.Is(err, context.Canceled) {
			a.finishOperationWithState(ctx, func() {
				a.level = LevelState{}
				a.setStateLocked(StatusIdle, "Press Enter to record", "", "")
			})
			return
		}
		a.finishOperationWithState(ctx, func() {
			a.level = LevelState{}
			a.setStateLocked(StatusError, "Something went wrong", err.Error(), "")
		})
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	if a.operationCtx != ctx {
		recordCancel()
		_ = session.Cancel()
		return
	}
	if ctx.Err() != nil {
		a.operationCtx = nil
		a.operationCancel = nil
		recordCancel()
		_ = session.Cancel()
		a.level = LevelState{}
		a.setStateLocked(StatusIdle, "Press Enter to record", "", "")
		return
	}
	a.operationCtx = nil
	a.operationCancel = nil
	a.session = session
	a.recordingCancel = recordCancel
	a.level = LevelState{}
	a.setStateLocked(StatusRecording, "Recording...", "", "")
}

func (a *App) stopAndTranscribe(ctx context.Context, session audio.SessionHandle) {
	recording, err := session.Stop(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			a.finishOperationWithState(ctx, func() {
				a.level = LevelState{}
				a.setStateLocked(StatusIdle, "Press Enter to record", "", "")
			})
			return
		}
		a.finishOperationWithState(ctx, func() {
			a.level = LevelState{}
			a.setStateLocked(StatusError, "Something went wrong", err.Error(), "")
		})
		return
	}
	defer os.Remove(recording.Path)

	result, err := a.transcribe.Run(ctx, appsvc.TranscribeInput{
		AudioPath:  recording.Path,
		Target:     output.Target{Kind: output.TargetClipboard},
		SourceKind: "recording",
		SourcePath: "",
	})
	if errors.Is(err, context.Canceled) {
		a.finishOperationWithState(ctx, func() {
			a.level = LevelState{}
			a.setStateLocked(StatusIdle, "Press Enter to record", "", "")
		})
		return
	}
	if err != nil {
		a.finishOperationWithState(ctx, func() {
			a.level = LevelState{}
			a.setStateLocked(StatusError, "Something went wrong", fmt.Errorf("transcribe recording: %w", err).Error(), "")
		})
		return
	}
	_ = a.finishOperationWithState(ctx, func() {
		a.level = LevelState{}
		a.setStateLocked(StatusCopied, "Copied to clipboard", "", result.Text)
	})
}

func (a *App) finishOperationWithState(ctx context.Context, apply func()) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.operationCtx == ctx {
		a.operationCtx = nil
		a.operationCancel = nil
		apply()
		return true
	}
	return false
}

func (a *App) setIdle() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.level = LevelState{}
	a.setStateLocked(StatusIdle, "Press Enter to record", "", "")
}

func (a *App) setError(err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.level = LevelState{}
	a.setStateLocked(StatusError, "Something went wrong", err.Error(), "")
}

func (a *App) setStateLocked(status Status, message, errText, text string) GUIState {
	a.state.Status = string(status)
	a.state.Message = message
	a.state.Error = errText
	a.state.Text = text
	return a.state
}

func (a *App) baseContext() context.Context {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}
