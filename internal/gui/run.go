//go:build fyne

package gui

import (
	"context"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
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

	statusLabel := widget.NewLabel("Press Enter to record")
	detailLabel := widget.NewLabel("Ready to record.")
	detailLabel.Wrapping = fyne.TextWrapWord
	meter := widget.NewProgressBar()
	primary := widget.NewButton("Record", nil)
	secondary := widget.NewButton("Close", nil)

	content := container.NewVBox(
		widget.NewLabel("mestt"),
		statusLabel,
		detailLabel,
		meter,
		container.NewGridWithColumns(2, primary, secondary),
	)
	win.SetContent(content)

	applyState := func(state GUIState, levels LevelState) {
		statusLabel.SetText(state.Message)
		detailLabel.SetText(detailText(state))
		applyButtons(primary, secondary, state.Status)
		if state.Status == string(StatusRecording) {
			meter.SetValue(clamp01(levels.Level / 100))
		} else {
			meter.SetValue(idleMeterValue(state.Status))
		}
	}

	triggerPrimary := func() {
		state := backend.Status()
		switch state.Status {
		case string(StatusRecording):
			applyState(backend.StopAndTranscribe(), LevelState{})
		case string(StatusCopied), string(StatusError):
			backend.DismissError()
			applyState(backend.StartRecording(), LevelState{})
		case string(StatusPreparing), string(StatusTranscribing):
			return
		default:
			applyState(backend.StartRecording(), LevelState{})
		}
	}

	triggerSecondary := func() {
		state := backend.Status()
		switch state.Status {
		case string(StatusPreparing), string(StatusRecording), string(StatusTranscribing):
			applyState(backend.CancelRecording(), LevelState{})
		case string(StatusCopied), string(StatusError):
			applyState(backend.DismissError(), LevelState{})
		default:
			win.Close()
		}
	}

	primary.OnTapped = triggerPrimary
	secondary.OnTapped = triggerSecondary
	win.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
		switch ev.Name {
		case fyne.KeyReturn, fyne.KeyEnter:
			triggerPrimary()
		case fyne.KeyEscape:
			triggerSecondary()
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
			})
		}
	}()

	applyState(backend.Status(), LevelState{})
	win.ShowAndRun()
	return nil
}

func applyButtons(primary, secondary *widget.Button, status string) {
	primary.Enable()
	secondary.Enable()
	switch status {
	case string(StatusPreparing):
		primary.SetText("Preparing...")
		primary.Disable()
		secondary.SetText("Cancel")
	case string(StatusRecording):
		primary.SetText("Stop")
		secondary.SetText("Cancel")
	case string(StatusTranscribing):
		primary.SetText("Transcribing...")
		primary.Disable()
		secondary.SetText("Cancel")
	case string(StatusCopied):
		primary.SetText("Record Again")
		secondary.SetText("Dismiss")
	case string(StatusError):
		primary.SetText("Retry")
		secondary.SetText("Dismiss")
	default:
		primary.SetText("Record")
		secondary.SetText("Close")
	}
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
