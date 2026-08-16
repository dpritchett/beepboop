// Package recipe drives the pipeline from a checked-in JSON file, so a set of
// sounds and the effects chains applied to them are source you can diff rather
// than a shell history you have to remember.
//
// JSON rather than YAML because the core stays standard-library only. A recipe
// names its outputs; where each one lands is the caller's business, injected as
// an Open function, which keeps this package off the filesystem and testable.
package recipe

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"

	"beepboop/internal/audio"
	"beepboop/internal/effects"
	"beepboop/internal/pipeline"
	"beepboop/internal/presets"
	"beepboop/internal/voice"
)

// LabelSuffix is appended to an output's name to form its spoken companion
// file. It sits directly beside the primary sound and sorts next to it, so a
// directory listing pairs them without a naming convention to memorize.
const LabelSuffix = ".label"

var (
	ErrNoOutputs     = errors.New("recipe: no outputs")
	ErrNoName        = errors.New("recipe: output has no name")
	ErrDuplicateName = errors.New("recipe: duplicate output name")
	ErrBadSource     = errors.New("recipe: output needs exactly one of preset or say")
	ErrUnknownPreset = errors.New("recipe: unknown preset")
	ErrUnknownEffect = errors.New("recipe: unknown effect type")
	ErrUnknownShape  = errors.New("recipe: unknown shape type")
)

// Recipe is a batch of sounds to render with a shared voice and effects chain.
type Recipe struct {
	Voice VoiceSpec `json:"voice"`
	// Effects applies to every output that does not set its own.
	Effects []EffectSpec `json:"effects"`
	// Labels renders a spoken companion file for every output.
	Labels  bool     `json:"labels"`
	Outputs []Output `json:"outputs"`
}

type VoiceSpec struct {
	Binary string   `json:"binary"`
	Model  string   `json:"model"`
	Args   []string `json:"args"`
}

// Output is one rendered sound: a name, a source, and optional effects.
type Output struct {
	Name string `json:"name"`
	// Exactly one source is required. Preset names a synthetic sound, Say is
	// a line for the voice, Loop defines a repeating bed inline, Sequence
	// places notes in time, and Layers mixes several of those together.
	Preset   string        `json:"preset"`
	Say      string        `json:"say"`
	Loop     *LoopSpec     `json:"loop"`
	Sequence *SequenceSpec `json:"sequence"`
	Layers   []LayerSpec   `json:"layers"`
	// Effects overrides the recipe chain when present. An explicit empty
	// list means no effects, which is how a single output opts out.
	Effects []EffectSpec `json:"effects"`
	// Label overrides the spoken label text. An explicit empty string opts
	// this output out of labeling.
	Label *string `json:"label"`
}

// LoopSpec defines a seamlessly repeating sound inline in the recipe, so
// tuning a turbine or a drone is an edit and a rebake rather than a Go change
// and a recompile. Fields mirror audio.LoopSpec.
type LoopSpec struct {
	SampleRate int           `json:"sample_rate"`
	Duration   float64       `json:"duration"`
	Partials   []PartialSpec `json:"partials"`
	// Shapes expand to harmonic stacks and are added to Partials. Several
	// shapes at once is how a chord is written.
	Shapes     []ShapeSpec `json:"shapes"`
	Noise      float64     `json:"noise"`
	NoiseTone  float64     `json:"noise_tone"`
	NoisePoles int         `json:"noise_poles"`
	Seed       int64       `json:"seed"`
}

type PartialSpec struct {
	Frequency float64 `json:"frequency"`
	Gain      float64 `json:"gain"`
	Phase     float64 `json:"phase"`
}

// ShapeSpec is one classic oscillator shape as a harmonic stack.
type ShapeSpec struct {
	// Type is saw, square, or triangle.
	Type        string  `json:"type"`
	Fundamental float64 `json:"fundamental"`
	// Limit caps the stack, doubling as a brightness control. Defaults to
	// Nyquist for the loop's sample rate.
	Limit float64 `json:"limit"`
	// Gain scales the whole stack, which is how voices are balanced against
	// each other in a chord. Defaults to 1.
	Gain float64 `json:"gain"`
}

func (s ShapeSpec) build(sampleRate int) ([]audio.Partial, error) {
	limit := s.Limit
	if limit <= 0 {
		limit = float64(sampleRate) / 2
	}
	var partials []audio.Partial
	switch strings.ToLower(s.Type) {
	case "saw":
		partials = audio.SawPartials(s.Fundamental, limit)
	case "square":
		partials = audio.SquarePartials(s.Fundamental, limit)
	case "triangle":
		partials = audio.TrianglePartials(s.Fundamental, limit)
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownShape, s.Type)
	}
	gain := s.Gain
	if gain == 0 {
		gain = 1
	}
	for i := range partials {
		partials[i].Gain *= gain
	}
	return partials, nil
}

func (s LoopSpec) build() (audio.Sound, error) {
	partials := make([]audio.Partial, 0, len(s.Partials))
	for _, p := range s.Partials {
		partials = append(partials, audio.Partial{
			Frequency: p.Frequency,
			Gain:      p.Gain,
			Phase:     p.Phase,
		})
	}
	for _, shape := range s.Shapes {
		expanded, err := shape.build(s.SampleRate)
		if err != nil {
			return nil, err
		}
		partials = append(partials, expanded...)
	}
	return audio.Loop(audio.LoopSpec{
		SampleRate: s.SampleRate,
		Duration:   s.Duration,
		Partials:   partials,
		Noise:      s.Noise,
		NoiseTone:  s.NoiseTone,
		NoisePoles: s.NoisePoles,
		Seed:       s.Seed,
	}), nil
}

// SequenceSpec places notes on a loop-length grid. Anything overhanging the
// end wraps to the beginning, so the pattern repeats cleanly.
type SequenceSpec struct {
	SampleRate int        `json:"sample_rate"`
	Duration   float64    `json:"duration"`
	Notes      []NoteSpec `json:"notes"`
}

// NoteSpec is one note, or a run of evenly spaced ones.
type NoteSpec struct {
	Start     float64 `json:"start"`
	Duration  float64 `json:"duration"`
	Frequency float64 `json:"frequency"`
	// Shape gives the note a timbre: saw, square, triangle, or sine
	// (the default). Limit caps the harmonic stack.
	Shape string  `json:"shape"`
	Limit float64 `json:"limit"`
	Gain  float64 `json:"gain"`
	// Attack defaults to 5ms; Decay defaults to 5, higher being pluckier.
	Attack float64 `json:"attack"`
	Decay  float64 `json:"decay"`
	// Repeat and Interval expand one entry into a run of notes, which is
	// what keeps a pattern from being hundreds of lines of JSON.
	Repeat   int     `json:"repeat"`
	Interval float64 `json:"interval"`
}

func (n NoteSpec) build(sampleRate int) ([]audio.Note, error) {
	limit := n.Limit
	if limit <= 0 {
		limit = float64(sampleRate) / 2
	}
	var partials []audio.Partial
	switch strings.ToLower(n.Shape) {
	case "", "sine":
		partials = []audio.Partial{{Frequency: n.Frequency, Gain: 1}}
	case "saw":
		partials = audio.SawPartials(n.Frequency, limit)
	case "square":
		partials = audio.SquarePartials(n.Frequency, limit)
	case "triangle":
		partials = audio.TrianglePartials(n.Frequency, limit)
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownShape, n.Shape)
	}

	gain := n.Gain
	if gain == 0 {
		gain = 1
	}
	repeat := n.Repeat
	if repeat < 1 {
		repeat = 1
	}

	notes := make([]audio.Note, 0, repeat)
	for i := 0; i < repeat; i++ {
		notes = append(notes, audio.Note{
			Start:    n.Start + float64(i)*n.Interval,
			Duration: n.Duration,
			Partials: partials,
			Gain:     gain,
			Attack:   n.Attack,
			Decay:    n.Decay,
		})
	}
	return notes, nil
}

func (s SequenceSpec) build() (audio.Sound, error) {
	var notes []audio.Note
	for _, spec := range s.Notes {
		built, err := spec.build(s.SampleRate)
		if err != nil {
			return nil, err
		}
		notes = append(notes, built...)
	}
	return audio.Sequence(audio.SequenceSpec{
		SampleRate: s.SampleRate,
		Duration:   s.Duration,
		Notes:      notes,
	}), nil
}

// LayerSpec is one voice in a mix: a bed or a pattern, at a relative level.
type LayerSpec struct {
	Loop     *LoopSpec     `json:"loop"`
	Sequence *SequenceSpec `json:"sequence"`
	Gain     float64       `json:"gain"`
}

func (l LayerSpec) build() (audio.Sound, float64, error) {
	gain := l.Gain
	if gain == 0 {
		gain = 1
	}
	switch {
	case l.Loop != nil && l.Sequence != nil:
		return nil, 0, errors.New("recipe: layer sets both loop and sequence")
	case l.Loop != nil:
		sound, err := l.Loop.build()
		return sound, gain, err
	case l.Sequence != nil:
		sound, err := l.Sequence.build()
		return sound, gain, err
	default:
		return nil, 0, errors.New("recipe: layer has no loop or sequence")
	}
}

// mixLayers sums layers at their gains. Layers are expected to share a sample
// rate and length; the longest wins and shorter ones simply stop, which is
// what a one-shot over a bed should do.
func mixLayers(layers []LayerSpec) (audio.Sound, error) {
	rate, longest := 0, 0
	sounds := make([]audio.Sound, 0, len(layers))
	gains := make([]float64, 0, len(layers))

	for _, layer := range layers {
		sound, gain, err := layer.build()
		if err != nil {
			return nil, err
		}
		if rate == 0 {
			rate = sound.SampleRate()
		} else if sound.SampleRate() != rate {
			return nil, fmt.Errorf("recipe: layer sample rate %d does not match %d",
				sound.SampleRate(), rate)
		}
		if n := len(sound.Samples()); n > longest {
			longest = n
		}
		sounds = append(sounds, sound)
		gains = append(gains, gain)
	}

	mixed := make([]float64, longest)
	for i, sound := range sounds {
		for j, v := range sound.Samples() {
			mixed[j] += v * gains[i]
		}
	}
	return audio.NewSound(rate, mixed), nil
}

// EffectSpec is a tagged union over the effects package. Every field is
// optional; only the ones the named type reads are used.
type EffectSpec struct {
	Type      string  `json:"type"`
	Factor    float64 `json:"factor"`
	Threshold float64 `json:"threshold"`
	Drive     float64 `json:"drive"`
	Bias      float64 `json:"bias"`
	Peak      float64 `json:"peak"`
	Cutoff    float64 `json:"cutoff"`
	Poles     int     `json:"poles"`
}

// Options injects everything ambient: where output goes and how the voice
// runs. Open receives a bare name (no extension); mapping that to a path is
// the caller's job.
type Options struct {
	Open     func(name string) (io.WriteCloser, error)
	Run      voice.Runner
	LookPath voice.LookPath
}

// Result records what one output produced. Label is empty when none was made.
type Result struct {
	Name  string
	Label string
}

func Parse(r io.Reader) (Recipe, error) {
	var recipe Recipe
	if err := json.NewDecoder(r).Decode(&recipe); err != nil {
		return Recipe{}, fmt.Errorf("recipe: parse: %w", err)
	}
	return recipe, nil
}

// Render writes every output, plus a spoken label each when Labels is set.
//
// The whole recipe is validated before anything is written: a typo in the last
// output should not leave half a batch of stale files behind.
func (r Recipe) Render(opts Options) ([]Result, error) {
	if err := r.validate(opts); err != nil {
		return nil, err
	}
	if opts.Open == nil {
		return nil, errors.New("recipe: no output opener configured")
	}

	results := make([]Result, 0, len(r.Outputs))
	for _, out := range r.Outputs {
		result, err := r.renderOutput(out, opts)
		if err != nil {
			return nil, fmt.Errorf("recipe: output %q: %w", out.Name, err)
		}
		results = append(results, result)
	}
	return results, nil
}

func (r Recipe) validate(opts Options) error {
	if len(r.Outputs) == 0 {
		return ErrNoOutputs
	}
	needsVoice := false
	seen := make(map[string]bool, len(r.Outputs))
	for _, out := range r.Outputs {
		if out.Name == "" {
			return ErrNoName
		}
		if seen[out.Name] {
			return fmt.Errorf("%w: %q", ErrDuplicateName, out.Name)
		}
		seen[out.Name] = true

		sources := 0
		for _, set := range []bool{
			out.Preset != "", out.Say != "", out.Loop != nil,
			out.Sequence != nil, len(out.Layers) > 0,
		} {
			if set {
				sources++
			}
		}
		if sources != 1 {
			return fmt.Errorf("%w: %q", ErrBadSource, out.Name)
		}
		if out.Say != "" {
			if r.Voice.Model == "" {
				return fmt.Errorf("recipe: output %q speaks but no voice model is configured", out.Name)
			}
			needsVoice = true
		}
		if r.labelText(out) != "" {
			if r.Voice.Model == "" {
				return fmt.Errorf("recipe: output %q needs a label but no voice model is configured", out.Name)
			}
			needsVoice = true
		}
		for _, spec := range r.effectsFor(out) {
			if _, err := spec.build(); err != nil {
				return fmt.Errorf("recipe: output %q: %w", out.Name, err)
			}
		}
	}

	// Probe Piper once up front rather than discovering it is missing on the
	// first spoken output, which would leave a half-rendered batch on disk.
	if needsVoice {
		if err := r.speak("probe", opts).Available(); err != nil {
			return err
		}
	}
	return nil
}

func (r Recipe) renderOutput(out Output, opts Options) (Result, error) {
	source, err := r.sourceFor(out, opts)
	if err != nil {
		return Result{}, err
	}
	chain, err := buildChain(r.effectsFor(out))
	if err != nil {
		return Result{}, err
	}
	sound, err := renderSound(source, chain)
	if err != nil {
		return Result{}, err
	}
	if err := writeSound(opts, out.Name, sound); err != nil {
		return Result{}, err
	}

	result := Result{Name: out.Name}
	if text := r.labelText(out); text != "" {
		label, err := r.renderLabel(text, peakOf(sound), opts)
		if err != nil {
			return Result{}, fmt.Errorf("label: %w", err)
		}
		name := out.Name + LabelSuffix
		if err := writeSound(opts, name, label); err != nil {
			return Result{}, fmt.Errorf("label: %w", err)
		}
		result.Label = name
	}
	return result, nil
}

// renderLabel speaks text at the same peak as the sound it labels.
//
// Labels skip the recipe's effects chain: a spoken name run through a fuzz
// pedal cannot do its job. But level is not decoration. Piper normalizes to
// full scale, so an unmatched label lands roughly three times louder than a
// gentle notification boop, which is startling in exactly the situation these
// sounds exist to stay calm in.
//
// A silent sound is left alone rather than matched, since a silent label
// identifies nothing.
func (r Recipe) renderLabel(text string, level float64, opts Options) (audio.Sound, error) {
	var chain []pipeline.Effect
	if level > 0 {
		chain = []pipeline.Effect{effects.Normalize{Peak: level}}
	}
	return renderSound(r.speak(text, opts), chain)
}

// captureSound is an Exporter that keeps the finished sound so callers can
// measure it before writing.
type captureSound struct{ sound audio.Sound }

func (c *captureSound) Export(s audio.Sound) error {
	c.sound = s
	return nil
}

func renderSound(source pipeline.Source, chain []pipeline.Effect) (audio.Sound, error) {
	capture := &captureSound{}
	err := pipeline.Pipeline{
		Source:   source,
		Effects:  chain,
		Exporter: capture,
	}.Run()
	if err != nil {
		return nil, err
	}
	return capture.sound, nil
}

func writeSound(opts Options, name string, sound audio.Sound) error {
	w, err := opts.Open(name)
	if err != nil {
		return err
	}
	defer w.Close()
	return pipeline.WAVExporter{W: w}.Export(sound)
}

func peakOf(s audio.Sound) float64 {
	loudest := 0.0
	for _, v := range s.Samples() {
		loudest = math.Max(loudest, math.Abs(v))
	}
	return loudest
}

func (r Recipe) sourceFor(out Output, opts Options) (pipeline.Source, error) {
	if out.Loop != nil {
		sound, err := out.Loop.build()
		if err != nil {
			return nil, err
		}
		return pipeline.StaticSource{Sound: sound}, nil
	}
	if out.Sequence != nil {
		sound, err := out.Sequence.build()
		if err != nil {
			return nil, err
		}
		return pipeline.StaticSource{Sound: sound}, nil
	}
	if len(out.Layers) > 0 {
		sound, err := mixLayers(out.Layers)
		if err != nil {
			return nil, err
		}
		return pipeline.StaticSource{Sound: sound}, nil
	}
	if out.Preset != "" {
		preset, ok := presets.Resolve(out.Preset)
		if !ok {
			return nil, fmt.Errorf("%w: %q", ErrUnknownPreset, out.Preset)
		}
		return pipeline.StaticSource{Sound: preset.Sound}, nil
	}
	return r.speak(out.Say, opts), nil
}

func (r Recipe) speak(text string, opts Options) voice.Piper {
	return voice.Piper{
		Binary:   r.Voice.Binary,
		Model:    r.Voice.Model,
		Args:     r.Voice.Args,
		Text:     text,
		Run:      opts.Run,
		LookPath: opts.LookPath,
	}
}

// effectsFor resolves the chain for one output. A nil Effects field inherits
// the recipe chain; an explicit empty list means none.
func (r Recipe) effectsFor(out Output) []EffectSpec {
	if out.Effects == nil {
		return r.Effects
	}
	return out.Effects
}

// labelText resolves the spoken label for an output, or "" for no label. The
// default reads the output name aloud with separators as word breaks, so
// "turn-ready_soft" is spoken "turn ready soft".
func (r Recipe) labelText(out Output) string {
	if out.Label != nil {
		return *out.Label
	}
	if !r.Labels {
		return ""
	}
	return strings.Join(strings.FieldsFunc(out.Name, func(c rune) bool {
		return c == '-' || c == '_' || c == '.'
	}), " ")
}

func buildChain(specs []EffectSpec) ([]pipeline.Effect, error) {
	if len(specs) == 0 {
		return nil, nil
	}
	chain := make([]pipeline.Effect, 0, len(specs))
	for _, spec := range specs {
		effect, err := spec.build()
		if err != nil {
			return nil, err
		}
		chain = append(chain, effect)
	}
	return chain, nil
}

func (s EffectSpec) build() (pipeline.Effect, error) {
	switch strings.ToLower(s.Type) {
	case "gain":
		return effects.Gain{Factor: s.Factor}, nil
	case "hardclip":
		return effects.HardClip{Threshold: s.Threshold}, nil
	case "softclip":
		return effects.SoftClip{Drive: s.Drive}, nil
	case "fuzz":
		return effects.Fuzz{Drive: s.Drive, Bias: s.Bias}, nil
	case "normalize":
		return effects.Normalize{Peak: s.Peak}, nil
	case "highpass":
		return effects.HighPass{Cutoff: s.Cutoff, Poles: s.Poles}, nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownEffect, s.Type)
	}
}
