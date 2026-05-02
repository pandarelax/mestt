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
}

type recorder interface {
	Start(ctx context.Context, opts audio.RecordOptions) (audio.SessionHandle, error)
}

type recordingUIRunner interface {
	Run(session audio.SessionHandle, target output.Target) (audio.Recording, error)
}

type transcribeRunner interface {
	Run(ctx context.Context, input TranscribeInput) (transcribe.Result, error)
}

type RecordInput struct {
	Target output.Target
}

func (s RecordService) Run(ctx context.Context, input RecordInput) error {
	cfg, err := config.Load()
	if err != nil {
		return err
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

	recording, err := s.RunUI.Run(session, input.Target)
	if err != nil {
		if errors.Is(err, recordtui.ErrCanceled) {
			return nil
		}
		return err
	}
	defer os.Remove(recording.Path)

	_, err = s.Transcribe.Run(ctx, TranscribeInput{
		AudioPath:  recording.Path,
		Target:     input.Target,
		SourceKind: "recording",
		SourcePath: "",
	})
	if err != nil {
		return fmt.Errorf("transcribe recording: %w", err)
	}

	return nil
}
