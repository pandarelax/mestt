package app

import (
	"context"

	"pandarelax/mestt/internal/history"
)

type HistoryService struct {
	Store *history.Store
}

func (s HistoryService) List(ctx context.Context, limit int) ([]history.Entry, error) {
	return s.Store.List(ctx, limit)
}
