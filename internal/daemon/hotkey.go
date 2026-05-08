package daemon

import (
	"context"
	"fmt"
)

type HotkeyBackend interface {
	Start(ctx context.Context, trigger func()) error
}

type UnsupportedHotkeyBackend struct {
	Reason string
}

func (b UnsupportedHotkeyBackend) Start(context.Context, func()) error {
	if b.Reason == "" {
		return fmt.Errorf("hotkey backend is not implemented")
	}
	return fmt.Errorf("hotkey backend is not implemented: %s", b.Reason)
}
