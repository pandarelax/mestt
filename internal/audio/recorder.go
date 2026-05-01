package audio

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"pandarelax/mestt/internal/paths"
)

type Recording struct {
	Path      string
	Duration  time.Duration
	CreatedAt time.Time
}

type Recorder struct {
	FFmpegPath string
	GOOS       string
}

type Session struct {
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	outputPath string
	startedAt  time.Time
	mu         sync.Mutex
	closed     bool
	stderr     strings.Builder
}

func NewRecorder() Recorder {
	return Recorder{GOOS: currentOS()}
}

func (r Recorder) Start(ctx context.Context, opts RecordOptions) (*Session, error) {
	ffmpegPath := r.FFmpegPath
	if ffmpegPath == "" {
		path, err := exec.LookPath("ffmpeg")
		if err != nil {
			return nil, fmt.Errorf("ffmpeg not found in PATH")
		}
		ffmpegPath = path
	}

	p := paths.Resolve()
	if err := p.Ensure(); err != nil {
		return nil, fmt.Errorf("ensure data directories: %w", err)
	}

	tempFile, err := os.CreateTemp(p.DataDir, "recording-*.wav")
	if err != nil {
		return nil, fmt.Errorf("create temporary recording file: %w", err)
	}
	outputPath := tempFile.Name()
	if err := tempFile.Close(); err != nil {
		return nil, fmt.Errorf("close temporary recording file: %w", err)
	}

	args, err := buildRecordArgs(r.goos(), opts, outputPath)
	if err != nil {
		_ = os.Remove(outputPath)
		return nil, err
	}

	cmd := exec.CommandContext(ctx, ffmpegPath, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		_ = os.Remove(outputPath)
		return nil, fmt.Errorf("open ffmpeg stdin: %w", err)
	}
	cmd.Stdout = io.Discard
	cmd.Stderr = &strings.Builder{}

	session := &Session{
		cmd:        cmd,
		stdin:      stdin,
		outputPath: outputPath,
		startedAt:  time.Now(),
	}
	cmd.Stderr = &session.stderr

	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = os.Remove(outputPath)
		return nil, fmt.Errorf("start ffmpeg: %w", err)
	}

	return session, nil
}

func (s *Session) Stop(ctx context.Context) (Recording, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return Recording{}, fmt.Errorf("recording session already closed")
	}
	s.closed = true
	s.mu.Unlock()

	if _, err := io.WriteString(s.stdin, "q\n"); err != nil && !errors.Is(err, os.ErrClosed) {
		_ = s.stdin.Close()
		_ = s.cmd.Process.Kill()
		_ = os.Remove(s.outputPath)
		return Recording{}, fmt.Errorf("signal ffmpeg to stop: %w", err)
	}
	_ = s.stdin.Close()

	waitCh := make(chan error, 1)
	go func() { waitCh <- s.cmd.Wait() }()

	select {
	case err := <-waitCh:
		if err != nil {
			stderr := strings.TrimSpace(s.stderr.String())
			_ = os.Remove(s.outputPath)
			if stderr != "" {
				return Recording{}, fmt.Errorf("ffmpeg exited with error: %s", stderr)
			}
			return Recording{}, fmt.Errorf("wait for ffmpeg: %w", err)
		}
	case <-ctx.Done():
		_ = s.cmd.Process.Kill()
		_ = os.Remove(s.outputPath)
		return Recording{}, ctx.Err()
	}

	return Recording{Path: s.outputPath, Duration: s.Duration(), CreatedAt: s.startedAt}, nil
}

func (s *Session) Cancel() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.mu.Unlock()

	_ = s.stdin.Close()
	if s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	_ = s.cmd.Wait()
	if err := os.Remove(s.outputPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove temporary recording: %w", err)
	}
	return nil
}

func (s *Session) Duration() time.Duration {
	return time.Since(s.startedAt).Round(time.Second)
}

func (s *Session) Levels() (float64, float64, error) {
	return ReadWAVLevel(s.outputPath)
}

func (s *Session) Path() string {
	return s.outputPath
}

func (r Recorder) ListDevices(ctx context.Context) ([]Device, error) {
	ffmpegPath := r.FFmpegPath
	if ffmpegPath == "" {
		path, err := exec.LookPath("ffmpeg")
		if err != nil {
			return nil, fmt.Errorf("ffmpeg not found in PATH")
		}
		ffmpegPath = path
	}

	goos := r.goos()
	if goos == "linux" {
		for _, driver := range []string{"pulse", "alsa"} {
			devices, err := runListDevices(ctx, ffmpegPath, goos, driver)
			if err == nil && len(devices) > 0 {
				return devices, nil
			}
		}
		return nil, fmt.Errorf("no audio input devices detected via ffmpeg")
	}

	devices, err := runListDevices(ctx, ffmpegPath, goos, "")
	if err != nil {
		return nil, err
	}
	if len(devices) == 0 {
		return nil, fmt.Errorf("no audio input devices detected via ffmpeg")
	}
	return devices, nil
}

func runListDevices(ctx context.Context, ffmpegPath, goos, driver string) ([]Device, error) {
	args, err := buildListDevicesArgs(goos, driver)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, ffmpegPath, args...)
	output, err := cmd.CombinedOutput()
	devices := parseDevices(goos, string(output))
	if len(devices) > 0 {
		return devices, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ffmpeg list devices failed: %w", err)
	}
	return nil, nil
}

func (r Recorder) goos() string {
	if r.GOOS != "" {
		return r.GOOS
	}
	return currentOS()
}

func tempRecordingName(dir string) string {
	return filepath.Join(dir, fmt.Sprintf("recording-%d.wav", time.Now().UnixNano()))
}
