package voice

import (
	"bytes"
	"errors"
	"io"
	"os/exec"
	"strings"
	"testing"

	"beepboop/internal/pipeline"
	"beepboop/internal/wav"
)

// Piper's whole reason to exist is being a pipeline Source.
var _ pipeline.Source = Piper{}

// fakeRun records what it was asked to execute and emits a canned WAV, so
// tests cover the adapter without Piper installed.
type fakeRun struct {
	gotCommand Command
	gotStdin   string
	rate       int
	samples    []float64
	err        error
}

func (f *fakeRun) run(cmd Command, stdin io.Reader, stdout io.Writer) error {
	text, _ := io.ReadAll(stdin)
	f.gotCommand, f.gotStdin = cmd, string(text)
	if f.err != nil {
		return f.err
	}
	return wav.WritePCM16Mono(stdout, f.rate, f.samples)
}

func found(string) (string, error)   { return "/usr/bin/piper", nil }
func missing(string) (string, error) { return "", exec.ErrNotFound }

func TestPiperRendersSpokenAudio(t *testing.T) {
	fake := &fakeRun{rate: 22050, samples: []float64{0, 0.5, -0.5}}
	p := Piper{
		Model:    "/voices/en_US-lessac-medium.onnx",
		Text:     "your turn",
		Run:      fake.run,
		LookPath: found,
	}

	sound, err := p.Render()
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if got := sound.SampleRate(); got != 22050 {
		t.Errorf("SampleRate() = %d, want 22050", got)
	}
	if got := len(sound.Samples()); got != 3 {
		t.Errorf("sample count = %d, want 3", got)
	}
}

func TestPiperPassesTextOnStdin(t *testing.T) {
	// Text goes over stdin, never into argv: lines are arbitrary user
	// content and must not be parsed as flags or hit an argv length cap.
	fake := &fakeRun{rate: 22050, samples: []float64{0.1}}
	p := Piper{Model: "voice.onnx", Text: "hello there", Run: fake.run, LookPath: found}

	if _, err := p.Render(); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if fake.gotStdin != "hello there\n" {
		t.Errorf("stdin = %q, want %q", fake.gotStdin, "hello there\n")
	}
	for _, arg := range fake.gotCommand.Args {
		if strings.Contains(arg, "hello there") {
			t.Errorf("text leaked into argv: %v", fake.gotCommand.Args)
		}
	}
}

func TestPiperBuildsDefaultCommand(t *testing.T) {
	fake := &fakeRun{rate: 22050, samples: []float64{0.1}}
	p := Piper{Model: "voice.onnx", Text: "hi", Run: fake.run, LookPath: found}

	if _, err := p.Render(); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if fake.gotCommand.Name != "piper" {
		t.Errorf("command = %q, want %q", fake.gotCommand.Name, "piper")
	}
	want := []string{"--model", "voice.onnx", "--output_file", "-"}
	if got := strings.Join(fake.gotCommand.Args, " "); got != strings.Join(want, " ") {
		t.Errorf("args = %v, want %v", fake.gotCommand.Args, want)
	}
}

func TestPiperHonorsBinaryAndArgsOverride(t *testing.T) {
	// Piper's flags have shifted between releases, so callers can override
	// the whole argv rather than being stuck on our default.
	fake := &fakeRun{rate: 16000, samples: []float64{0.1}}
	p := Piper{
		Binary:   "piper-tts",
		Model:    "voice.onnx",
		Text:     "hi",
		Args:     []string{"-m", "voice.onnx", "-f", "-"},
		Run:      fake.run,
		LookPath: found,
	}

	if _, err := p.Render(); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if fake.gotCommand.Name != "piper-tts" {
		t.Errorf("command = %q, want %q", fake.gotCommand.Name, "piper-tts")
	}
	if got := strings.Join(fake.gotCommand.Args, " "); got != "-m voice.onnx -f -" {
		t.Errorf("args = %v, want the override", fake.gotCommand.Args)
	}
}

func TestPiperMissingBinaryIsClear(t *testing.T) {
	p := Piper{Model: "voice.onnx", Text: "hi", Run: (&fakeRun{}).run, LookPath: missing}

	_, err := p.Render()
	if !errors.Is(err, ErrPiperNotFound) {
		t.Fatalf("err = %v, want ErrPiperNotFound", err)
	}
	if !strings.Contains(err.Error(), "piper") {
		t.Errorf("error %q does not name the missing tool", err)
	}
}

func TestPiperMissingModel(t *testing.T) {
	p := Piper{Text: "hi", Run: (&fakeRun{}).run, LookPath: found}

	if _, err := p.Render(); !errors.Is(err, ErrNoModel) {
		t.Errorf("err = %v, want ErrNoModel", err)
	}
}

func TestPiperMissingText(t *testing.T) {
	// Empty text would make Piper emit an empty or hanging stream; catch it
	// here where the message can say what is actually wrong.
	p := Piper{Model: "voice.onnx", Text: "   ", Run: (&fakeRun{}).run, LookPath: found}

	if _, err := p.Render(); !errors.Is(err, ErrNoText) {
		t.Errorf("err = %v, want ErrNoText", err)
	}
}

func TestPiperRunErrorPropagates(t *testing.T) {
	sentinel := errors.New("exit status 1")
	p := Piper{
		Model:    "voice.onnx",
		Text:     "hi",
		Run:      (&fakeRun{err: sentinel}).run,
		LookPath: found,
	}

	if _, err := p.Render(); !errors.Is(err, sentinel) {
		t.Errorf("err = %v, want %v", err, sentinel)
	}
}

func TestPiperGarbageOutputIsAnError(t *testing.T) {
	garbage := func(Command, io.Reader, io.Writer) error { return nil }
	p := Piper{Model: "voice.onnx", Text: "hi", Run: garbage, LookPath: found}

	if _, err := p.Render(); err == nil {
		t.Error("err = nil, want a decode error for empty output")
	}
}

func TestPiperFeedsThePipeline(t *testing.T) {
	// End to end through the spine: Piper as Source, WAV back out.
	fake := &fakeRun{rate: 22050, samples: []float64{0.2, -0.4}}
	var out bytes.Buffer
	p := pipeline.Pipeline{
		Source:   Piper{Model: "voice.onnx", Text: "shipping", Run: fake.run, LookPath: found},
		Exporter: pipeline.WAVExporter{W: &out},
	}

	if err := p.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := out.Bytes(); string(got[0:4]) != "RIFF" {
		t.Errorf("bad WAV header: %q", got[0:4])
	}
}
