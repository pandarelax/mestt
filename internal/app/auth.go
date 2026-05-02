package app

import (
	"context"
	"fmt"
	"strings"

	"pandarelax/mestt/internal/config"
	"pandarelax/mestt/internal/secret"
	"pandarelax/mestt/internal/transcribe"
)

type AuthService struct {
	Secrets secret.Store
}

func (s AuthService) SaveOpenAI(ctx context.Context, modelID, apiKey string) error {
	model, err := transcribe.LookupModel(modelID)
	if err != nil {
		return err
	}
	if model.Provider != transcribe.ProviderOpenAI {
		return fmt.Errorf("model %q is not an OpenAI model", modelID)
	}
	if strings.TrimSpace(apiKey) == "" {
		return fmt.Errorf("api key cannot be empty")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	cfg.Transcription.Provider = string(transcribe.ProviderOpenAI)
	cfg.Transcription.Model = modelID
	if err := config.Save(cfg); err != nil {
		return err
	}

	return s.Secrets.Set(ctx, string(transcribe.ProviderOpenAI), strings.TrimSpace(apiKey))
}

func (s AuthService) SaveLocal(modelID string) error {
	model, err := transcribe.LookupModel(modelID)
	if err != nil {
		return err
	}
	if model.Provider != transcribe.ProviderLocal {
		return fmt.Errorf("model %q is not a local model", modelID)
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	cfg.Transcription.Provider = string(transcribe.ProviderLocal)
	cfg.Transcription.Model = modelID
	return config.Save(cfg)
}
