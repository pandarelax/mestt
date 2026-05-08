package daemon

import (
	"context"
	"fmt"
	"os/exec"
)

type TriggerOptions struct {
	Command string
	Args    []string
}

type launcher struct {
	lookPath func(string) (string, error)
	start    func(context.Context, string, []string) error
}

func Trigger(ctx context.Context, opts TriggerOptions) error {
	return defaultLauncher().Trigger(ctx, opts)
}

func defaultLauncher() launcher {
	return launcher{
		lookPath: exec.LookPath,
		start: func(ctx context.Context, name string, args []string) error {
			return exec.CommandContext(ctx, name, args...).Start()
		},
	}
}

func (l launcher) Trigger(ctx context.Context, opts TriggerOptions) error {
	if opts.Command == "" {
		return fmt.Errorf("trigger command is empty")
	}
	if l.lookPath == nil {
		l.lookPath = exec.LookPath
	}
	if l.start == nil {
		l.start = func(ctx context.Context, name string, args []string) error {
			return exec.CommandContext(ctx, name, args...).Start()
		}
	}

	path, err := l.lookPath(opts.Command)
	if err != nil {
		return fmt.Errorf("trigger command %q not found: %w", opts.Command, err)
	}
	return l.start(ctx, path, append([]string(nil), opts.Args...))
}
