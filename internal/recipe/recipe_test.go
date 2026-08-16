package recipe

import (
	"bytes"
	"errors"
	"io"
	"os/exec"
	"strings"
	"testing"

	"beepboop/internal/voice"
	"beepboop/internal/wav"
)

// sink collects rendered output per name instead of touching the filesystem.
type sink struct {
	files map[string]*bytes.Buffer
	err   error
}

func newSink() *sink { return &sink{files: map[string]*bytes.Buffer{}} }

type nopCloser struct{ io.Writer }

func (nopCloser) Close() error { return nil }

func (s *sink) open(name string) (io.WriteCloser, error) {
	if s.err != nil {
		return nil, s.err
	}
	buf := &bytes.Buffer{}
	s.files[name] = buf
	return nopCloser{buf}, nil
}

// decode reads back a rendered file so tests assert on audio, not bytes.
func (s *sink) decode(t *testing.T, name string) (int, []float64) {
	t.Helper()
	buf, ok := s.files[name]
	if !ok {
		t.Fatalf("no output named %q; got %v", name, s.names())
	}
	rate, samples, err := wav.ReadPCM16Mono(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("decode %q: %v", name, err)
	}
	return rate, samples
}

func (s *sink) names() []string {
	out := make([]string, 0, len(s.files))
	for name := range s.files {
		out = append(out, name)
	}
	return out
}

// fakeSpeech stands in for Piper output: full-scale-ish, but only one sample
// actually at the peak, the way real speech looks and clipped audio does not.
var fakeSpeech = []float64{0.1, -0.2, 0.5, 0.3, -0.1, 0.2, -0.3, 0.1, 0.05, -0.15}

func fakePiper(_ voice.Command, stdin io.Reader, stdout io.Writer) error {
	io.ReadAll(stdin)
	return wav.WritePCM16Mono(stdout, 22050, fakeSpeech)
}

func found(string) (string, error)   { return "/usr/bin/piper", nil }
func missing(string) (string, error) { return "", exec.ErrNotFound }

func options(s *sink) Options {
	return Options{Open: s.open, Run: fakePiper, LookPath: found}
}

// atPeak is the fraction of samples sitting at the buffer's peak, a cheap
// proxy for "this was hard clipped".
func atPeak(samples []float64) float64 {
	top := peak(samples)
	if top == 0 {
		return 0
	}
	count := 0
	for _, v := range samples {
		if v >= top-1e-4 || v <= -top+1e-4 {
			count++
		}
	}
	return float64(count) / float64(len(samples))
}

func peak(samples []float64) float64 {
	loudest := 0.0
	for _, v := range samples {
		if v > loudest {
			loudest = v
		} else if -v > loudest {
			loudest = -v
		}
	}
	return loudest
}

func TestParseRejectsInvalidJSON(t *testing.T) {
	if _, err := Parse(strings.NewReader("{nope")); err == nil {
		t.Error("err = nil, want a parse error")
	}
}

func TestRenderPresetOutputs(t *testing.T) {
	r, err := Parse(strings.NewReader(`{
		"outputs": [
			{"name": "alarm", "preset": "alarm-basic"},
			{"name": "blip", "preset": "notify-blip"}
		]
	}`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	s := newSink()
	results, err := r.Render(options(s))
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %d, want 2", len(results))
	}
	rate, samples := s.decode(t, "alarm")
	if rate != 44100 {
		t.Errorf("alarm rate = %d, want 44100", rate)
	}
	if len(samples) != 35280 {
		t.Errorf("alarm samples = %d, want 35280", len(samples))
	}
	if _, blip := s.decode(t, "blip"); len(blip) != 6174 {
		t.Errorf("blip samples = %d, want 6174", len(blip))
	}
}

func TestRenderSpokenOutputs(t *testing.T) {
	r, _ := Parse(strings.NewReader(`{
		"voice": {"model": "voice.onnx"},
		"outputs": [
			{"name": "turn", "say": "your turn"},
			{"name": "done", "say": "build finished"}
		]
	}`))

	s := newSink()
	if _, err := r.Render(options(s)); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	rate, samples := s.decode(t, "turn")
	if rate != 22050 {
		t.Errorf("rate = %d, want 22050", rate)
	}
	if len(samples) != len(fakeSpeech) {
		t.Errorf("samples = %d, want %d", len(samples), len(fakeSpeech))
	}
	if _, ok := s.files["done"]; !ok {
		t.Error("second line was not rendered")
	}
}

func TestRenderAppliesRecipeEffects(t *testing.T) {
	// Recipe-level effects apply to every output.
	r, _ := Parse(strings.NewReader(`{
		"effects": [{"type": "normalize", "peak": 0.1}],
		"outputs": [{"name": "quiet", "preset": "alarm-basic"}]
	}`))

	s := newSink()
	if _, err := r.Render(options(s)); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	_, samples := s.decode(t, "quiet")
	if got := peak(samples); got < 0.09 || got > 0.11 {
		t.Errorf("peak = %v, want ~0.1", got)
	}
}

func TestOutputEffectsOverrideRecipeEffects(t *testing.T) {
	r, _ := Parse(strings.NewReader(`{
		"effects": [{"type": "normalize", "peak": 0.1}],
		"outputs": [
			{"name": "quiet", "preset": "alarm-basic"},
			{"name": "loud", "preset": "alarm-basic",
			 "effects": [{"type": "normalize", "peak": 0.9}]}
		]
	}`))

	s := newSink()
	if _, err := r.Render(options(s)); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	_, quiet := s.decode(t, "quiet")
	_, loud := s.decode(t, "loud")
	if got := peak(quiet); got > 0.11 {
		t.Errorf("quiet peak = %v, want ~0.1", got)
	}
	if got := peak(loud); got < 0.89 {
		t.Errorf("loud peak = %v, want ~0.9", got)
	}
}

func TestEmptyOutputEffectsDisableRecipeEffects(t *testing.T) {
	// An explicit empty list means "no effects here", distinct from omitting
	// the key, which inherits. Without that distinction there is no way to
	// opt a single output out of a recipe-wide chain.
	r, _ := Parse(strings.NewReader(`{
		"effects": [{"type": "normalize", "peak": 0.1}],
		"outputs": [{"name": "raw", "preset": "alarm-basic", "effects": []}]
	}`))

	s := newSink()
	if _, err := r.Render(options(s)); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	_, samples := s.decode(t, "raw")
	if got := peak(samples); got < 0.5 {
		t.Errorf("peak = %v, want the preset's own 0.55", got)
	}
}

func TestEffectChainAppliesInOrder(t *testing.T) {
	r, _ := Parse(strings.NewReader(`{
		"outputs": [{"name": "crunch", "preset": "alarm-basic", "effects": [
			{"type": "gain", "factor": 4},
			{"type": "hardclip", "threshold": 0.3},
			{"type": "normalize", "peak": 0.8}
		]}]
	}`))

	s := newSink()
	if _, err := r.Render(options(s)); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	_, samples := s.decode(t, "crunch")
	if got := peak(samples); got < 0.79 || got > 0.81 {
		t.Errorf("peak = %v, want ~0.8", got)
	}
}

func TestAllEffectTypesResolve(t *testing.T) {
	for _, spec := range []string{
		`{"type": "gain", "factor": 1.2}`,
		`{"type": "hardclip", "threshold": 0.5}`,
		`{"type": "softclip", "drive": 3}`,
		`{"type": "fuzz", "drive": 8, "bias": 0.3}`,
		`{"type": "normalize", "peak": 0.9}`,
	} {
		r, err := Parse(strings.NewReader(`{"outputs": [{"name": "x",
			"preset": "notify-blip", "effects": [` + spec + `]}]}`))
		if err != nil {
			t.Fatalf("Parse(%s) error = %v", spec, err)
		}
		if _, err := r.Render(options(newSink())); err != nil {
			t.Errorf("Render(%s) error = %v", spec, err)
		}
	}
}

func TestUnknownEffectTypeIsAnError(t *testing.T) {
	r, _ := Parse(strings.NewReader(`{"outputs": [{"name": "x",
		"preset": "notify-blip", "effects": [{"type": "reverb"}]}]}`))

	_, err := r.Render(options(newSink()))
	if !errors.Is(err, ErrUnknownEffect) {
		t.Fatalf("err = %v, want ErrUnknownEffect", err)
	}
	if !strings.Contains(err.Error(), "reverb") {
		t.Errorf("error %q does not name the bad effect", err)
	}
}

func TestUnknownPresetIsAnError(t *testing.T) {
	r, _ := Parse(strings.NewReader(`{"outputs": [{"name": "x", "preset": "nope"}]}`))

	_, err := r.Render(options(newSink()))
	if !errors.Is(err, ErrUnknownPreset) {
		t.Errorf("err = %v, want ErrUnknownPreset", err)
	}
}

func TestOutputNeedsExactlyOneSource(t *testing.T) {
	for _, body := range []string{
		`{"name": "x"}`,
		`{"name": "x", "preset": "notify-blip", "say": "hello"}`,
	} {
		r, _ := Parse(strings.NewReader(`{"outputs": [` + body + `]}`))
		if _, err := r.Render(options(newSink())); !errors.Is(err, ErrBadSource) {
			t.Errorf("%s: err = %v, want ErrBadSource", body, err)
		}
	}
}

func TestOutputNeedsAName(t *testing.T) {
	r, _ := Parse(strings.NewReader(`{"outputs": [{"preset": "notify-blip"}]}`))

	if _, err := r.Render(options(newSink())); !errors.Is(err, ErrNoName) {
		t.Errorf("err = %v, want ErrNoName", err)
	}
}

func TestDuplicateNamesAreRejected(t *testing.T) {
	// Two outputs with one name would silently clobber a file, so this is
	// caught before anything is written.
	r, _ := Parse(strings.NewReader(`{"outputs": [
		{"name": "dup", "preset": "notify-blip"},
		{"name": "dup", "preset": "notify-chime"}
	]}`))

	s := newSink()
	_, err := r.Render(options(s))
	if !errors.Is(err, ErrDuplicateName) {
		t.Fatalf("err = %v, want ErrDuplicateName", err)
	}
	if len(s.files) != 0 {
		t.Errorf("wrote %v before failing, want nothing", s.names())
	}
}

func TestMissingPiperFailsBeforeWritingAnything(t *testing.T) {
	// A batch that cannot finish must not leave a half-rendered directory
	// of stale sounds and empty label files behind.
	r, _ := Parse(strings.NewReader(`{
		"voice": {"model": "voice.onnx"},
		"labels": true,
		"outputs": [
			{"name": "a", "preset": "notify-blip"},
			{"name": "b", "preset": "notify-chime"}
		]
	}`))

	s := newSink()
	_, err := r.Render(Options{Open: s.open, Run: fakePiper, LookPath: missing})
	if !errors.Is(err, voice.ErrPiperNotFound) {
		t.Fatalf("err = %v, want ErrPiperNotFound", err)
	}
	if len(s.files) != 0 {
		t.Errorf("wrote %v before failing, want nothing", s.names())
	}
}

func TestMissingPiperIsFineWhenNoVoiceIsNeeded(t *testing.T) {
	// Preset-only recipes must keep working on machines without Piper.
	r, _ := Parse(strings.NewReader(
		`{"outputs": [{"name": "a", "preset": "notify-blip"}]}`))

	s := newSink()
	if _, err := r.Render(Options{Open: s.open, LookPath: missing}); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if len(s.files) != 1 {
		t.Errorf("files = %v, want one", s.names())
	}
}

func TestEmptyRecipeIsAnError(t *testing.T) {
	r, _ := Parse(strings.NewReader(`{"outputs": []}`))

	if _, err := r.Render(options(newSink())); !errors.Is(err, ErrNoOutputs) {
		t.Errorf("err = %v, want ErrNoOutputs", err)
	}
}

func TestSpokenOutputNeedsAVoiceModel(t *testing.T) {
	r, _ := Parse(strings.NewReader(`{"outputs": [{"name": "x", "say": "hello"}]}`))

	_, err := r.Render(options(newSink()))
	if err == nil || !strings.Contains(err.Error(), "model") {
		t.Errorf("err = %v, want a message about the missing voice model", err)
	}
}

func TestRenderErrorNamesTheOutput(t *testing.T) {
	r, _ := Parse(strings.NewReader(`{"outputs": [
		{"name": "fine", "preset": "notify-blip"},
		{"name": "broken", "preset": "nope"}
	]}`))

	_, err := r.Render(options(newSink()))
	if err == nil || !strings.Contains(err.Error(), "broken") {
		t.Errorf("err = %v, want the failing output named", err)
	}
}

func TestOpenErrorPropagates(t *testing.T) {
	r, _ := Parse(strings.NewReader(`{"outputs": [{"name": "x", "preset": "notify-blip"}]}`))
	s := newSink()
	s.err = errors.New("disk full")

	if _, err := r.Render(options(s)); err == nil {
		t.Error("err = nil, want the open error")
	}
}

func TestLabelsRenderBesideEachSound(t *testing.T) {
	r, _ := Parse(strings.NewReader(`{
		"voice": {"model": "voice.onnx"},
		"labels": true,
		"outputs": [
			{"name": "turn-ready", "preset": "turn-ready"},
			{"name": "notify-blip", "preset": "notify-blip"}
		]
	}`))

	s := newSink()
	if _, err := r.Render(options(s)); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	for _, name := range []string{
		"turn-ready", "turn-ready" + LabelSuffix,
		"notify-blip", "notify-blip" + LabelSuffix,
	} {
		if _, ok := s.files[name]; !ok {
			t.Errorf("missing %q; got %v", name, s.names())
		}
	}
}

func TestLabelSpeaksTheSoundName(t *testing.T) {
	// Hyphens are word separators in preset names, not something to read
	// aloud, so "turn-ready" is spoken as "turn ready".
	var spoken []string
	capture := func(cmd voice.Command, stdin io.Reader, stdout io.Writer) error {
		text, _ := io.ReadAll(stdin)
		spoken = append(spoken, strings.TrimSpace(string(text)))
		return wav.WritePCM16Mono(stdout, 22050, []float64{0.1})
	}

	r, _ := Parse(strings.NewReader(`{
		"voice": {"model": "voice.onnx"},
		"labels": true,
		"outputs": [{"name": "turn-ready_soft", "preset": "turn-ready-soft"}]
	}`))

	if _, err := r.Render(Options{Open: newSink().open, Run: capture, LookPath: found}); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if len(spoken) != 1 || spoken[0] != "turn ready soft" {
		t.Errorf("spoken = %v, want [\"turn ready soft\"]", spoken)
	}
}

func TestLabelTextCanBeOverridden(t *testing.T) {
	var spoken []string
	capture := func(cmd voice.Command, stdin io.Reader, stdout io.Writer) error {
		text, _ := io.ReadAll(stdin)
		spoken = append(spoken, strings.TrimSpace(string(text)))
		return wav.WritePCM16Mono(stdout, 22050, []float64{0.1})
	}

	r, _ := Parse(strings.NewReader(`{
		"voice": {"model": "voice.onnx"},
		"labels": true,
		"outputs": [{"name": "tr1", "preset": "turn-ready", "label": "your turn is ready"}]
	}`))

	if _, err := r.Render(Options{Open: newSink().open, Run: capture, LookPath: found}); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if len(spoken) != 1 || spoken[0] != "your turn is ready" {
		t.Errorf("spoken = %v, want the override", spoken)
	}
}

func TestLabelsSkipTheEffectsChain(t *testing.T) {
	// A label is a spoken index entry. Running it through the recipe's
	// distortion chain would defeat the point of being able to identify the
	// file by ear, so only level matching is applied.
	r, _ := Parse(strings.NewReader(`{
		"voice": {"model": "voice.onnx"},
		"labels": true,
		"effects": [
			{"type": "gain", "factor": 20},
			{"type": "hardclip", "threshold": 0.4}
		],
		"outputs": [{"name": "x", "preset": "notify-blip"}]
	}`))

	s := newSink()
	if _, err := r.Render(options(s)); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	// A clipped square wave pins most samples at the threshold; clean speech
	// does not. Compare how much of each file sits at its own peak.
	_, primary := s.decode(t, "x")
	_, label := s.decode(t, "x"+LabelSuffix)
	if atPeak(primary) < 0.5 {
		t.Fatalf("primary is not clipped (%.2f at peak); test is not measuring anything", atPeak(primary))
	}
	if atPeak(label) > 0.5 {
		t.Errorf("label looks clipped: %.2f of samples at peak", atPeak(label))
	}
}

func TestLabelsMatchTheirSoundsLevel(t *testing.T) {
	// Piper normalizes to full scale, which would make every label roughly
	// three times louder than the gentle sound it names. Matching the peak
	// keeps a sound and its label feeling like one unit.
	r, _ := Parse(strings.NewReader(`{
		"voice": {"model": "voice.onnx"},
		"labels": true,
		"outputs": [
			{"name": "quiet", "preset": "notify-blip",
			 "effects": [{"type": "normalize", "peak": 0.2}]},
			{"name": "loud", "preset": "alarm-urgent",
			 "effects": [{"type": "normalize", "peak": 0.9}]}
		]
	}`))

	s := newSink()
	if _, err := r.Render(options(s)); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	for _, tc := range []struct {
		name string
		want float64
	}{{"quiet", 0.2}, {"loud", 0.9}} {
		_, label := s.decode(t, tc.name+LabelSuffix)
		if got := peak(label); got < tc.want-0.01 || got > tc.want+0.01 {
			t.Errorf("%s label peak = %v, want ~%v", tc.name, got, tc.want)
		}
	}
}

func TestSilentSoundLeavesItsLabelAudible(t *testing.T) {
	// Matching a silent sound would render a silent label, which is useless
	// for identifying the file. Leave the voice at its own level instead.
	r, _ := Parse(strings.NewReader(`{
		"voice": {"model": "voice.onnx"},
		"labels": true,
		"outputs": [{"name": "hush", "preset": "notify-blip",
			"effects": [{"type": "gain", "factor": 0}]}]
	}`))

	s := newSink()
	if _, err := r.Render(options(s)); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if _, primary := s.decode(t, "hush"); peak(primary) != 0 {
		t.Fatalf("primary peak = %v, want silence", peak(primary))
	}
	if _, label := s.decode(t, "hush"+LabelSuffix); peak(label) < 0.49 {
		t.Errorf("label peak = %v, want the voice's own 0.5", peak(label))
	}
}

func TestLabelsOffByDefault(t *testing.T) {
	r, _ := Parse(strings.NewReader(`{
		"voice": {"model": "voice.onnx"},
		"outputs": [{"name": "x", "preset": "notify-blip"}]
	}`))

	s := newSink()
	if _, err := r.Render(options(s)); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if _, ok := s.files["x"+LabelSuffix]; ok {
		t.Error("rendered a label without labels enabled")
	}
}

func TestPerOutputLabelOptOut(t *testing.T) {
	r, _ := Parse(strings.NewReader(`{
		"voice": {"model": "voice.onnx"},
		"labels": true,
		"outputs": [
			{"name": "keep", "preset": "notify-blip"},
			{"name": "skip", "preset": "notify-chime", "label": ""}
		]
	}`))

	s := newSink()
	if _, err := r.Render(options(s)); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if _, ok := s.files["keep"+LabelSuffix]; !ok {
		t.Error("missing label for keep")
	}
	if _, ok := s.files["skip"+LabelSuffix]; ok {
		t.Error("rendered a label for an output that opted out")
	}
}

func TestLabelsNeedAVoiceModel(t *testing.T) {
	r, _ := Parse(strings.NewReader(`{
		"labels": true,
		"outputs": [{"name": "x", "preset": "notify-blip"}]
	}`))

	_, err := r.Render(options(newSink()))
	if err == nil || !strings.Contains(err.Error(), "model") {
		t.Errorf("err = %v, want a message about the missing voice model", err)
	}
}

func TestLabelIsReportedInResults(t *testing.T) {
	r, _ := Parse(strings.NewReader(`{
		"voice": {"model": "voice.onnx"},
		"labels": true,
		"outputs": [{"name": "x", "preset": "notify-blip"}]
	}`))

	results, err := r.Render(options(newSink()))
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}
	if results[0].Label != "x"+LabelSuffix {
		t.Errorf("Label = %q, want %q", results[0].Label, "x"+LabelSuffix)
	}
}

func TestRenderIsDeterministic(t *testing.T) {
	source := `{"effects": [{"type": "fuzz", "drive": 6, "bias": 0.2}],
		"outputs": [{"name": "x", "preset": "alarm-basic"}]}`

	build := func() []byte {
		r, _ := Parse(strings.NewReader(source))
		s := newSink()
		if _, err := r.Render(options(s)); err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		return s.files["x"].Bytes()
	}
	if !bytes.Equal(build(), build()) {
		t.Error("recipe rendering is not byte-identical across runs")
	}
}
