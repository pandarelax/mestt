package logging

import (
	"fmt"
	"log/slog"
	"os"

	"pandarelax/mestt/internal/paths"
)

func Setup() (*slog.Logger, error) {
	p := paths.Resolve()
	if err := p.Ensure(); err != nil {
		return nil, fmt.Errorf("ensure state directories: %w", err)
	}

	file, err := os.OpenFile(p.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	logger := slog.New(slog.NewJSONHandler(file, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	return logger, nil
}
