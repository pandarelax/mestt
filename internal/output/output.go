package output

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

type TargetKind string

const (
	TargetStdout    TargetKind = "stdout"
	TargetClipboard TargetKind = "clipboard"
	TargetFile      TargetKind = "file"
)

type Target struct {
	Kind TargetKind
	Path string
}

type Writer struct{}

func (Writer) Write(_ context.Context, text string, target Target) error {
	switch target.Kind {
	case TargetStdout:
		_, err := fmt.Fprintln(os.Stdout, text)
		return err
	case TargetFile:
		return os.WriteFile(target.Path, []byte(text), 0o644)
	case TargetClipboard:
		return copyToClipboard(text)
	default:
		return fmt.Errorf("unsupported output target %q", target.Kind)
	}
}

func copyToClipboard(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	default:
		if path, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.Command(path)
		} else if path, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command(path, "-selection", "clipboard")
		} else {
			return fmt.Errorf("clipboard command not found; install wl-copy or xclip")
		}
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("open clipboard stdin: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start clipboard command: %w", err)
	}

	if _, err := stdin.Write([]byte(text)); err != nil {
		_ = stdin.Close()
		return fmt.Errorf("write clipboard data: %w", err)
	}

	if err := stdin.Close(); err != nil {
		return fmt.Errorf("close clipboard stdin: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("clipboard command failed: %w", err)
	}

	return nil
}
