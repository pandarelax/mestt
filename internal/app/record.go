package app

import (
	"context"
	"errors"
	"fmt"
	"os"

	"pandarelax/mestt/internal/audio"
	"pandarelax/mestt/internal/config"
	"pandarelax/mestt/internal/output"
	recordtui "pandarelax/mestt/internal/tui/record"
)

type RecordService struct {
	Recorder   audio.Recorder
	Transcribe TranscribeService
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
		SampleRate: cfg.Audio.SampleRate,
		Format:     cfg.Audio.Format,
	})
	if err != nil {
		return err
	}

	recording, err := recordtui.Run(session, input.Target)
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
	})
	if err != nil {
		return fmt.Errorf("transcribe recording: %w", err)
	}

	return nil
}
