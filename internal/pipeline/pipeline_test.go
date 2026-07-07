package pipeline

import (
	"bytes"
	"errors"
	"testing"

	"beepboop/internal/audio"
)

// scaleEffect multiplies every sample by factor. It is intentionally
// non-commutative with biasEffect so ordering tests are meaningful.
type scaleEffect struct{ factor float64 }

func (e scaleEffect) Apply(s audio.Sound) audio.Sound {
	in := s.Samples()
	out := make([]float64, len(in))
	for i, v := range in {
		out[i] = v * e.factor
	}
	return audio.NewSound(s.SampleRate(), out)
}

// biasEffect adds offset to every sample.
type biasEffect struct{ offset float64 }

func (e biasEffect) Apply(s audio.Sound) audio.Sound {
	in := s.Samples()
	out := make([]float64, len(in))
	for i, v := range in {
		out[i] = v + e.offset
	}
	return audio.NewSound(s.SampleRate(), out)
}

// captureExporter records the final sound instead of writing anywhere.
type captureExporter struct {
	sound audio.Sound
	err   error
}

func (e *captureExporter) Export(s audio.Sound) error {
	e.sound = s
	return e.err
}

type erroringSource struct{ err error }

func (s erroringSource) Render() (audio.Sound, error) { return nil, s.err }

func newSource(rate int, samples ...float64) Source {
	return StaticSource{Sound: audio.NewSound(rate, samples)}
}

func TestRunAppliesEffectsInOrder(t *testing.T) {
	cap := &captureExporter{}
	// (x * 2) + 1 with scale-then-bias.
	p := Pipeline{
		Source:   newSource(8000, 0.1, 0.2),
		Effects:  []Effect{scaleEffect{factor: 2}, biasEffect{offset: 1}},
		Exporter: cap,
	}
	if err := p.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	got := cap.sound.Samples()
	want := []float64{1.2, 1.4}
	if len(got) != len(want) {
		t.Fatalf("sample count = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("sample[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestRunEffectOrderMatters(t *testing.T) {
	// bias-then-scale: (x + 1) * 2, distinct from scale-then-bias above.
	cap := &captureExporter{}
	p := Pipeline{
		Source:   newSource(8000, 0.1, 0.2),
		Effects:  []Effect{biasEffect{offset: 1}, scaleEffect{factor: 2}},
		Exporter: cap,
	}
	if err := p.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	got := cap.sound.Samples()
	want := []float64{2.2, 2.4}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("sample[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestRunIsDeterministic(t *testing.T) {
	build := func() []float64 {
		cap := &captureExporter{}
		p := Pipeline{
			Source:   newSource(8000, 0.3, -0.4, 0.5),
			Effects:  []Effect{scaleEffect{factor: 1.5}},
			Exporter: cap,
		}
		if err := p.Run(); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		return cap.sound.Samples()
	}
	a, b := build(), build()
	if len(a) != len(b) {
		t.Fatalf("length mismatch %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("nondeterministic at %d: %v vs %v", i, a[i], b[i])
		}
	}
}

func TestRunNilEffectsPassThrough(t *testing.T) {
	cap := &captureExporter{}
	p := Pipeline{
		Source:   newSource(8000, 0.1, 0.2),
		Effects:  []Effect{nil},
		Exporter: cap,
	}
	if err := p.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := cap.sound.Samples(); got[0] != 0.1 || got[1] != 0.2 {
		t.Errorf("nil effect changed samples: %v", got)
	}
}

func TestRunMissingSource(t *testing.T) {
	err := Pipeline{Exporter: &captureExporter{}}.Run()
	if !errors.Is(err, ErrNoSource) {
		t.Errorf("err = %v, want ErrNoSource", err)
	}
}

func TestRunMissingExporter(t *testing.T) {
	err := Pipeline{Source: newSource(8000, 0.1)}.Run()
	if !errors.Is(err, ErrNoExporter) {
		t.Errorf("err = %v, want ErrNoExporter", err)
	}
}

func TestRunSourceErrorPropagates(t *testing.T) {
	sentinel := errors.New("boom")
	err := Pipeline{Source: erroringSource{err: sentinel}, Exporter: &captureExporter{}}.Run()
	if !errors.Is(err, sentinel) {
		t.Errorf("err = %v, want %v", err, sentinel)
	}
}

func TestWAVExporterWritesValidHeader(t *testing.T) {
	var buf bytes.Buffer
	p := Pipeline{
		Source:   newSource(8000, 0.0, 0.5, -0.5),
		Exporter: WAVExporter{W: &buf},
	}
	if err := p.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	out := buf.Bytes()
	if len(out) < 44 {
		t.Fatalf("output too short: %d bytes", len(out))
	}
	if string(out[0:4]) != "RIFF" || string(out[8:12]) != "WAVE" {
		t.Errorf("bad WAV header: %q", out[0:12])
	}
	// 44-byte header + 3 samples * 2 bytes.
	if want := 44 + 3*2; len(out) != want {
		t.Errorf("output size = %d, want %d", len(out), want)
	}
}
