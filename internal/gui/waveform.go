//go:build fyne

package gui

import (
	"image/color"
	"math"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

const waveformBars = 40

type waveformWidget struct {
	widget.BaseWidget

	mu      sync.RWMutex
	history []float64
	peak    float64
	lines   []*canvas.Line
	base    *canvas.Line
}

func newWaveformWidget() *waveformWidget {
	w := &waveformWidget{
		history: make([]float64, waveformBars),
		lines:   make([]*canvas.Line, waveformBars),
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *waveformWidget) Push(level, peak float64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	amplitude := clamp01(level / 100)
	if peakValue := clamp01(peak / 100); peakValue > amplitude {
		amplitude = (amplitude + peakValue) / 2
	}

	copy(w.history, w.history[1:])
	w.history[len(w.history)-1] = amplitude
	w.peak = clamp01(peak / 100)
}

func (w *waveformWidget) SetIdle(status string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	base := idleMeterValue(status) * 0.35
	for i := range w.history {
		falloff := float64(i+1) / float64(len(w.history))
		w.history[i] = base * falloff
	}
	w.peak = base
}

func (w *waveformWidget) CreateRenderer() fyne.WidgetRenderer {
	objects := make([]fyne.CanvasObject, 0, waveformBars+1)
	w.base = canvas.NewLine(color.NRGBA{R: 70, G: 78, B: 92, A: 255})
	w.base.StrokeWidth = 1
	objects = append(objects, w.base)

	for i := range w.lines {
		line := canvas.NewLine(color.NRGBA{R: 114, G: 168, B: 255, A: 255})
		line.StrokeWidth = 3
		w.lines[i] = line
		objects = append(objects, line)
	}

	r := &waveformRenderer{widget: w, objects: objects}
	r.Refresh()
	return r
}

type waveformRenderer struct {
	widget  *waveformWidget
	objects []fyne.CanvasObject
}

func (r *waveformRenderer) Destroy() {}

func (r *waveformRenderer) Layout(size fyne.Size) {
	r.paint(size)
}

func (r *waveformRenderer) MinSize() fyne.Size {
	return fyne.NewSize(320, 84)
}

func (r *waveformRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *waveformRenderer) Refresh() {
	r.paint(r.widget.Size())
	for _, object := range r.objects {
		canvas.Refresh(object)
	}
}

func (r *waveformRenderer) paint(size fyne.Size) {
	if size.Width <= 0 || size.Height <= 0 {
		return
	}

	r.widget.mu.RLock()
	history := append([]float64(nil), r.widget.history...)
	peak := r.widget.peak
	r.widget.mu.RUnlock()

	centerY := size.Height / 2
	leftPad := float32(6)
	usableWidth := size.Width - (leftPad * 2)
	spacing := usableWidth / float32(len(history))
	maxHalfHeight := size.Height*0.42 - 2
	peakBoost := peak * 0.12

	r.widget.base.Position1 = fyne.NewPos(0, centerY)
	r.widget.base.Position2 = fyne.NewPos(size.Width, centerY)

	for i, line := range r.widget.lines {
		amplitude := history[i]
		smoothed := amplitude + peakBoost
		height := float32(math.Max(4, float64(maxHalfHeight*float32(clamp01(smoothed)))))
		x := leftPad + spacing*float32(i) + spacing/2
		line.Position1 = fyne.NewPos(x, centerY-height)
		line.Position2 = fyne.NewPos(x, centerY+height)
		line.Refresh()
	}

	r.widget.base.Refresh()
}

func (r *waveformRenderer) BackgroundColor() color.Color {
	return color.Transparent
}
