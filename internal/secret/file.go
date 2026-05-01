package secret

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"pandarelax/mestt/internal/paths"
)

type Store interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
}

type FileStore struct{}

type fileSecrets map[string]string

func NewFileStore() FileStore {
	return FileStore{}
}

func (FileStore) Get(_ context.Context, key string) (string, error) {
	secrets, err := load()
	if err != nil {
		return "", err
	}
	return secrets[key], nil
}

func (FileStore) Set(_ context.Context, key, value string) error {
	secrets, err := load()
	if err != nil {
		return err
	}
	secrets[key] = value
	return save(secrets)
}

func load() (fileSecrets, error) {
	p := paths.Resolve()
	if err := p.Ensure(); err != nil {
		return nil, fmt.Errorf("ensure secret directories: %w", err)
	}

	if _, err := os.Stat(p.SecretsFile); os.IsNotExist(err) {
		return fileSecrets{}, nil
	}

	data, err := os.ReadFile(p.SecretsFile)
	if err != nil {
		return nil, fmt.Errorf("read secrets file: %w", err)
	}

	if len(data) == 0 {
		return fileSecrets{}, nil
	}

	var secrets fileSecrets
	if err := json.Unmarshal(data, &secrets); err != nil {
		return nil, fmt.Errorf("parse secrets file: %w", err)
	}

	return secrets, nil
}

func save(secrets fileSecrets) error {
	p := paths.Resolve()
	data, err := json.MarshalIndent(secrets, "", "  ")
	if err != nil {
		return fmt.Errorf("encode secrets: %w", err)
	}

	if err := os.WriteFile(p.SecretsFile, data, 0o600); err != nil {
		return fmt.Errorf("write secrets file: %w", err)
	}

	return nil
}
