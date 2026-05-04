package popup

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

const childEnvVar = "MESTT_POPUP_CHILD"

type terminalSpec struct {
	name    string
	command func(executable string, args []string) []string
}

type launcher struct {
	lookPath   func(string) (string, error)
	executable func() (string, error)
	start      func(context.Context, string, []string, []string) error
	environ    func() []string
}

var supportedTerminals = []terminalSpec{
	{name: "ghostty", command: func(executable string, args []string) []string { return append([]string{"-e", executable}, args...) }},
	{name: "kitty", command: func(executable string, args []string) []string { return append([]string{executable}, args...) }},
	{name: "alacritty", command: func(executable string, args []string) []string { return append([]string{"-e", executable}, args...) }},
	{name: "wezterm", command: func(executable string, args []string) []string {
		return append([]string{"start", "--", executable}, args...)
	}},
	{name: "x-terminal-emulator", command: func(executable string, args []string) []string { return append([]string{"-e", executable}, args...) }},
	{name: "gnome-terminal", command: func(executable string, args []string) []string { return append([]string{"--", executable}, args...) }},
	{name: "konsole", command: func(executable string, args []string) []string { return append([]string{"-e", executable}, args...) }},
}

func MaybeLaunchRecorder(ctx context.Context) (bool, error) {
	return defaultLauncher().maybeLaunchRecorder(ctx)
}

func IsChild() bool {
	return os.Getenv(childEnvVar) == "1"
}

func defaultLauncher() launcher {
	return launcher{
		lookPath:   exec.LookPath,
		executable: os.Executable,
		start:      startCommand,
		environ:    os.Environ,
	}
}

func startCommand(ctx context.Context, name string, args []string, env []string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = env
	return cmd.Start()
}

func (l launcher) maybeLaunchRecorder(ctx context.Context) (bool, error) {
	if IsChild() {
		return false, nil
	}

	executable, err := l.executable()
	if err != nil {
		return false, fmt.Errorf("resolve current executable: %w", err)
	}

	childArgs := []string{"record", "--compact", "-c"}
	env := append(l.environ(), childEnvVar+"=1")

	var attempted bool
	var lastErr error
	for _, terminal := range supportedTerminals {
		path, err := l.lookPath(terminal.name)
		if err != nil {
			continue
		}
		attempted = true
		if err := l.start(ctx, path, terminal.command(executable, childArgs), env); err == nil {
			return true, nil
		} else {
			lastErr = fmt.Errorf("launch popup with %s: %w", terminal.name, err)
		}
	}

	if !attempted {
		return false, nil
	}
	return false, lastErr
}
