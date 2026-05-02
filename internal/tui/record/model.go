package record

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"pandarelax/mestt/internal/audio"
	"pandarelax/mestt/internal/output"
)

var ErrCanceled = errors.New("recording canceled")

type model struct {
	session   audio.SessionHandle
	target    output.Target
	status    string
	elapsed   time.Duration
	level     float64
	peak      float64
	err       error
	recording audio.Recording
	closing   bool
	quitting  bool
}

type tickMsg struct{}
type stopMsg struct {
	recording audio.Recording
	err       error
}
type cancelMsg struct {
	err error
}

func Run(session audio.SessionHandle, target output.Target) (audio.Recording, error) {
	p := tea.NewProgram(model{session: session, target: target, status: "recording"})
	finalModel, err := p.Run()
	if err != nil {
		return audio.Recording{}, err
	}
	m, ok := finalModel.(model)
	if !ok {
		return audio.Recording{}, fmt.Errorf("unexpected final TUI model type")
	}
	if m.err != nil {
		return audio.Recording{}, m.err
	}
	return m.recording, nil
}

type Runner struct{}

func (Runner) Run(session audio.SessionHandle, target output.Target) (audio.Recording, error) {
	return Run(session, target)
}

func (m model) Init() tea.Cmd {
	return tickCmd()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.closing {
			return m, nil
		}
		switch msg.String() {
		case "enter":
			m.closing = true
			m.status = "stopping"
			return m, func() tea.Msg {
				recording, err := m.session.Stop(context.Background())
				return stopMsg{recording: recording, err: err}
			}
		case "esc", "q", "ctrl+c":
			m.closing = true
			m.status = "canceling"
			return m, func() tea.Msg {
				return cancelMsg{err: m.session.Cancel()}
			}
		}
	case tickMsg:
		if m.quitting {
			return m, nil
		}
		m.elapsed = m.session.Duration()
		level, peak, err := m.session.Levels()
		if err == nil {
			m.level = level
			if peak > m.peak {
				m.peak = peak
			}
		}
		return m, tickCmd()
	case stopMsg:
		m.recording = msg.recording
		m.err = msg.err
		m.quitting = true
		if msg.err == nil {
			m.status = "transcribing"
		}
		return m, tea.Quit
	case cancelMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.err = ErrCanceled
		}
		m.quitting = true
		return m, tea.Quit
	}

	return m, nil
}

func (m model) View() string {
	var b strings.Builder
	b.WriteString("mestt\n\n")
	b.WriteString(fmt.Sprintf("Status: %s\n", m.status))
	b.WriteString(fmt.Sprintf("Duration: %s\n", formatDuration(m.elapsed)))
	b.WriteString(fmt.Sprintf("Level: %.0f%%\n", m.level))
	b.WriteString(fmt.Sprintf("Peak: %.0f%%\n", m.peak))
	b.WriteString(fmt.Sprintf("Output: %s\n\n", describeTarget(m.target)))
	if m.err != nil && !errors.Is(m.err, ErrCanceled) {
		b.WriteString(fmt.Sprintf("Error: %v\n\n", m.err))
	}
	b.WriteString("Enter: stop and transcribe\n")
	b.WriteString("Esc/q/Ctrl+C: cancel\n")
	return b.String()
}

func tickCmd() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
}

func formatDuration(d time.Duration) string {
	seconds := int(d.Seconds())
	if seconds < 0 {
		seconds = 0
	}
	return fmt.Sprintf("%02d:%02d", seconds/60, seconds%60)
}

func describeTarget(target output.Target) string {
	switch target.Kind {
	case output.TargetClipboard:
		return "clipboard"
	case output.TargetFile:
		return target.Path
	default:
		return "stdout"
	}
}
