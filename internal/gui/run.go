//go:build fyne

package gui

import (
	"context"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"pandarelax/mestt/internal/logging"
)

func RunFyne() error {
	if _, err := logging.Setup(); err != nil {
		return err
	}

	backend, err := NewApp()
	if err != nil {
		return err
	}
	backend.Startup(context.Background())
	defer backend.Shutdown(context.Background())

	guiApp := app.NewWithID("pandarelax.mestt")
	win := guiApp.NewWindow("mestt-record")
	win.Resize(fyne.NewSize(420, 240))
	win.SetFixedSize(true)

	var closeOnce sync.Once
	scheduleClose := func() {
		closeOnce.Do(func() {
			fyne.Do(func() {
				win.Close()
			})
		})
	}

	statusLabel := widget.NewLabel("Press Enter to record")
	detailLabel := widget.NewLabel("Ready to record.")
	detailLabel.Wrapping = fyne.TextWrapWord
	hintLabel := widget.NewLabel("Enter stop  Esc cancel")
	hintLabel.Alignment = fyne.TextAlignCenter
	waveform := newWaveformWidget()

	content := container.NewVBox(
		widget.NewLabel("mestt"),
		statusLabel,
		detailLabel,
		waveform,
		layout.NewSpacer(),
		hintLabel,
	)
	win.SetContent(content)

	applyState := func(state GUIState, levels LevelState) {
		statusLabel.SetText(state.Message)
		detailLabel.SetText(detailText(state))
		if state.Status == string(StatusRecording) {
			waveform.Push(levels.Level, levels.Peak)
		} else {
			waveform.SetIdle(state.Status)
		}
		waveform.Refresh()
		hintLabel.SetText(hintText(state.Status))
	}

	stopRecording := func() {
		state := backend.Status()
		switch state.Status {
		case string(StatusRecording):
			applyState(backend.StopAndTranscribe(), LevelState{})
		case string(StatusCopied):
			scheduleClose()
		case string(StatusError):
			return
		case string(StatusPreparing), string(StatusTranscribing):
			return
		default:
			applyState(backend.StartRecording(), LevelState{})
		}
	}

	cancelAndClose := func() {
		state := backend.Status()
		switch state.Status {
		case string(StatusPreparing), string(StatusRecording), string(StatusTranscribing):
			backend.CancelRecording()
			scheduleClose()
		default:
			scheduleClose()
		}
	}

	win.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
		switch ev.Name {
		case fyne.KeyReturn, fyne.KeyEnter:
			stopRecording()
		case fyne.KeyEscape:
			cancelAndClose()
		}
	})

	go func() {
		ticker := time.NewTicker(120 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			state := backend.Status()
			levels := LevelState{}
			if state.Status == string(StatusRecording) {
				levels = backend.Levels()
			}
			fyne.Do(func() {
				applyState(state, levels)
				if state.Status == string(StatusCopied) {
					scheduleClose()
				}
			})
		}
	}()

	applyState(backend.StartRecording(), LevelState{})
	win.ShowAndRun()
	return nil
}

func detailText(state GUIState) string {
	if state.Error != "" {
		return state.Error
	}
	if state.Text != "" && state.Status == string(StatusCopied) {
		return state.Text
	}
	switch state.Status {
	case string(StatusPreparing):
		return "Checking local model and dependencies before recording starts."
	case string(StatusRecording):
		return "Recording microphone audio. Press Enter again to stop."
	case string(StatusTranscribing):
		return "Turning the captured audio into text and copying it to the clipboard."
	case string(StatusCopied):
		return "Transcript copied to the clipboard."
	default:
		return "Ready to record."
	}
}

func hintText(status string) string {
	switch status {
	case string(StatusPreparing):
		return "Esc cancel"
	case string(StatusRecording):
		return "Enter stop  Esc cancel"
	case string(StatusTranscribing):
		return "Esc cancel"
	case string(StatusError):
		return "Esc close"
	default:
		return ""
	}
}

func idleMeterValue(status string) float64 {
	switch status {
	case string(StatusPreparing):
		return 0.42
	case string(StatusTranscribing):
		return 0.58
	case string(StatusCopied):
		return 1
	case string(StatusError):
		return 0.22
	default:
		return 0.18
	}
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}
