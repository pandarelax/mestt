package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"pandarelax/mestt/internal/config"
	"pandarelax/mestt/internal/history"
	"pandarelax/mestt/internal/output"
	"pandarelax/mestt/internal/secret"
	"pandarelax/mestt/internal/transcribe"
)

type TranscribeService struct {
	Secrets secret.Store
	History *history.Store
	Output  output.Writer
	Client  transcribe.OpenAIClient
	Local   transcribe.LocalClient
}

type TranscribeInput struct {
	AudioPath  string
	Target     output.Target
	SourceKind string
}

func (s TranscribeService) Run(ctx context.Context, input TranscribeInput) (transcribe.Result, error) {
	cfg, err := config.Load()
	if err != nil {
		return transcribe.Result{}, err
	}

	model, err := transcribe.LookupModel(cfg.Transcription.Model)
	if err != nil {
		return transcribe.Result{}, err
	}

	var result transcribe.Result
	switch model.Provider {
	case transcribe.ProviderOpenAI:
		apiKey, err := s.Secrets.Get(ctx, string(transcribe.ProviderOpenAI))
		if err != nil {
			return transcribe.Result{}, err
		}
		if strings.TrimSpace(apiKey) == "" {
			return transcribe.Result{}, fmt.Errorf("no OpenAI API key configured; run 'mestt auth'")
		}

		result, err = s.Client.Transcribe(ctx, transcribe.Request{
			AudioPath: input.AudioPath,
			APIKey:    apiKey,
			Model:     model,
			BaseURL:   strings.TrimRight(cfg.Transcription.BaseURL, "/"),
			Timeout:   time.Duration(cfg.Transcription.TimeoutSeconds) * time.Second,
		})
	case transcribe.ProviderLocal:
		result, err = s.Local.Transcribe(ctx, transcribe.LocalRequest{
			AudioPath:     input.AudioPath,
			Model:         model,
			PythonCommand: cfg.Local.PythonCommand,
			Device:        cfg.Local.Device,
			ComputeType:   cfg.Local.ComputeType,
		})
	default:
		return transcribe.Result{}, fmt.Errorf("unsupported transcription provider %q", model.Provider)
	}
	if err != nil {
		return transcribe.Result{}, err
	}

	if err := s.Output.Write(ctx, result.Text, input.Target); err != nil {
		return transcribe.Result{}, fmt.Errorf("write transcription output: %w", err)
	}

	if s.History != nil {
		sourceKind := input.SourceKind
		if sourceKind == "" {
			sourceKind = "file"
		}
		if err := s.History.Save(ctx, history.Entry{
			CreatedAt:  time.Now(),
			SourceKind: sourceKind,
			SourcePath: input.AudioPath,
			ModelID:    result.ModelID,
			Transcript: result.Text,
		}); err != nil {
			return transcribe.Result{}, err
		}
	}

	return result, nil
}
