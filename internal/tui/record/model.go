package record

import (
	"context"
	"errors"
	"fmt"
	"math"
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
	compact   bool
	submit    func(audio.Recording) error
	status    string
	elapsed   time.Duration
	level     float64
	peak      float64
	history   []float64
	frame     int
	err       error
	recording audio.Recording
	closing   bool
	quitting  bool
}

type prepareModel struct {
	compact  bool
	status   string
	ctx      context.Context
	cancel   context.CancelFunc
	action   func(context.Context) error
	err      error
	done     bool
	active   bool
	quitting bool
}

type (
	tickMsg struct{}
	stopMsg struct {
		recording audio.Recording
		err       error
	}
	prepareDoneMsg struct {
		err error
	}
	transcribeMsg struct {
		err error
	}
)

type cancelMsg struct {
	err error
}

type Options struct {
	Compact bool
	Submit  func(audio.Recording) error
}

type PrepareOptions struct {
	Compact bool
	Status  string
}

func Run(session audio.SessionHandle, target output.Target, opts Options) error {
	p := tea.NewProgram(model{session: session, target: target, compact: opts.Compact, submit: opts.Submit, status: "recording"})
	finalModel, err := p.Run()
	if err != nil {
		return err
	}
	m, ok := finalModel.(model)
	if !ok {
		return fmt.Errorf("unexpected final TUI model type")
	}
	if m.err != nil {
		return m.err
	}
	return nil
}

func RunPrepare(opts PrepareOptions, action func(context.Context) error) error {
	status := strings.TrimSpace(opts.Status)
	if status == "" {
		status = "preparing"
	}
	prepareCtx, cancel := context.WithCancel(context.Background())
	p := tea.NewProgram(prepareModel{compact: opts.Compact, status: status, ctx: prepareCtx, cancel: cancel, action: action, active: true})
	finalModel, err := p.Run()
	cancel()
	if err != nil {
		return err
	}
	m, ok := finalModel.(prepareModel)
	if !ok {
		return fmt.Errorf("unexpected final prepare TUI model type")
	}
	return m.err
}

type Runner struct{}

func (Runner) Run(session audio.SessionHandle, target output.Target, opts Options) error {
	return Run(session, target, opts)
}

func (Runner) RunPrepare(opts PrepareOptions, action func(context.Context) error) error {
	return RunPrepare(opts, action)
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
		m.frame++
		m.elapsed = m.session.Duration()
		level, peak, err := m.session.Levels()
		if err == nil {
			m.level = level
			m.history = appendHistory(m.history, level, visualizationHistorySize(m.compact))
			if peak > m.peak {
				m.peak = peak
			} else {
				m.peak = math.Max(peak, m.peak-4)
			}
		}
		return m, tickCmd()
	case stopMsg:
		m.recording = msg.recording
		m.err = msg.err
		if msg.err != nil {
			m.quitting = true
			return m, tea.Quit
		}
		m.status = "transcribing"
		return m, func() tea.Msg {
			if m.submit == nil {
				return transcribeMsg{}
			}
			return transcribeMsg{err: m.submit(msg.recording)}
		}
	case transcribeMsg:
		m.err = msg.err
		if msg.err == nil {
			m.quitting = true
			m.status = "done"
			return m, tea.Quit
		}
		m.status = "error"
		m.closing = false
		return m, nil
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

func (m prepareModel) Init() tea.Cmd {
	return func() tea.Msg {
		if m.action == nil {
			return prepareDoneMsg{}
		}
		return prepareDoneMsg{err: m.action(m.ctx)}
	}
}

func (m prepareModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q", "ctrl+c":
			if m.cancel != nil {
				m.cancel()
			}
			m.err = ErrCanceled
			m.quitting = true
			return m, tea.Quit
		case "enter":
			if !m.active {
				m.quitting = true
				return m, tea.Quit
			}
		}
	case prepareDoneMsg:
		if errors.Is(m.err, ErrCanceled) {
			return m, nil
		}
		m.err = msg.err
		m.active = false
		m.done = true
		if msg.err == nil {
			m.quitting = true
			return m, tea.Quit
		}
		m.status = "error"
	}

	return m, nil
}

func (m model) View() string {
	var b strings.Builder
	b.WriteString("mestt\n\n")
	b.WriteString(renderVisualization(m.history, visualizationWidth(m.compact), visualizationHeight(m.compact), m.frame))
	b.WriteString("\n")
	if m.compact {
		b.WriteString(fmt.Sprintf("%s  %s  L %.0f%%  P %.0f%%\n", formatDuration(m.elapsed), m.status, m.level, m.peak))
		b.WriteString(fmt.Sprintf("output: %s\n", describeTarget(m.target)))
	} else {
		b.WriteString(fmt.Sprintf("Status: %s\n", m.status))
		b.WriteString(fmt.Sprintf("Duration: %s\n", formatDuration(m.elapsed)))
		b.WriteString(fmt.Sprintf("Level: %.0f%%\n", m.level))
		b.WriteString(fmt.Sprintf("Peak: %.0f%%\n", m.peak))
		b.WriteString(fmt.Sprintf("Output: %s\n", describeTarget(m.target)))
	}
	b.WriteString("\n")
	if m.err != nil && !errors.Is(m.err, ErrCanceled) {
		b.WriteString(fmt.Sprintf("Error: %v\n\n", m.err))
	}
	b.WriteString("Enter: stop and transcribe\n")
	b.WriteString("Esc/q/Ctrl+C: cancel\n")
	return b.String()
}

func (m prepareModel) View() string {
	var b strings.Builder
	b.WriteString("mestt\n\n")
	if m.compact {
		b.WriteString(m.status)
		if m.active {
			b.WriteString("...\n\nPlease wait\n")
		} else if m.err != nil {
			b.WriteString("\n\n")
			b.WriteString(fmt.Sprintf("Error: %v\n\n", m.err))
			b.WriteString("Enter/Esc/q/Ctrl+C: close\n")
		}
		return b.String()
	}

	b.WriteString(fmt.Sprintf("Status: %s\n\n", m.status))
	if m.active {
		b.WriteString("Preparing local transcription before recording starts.\n")
		b.WriteString("Please wait.\n")
		return b.String()
	}
	if m.err != nil {
		b.WriteString(fmt.Sprintf("Error: %v\n\n", m.err))
		b.WriteString("Enter/Esc/q/Ctrl+C: close\n")
	}
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

func appendHistory(history []float64, value float64, limit int) []float64 {
	history = append(history, value)
	if len(history) <= limit {
		return history
	}
	trimmed := make([]float64, limit)
	copy(trimmed, history[len(history)-limit:])
	return trimmed
}

func renderVisualization(history []float64, width, height, frame int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	cols := make([]int, width)
	for col := 0; col < width; col++ {
		strength := visualizationStrength(history, width, col, frame)
		cols[col] = int(math.Round(strength * float64(height)))
		if cols[col] < 0 {
			cols[col] = 0
		}
		if cols[col] > height {
			cols[col] = height
		}
	}

	var b strings.Builder
	for row := height; row >= 1; row-- {
		for _, colHeight := range cols {
			if colHeight >= row {
				b.WriteByte('#')
			} else {
				b.WriteByte(' ')
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func visualizationStrength(history []float64, width, col, frame int) float64 {
	if len(history) == 0 {
		return 0
	}
	idx := len(history) - 1 - ((width - 1 - col) * len(history) / width)
	if idx < 0 {
		idx = 0
	}
	base := history[idx] / 100
	left := history[maxInt(0, idx-1)] / 100
	right := history[minInt(len(history)-1, idx+1)] / 100
	smoothed := (base*0.6 + left*0.2 + right*0.2)
	wobble := 0.08 * math.Sin(float64(frame+col*2)*0.45)
	shape := 0.12 * math.Sin(float64(col)*0.55)
	strength := smoothed + wobble + shape*smoothed
	if strength < 0 {
		return 0
	}
	if strength > 1 {
		return 1
	}
	return strength
}

func visualizationWidth(compact bool) int {
	if compact {
		return 24
	}
	return 32
}

func visualizationHeight(compact bool) int {
	if compact {
		return 5
	}
	return 7
}

func visualizationHistorySize(compact bool) int {
	if compact {
		return 48
	}
	return 64
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
