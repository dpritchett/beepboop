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
	"strings"

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
	// Preset names a synthetic sound; Say is a line for the voice. Exactly
	// one of the two is required.
	Preset string `json:"preset"`
	Say    string `json:"say"`
	// Effects overrides the recipe chain when present. An explicit empty
	// list means no effects, which is how a single output opts out.
	Effects []EffectSpec `json:"effects"`
	// Label overrides the spoken label text. An explicit empty string opts
	// this output out of labeling.
	Label *string `json:"label"`
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

		if (out.Preset == "") == (out.Say == "") {
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
	if err := writeSound(opts, out.Name, source, chain); err != nil {
		return Result{}, err
	}

	result := Result{Name: out.Name}
	if text := r.labelText(out); text != "" {
		// Labels are deliberately dry: no effects chain, so the spoken name
		// stays intelligible even when the primary sound is heavily fuzzed.
		label := out.Name + LabelSuffix
		if err := writeSound(opts, label, r.speak(text, opts), nil); err != nil {
			return Result{}, fmt.Errorf("label: %w", err)
		}
		result.Label = label
	}
	return result, nil
}

func writeSound(opts Options, name string, source pipeline.Source, chain []pipeline.Effect) error {
	w, err := opts.Open(name)
	if err != nil {
		return err
	}
	defer w.Close()
	return pipeline.Pipeline{
		Source:   source,
		Effects:  chain,
		Exporter: pipeline.WAVExporter{W: w},
	}.Run()
}

func (r Recipe) sourceFor(out Output, opts Options) (pipeline.Source, error) {
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
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownEffect, s.Type)
	}
}
