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
	History historySaver
	Output  outputWriter
	Client  openAITranscriber
	Local   localTranscriber
}

type historySaver interface {
	Save(ctx context.Context, entry history.Entry) error
}

type outputWriter interface {
	Write(ctx context.Context, text string, target output.Target) error
}

type openAITranscriber interface {
	Transcribe(ctx context.Context, req transcribe.Request) (transcribe.Result, error)
}

type localTranscriber interface {
	Transcribe(ctx context.Context, req transcribe.LocalRequest) (transcribe.Result, error)
	Prepare(ctx context.Context, req transcribe.LocalRequest) error
}

type TranscribeInput struct {
	AudioPath  string
	Target     output.Target
	SourceKind string
	SourcePath string
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
	if cfg.Transcription.Provider != "" && cfg.Transcription.Provider != string(model.Provider) {
		return transcribe.Result{}, fmt.Errorf("configured provider %q does not match selected model provider %q", cfg.Transcription.Provider, model.Provider)
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
			AudioPath:       input.AudioPath,
			Model:           model,
			Command:         cfg.Local.Command,
			DownloadCommand: cfg.Local.DownloadCommand,
			ModelPath:       cfg.Local.ModelPath,
			UseGPU:          cfg.Local.UseGPU,
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
		sourcePath := input.SourcePath
		if sourcePath == "" && sourceKind == "file" {
			sourcePath = input.AudioPath
		}
		if err := s.History.Save(ctx, history.Entry{
			CreatedAt:  time.Now(),
			SourceKind: sourceKind,
			SourcePath: sourcePath,
			ModelID:    result.ModelID,
			Transcript: result.Text,
		}); err != nil {
			return transcribe.Result{}, err
		}
	}

	return result, nil
}

func (s TranscribeService) PrepareLocal(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.Transcription.Provider != string(transcribe.ProviderLocal) {
		return nil
	}

	model, err := transcribe.LookupModel(cfg.Transcription.Model)
	if err != nil {
		return err
	}
	if model.Provider != transcribe.ProviderLocal {
		return nil
	}

	return s.Local.Prepare(ctx, transcribe.LocalRequest{
		Model:           model,
		Command:         cfg.Local.Command,
		DownloadCommand: cfg.Local.DownloadCommand,
		ModelPath:       cfg.Local.ModelPath,
		UseGPU:          cfg.Local.UseGPU,
	})
}
