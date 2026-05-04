package app

import (
	"context"
	"errors"
	"fmt"
	"os"

	"pandarelax/mestt/internal/audio"
	"pandarelax/mestt/internal/config"
	"pandarelax/mestt/internal/output"
	"pandarelax/mestt/internal/transcribe"
	recordtui "pandarelax/mestt/internal/tui/record"
)

type RecordService struct {
	Recorder   recorder
	RunUI      recordingUIRunner
	Transcribe transcribeRunner
	Prepare    localPreparer
}

type recorder interface {
	Start(ctx context.Context, opts audio.RecordOptions) (audio.SessionHandle, error)
}

type recordingUIRunner interface {
	RunPrepare(opts recordtui.PrepareOptions, action func(context.Context) error) error
	Run(session audio.SessionHandle, target output.Target, opts recordtui.Options) error
}

type transcribeRunner interface {
	Run(ctx context.Context, input TranscribeInput) (transcribe.Result, error)
}

type localPreparer interface {
	PrepareLocal(ctx context.Context) error
}

type RecordInput struct {
	Target  output.Target
	Compact bool
}

func (s RecordService) Run(ctx context.Context, input RecordInput) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.Transcription.Provider == string(transcribe.ProviderLocal) && s.Prepare != nil {
		err := s.RunUI.RunPrepare(recordtui.PrepareOptions{Compact: input.Compact, Status: "preparing local model"}, func(prepareCtx context.Context) error {
			return s.Prepare.PrepareLocal(prepareCtx)
		})
		if err != nil {
			if errors.Is(err, recordtui.ErrCanceled) || errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}
	}

	session, err := s.Recorder.Start(ctx, audio.RecordOptions{
		Device:     cfg.Audio.Device,
		Driver:     cfg.Audio.Driver,
		SampleRate: cfg.Audio.SampleRate,
		Format:     cfg.Audio.Format,
	})
	if err != nil {
		return err
	}

	err = s.RunUI.Run(session, input.Target, recordtui.Options{Compact: input.Compact, Submit: func(recording audio.Recording) error {
		defer os.Remove(recording.Path)
		_, err := s.Transcribe.Run(ctx, TranscribeInput{
			AudioPath:  recording.Path,
			Target:     input.Target,
			SourceKind: "recording",
			SourcePath: "",
		})
		if err != nil {
			return fmt.Errorf("transcribe recording: %w", err)
		}
		return nil
	}})
	if err != nil {
		if errors.Is(err, recordtui.ErrCanceled) {
			return nil
		}
		return err
	}

	return nil
}
