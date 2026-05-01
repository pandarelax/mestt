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
	if _, err := transcribe.LookupModel(modelID); err != nil {
		return err
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
