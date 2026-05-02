package config

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"

	"pandarelax/mestt/internal/paths"
)

type Config struct {
	Audio         AudioConfig         `toml:"audio"`
	Transcription TranscriptionConfig `toml:"transcription"`
	Local         LocalConfig         `toml:"local"`
	Output        OutputConfig        `toml:"output"`
}

type AudioConfig struct {
	Device     string `toml:"device"`
	SampleRate int    `toml:"sample_rate"`
	Format     string `toml:"format"`
}

type TranscriptionConfig struct {
	Provider       string `toml:"provider"`
	Model          string `toml:"model"`
	TimeoutSeconds int    `toml:"timeout_seconds"`
	BaseURL        string `toml:"base_url"`
}

type LocalConfig struct {
	PythonCommand string `toml:"python_command"`
	Device        string `toml:"device"`
	ComputeType   string `toml:"compute_type"`
}

type OutputConfig struct {
	DefaultTarget string `toml:"default_target"`
}

func Default() Config {
	return Config{
		Audio: AudioConfig{
			Device:     "default",
			SampleRate: 16000,
			Format:     "wav",
		},
		Transcription: TranscriptionConfig{
			Provider:       "openai",
			Model:          "gpt-4o-mini-transcribe",
			TimeoutSeconds: 120,
			BaseURL:        "https://api.openai.com/v1",
		},
		Local: LocalConfig{
			PythonCommand: "python3",
			Device:        "cpu",
			ComputeType:   "int8",
		},
		Output: OutputConfig{
			DefaultTarget: "stdout",
		},
	}
}

func Load() (Config, error) {
	p := paths.Resolve()
	if err := p.Ensure(); err != nil {
		return Config{}, fmt.Errorf("ensure config directories: %w", err)
	}

	if _, err := os.Stat(p.ConfigFile); os.IsNotExist(err) {
		cfg := Default()
		if err := Save(cfg); err != nil {
			return Config{}, err
		}
		return cfg, nil
	}

	data, err := os.ReadFile(p.ConfigFile)
	if err != nil {
		return Config{}, fmt.Errorf("read config file: %w", err)
	}

	cfg := Default()
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config file: %w", err)
	}

	return cfg, nil
}

func Save(cfg Config) error {
	p := paths.Resolve()
	if err := p.Ensure(); err != nil {
		return fmt.Errorf("ensure config directories: %w", err)
	}

	data, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}

	if err := os.WriteFile(p.ConfigFile, data, 0o644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}
