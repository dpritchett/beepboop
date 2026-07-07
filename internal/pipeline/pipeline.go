// Package pipeline composes audio sources, effects, and exporters into a
// single deterministic render path: source -> effects -> export.
//
// The interfaces are the durable product. Synthetic presets, WAV readback,
// and Piper all become Sources; gain/clip/fuzz become Effects; WAV and MP3
// become Exporters. Ambient state (files, process spawning) is injected into
// the concrete adapters, never reached for here, so pipelines stay testable
// without the filesystem or a real player.
package pipeline

import (
	"errors"
	"io"

	"beepboop/internal/audio"
	"beepboop/internal/wav"
)

var (
	ErrNoSource   = errors.New("pipeline: no source")
	ErrNoExporter = errors.New("pipeline: no exporter")
)

// Source produces an audio buffer. Rendering may fail for adapters that shell
// out or read files (Piper, WAV readback), so it returns an error.
type Source interface {
	Render() (audio.Sound, error)
}

// Effect is a pure, deterministic transform over a sound. Effects do not fail;
// a transform that needs configuration validates it at construction time.
type Effect interface {
	Apply(audio.Sound) audio.Sound
}

// Exporter writes a finished sound somewhere. The destination is injected into
// the concrete exporter, keeping the pipeline free of ambient state.
type Exporter interface {
	Export(audio.Sound) error
}

// Pipeline renders a Source, applies Effects in order, and hands the result to
// an Exporter. The zero value is unusable; Source and Exporter are required.
type Pipeline struct {
	Source   Source
	Effects  []Effect
	Exporter Exporter
}

// Run executes the pipeline. Effects are applied left to right; a nil effect
// is skipped so callers can build effect slices conditionally.
func (p Pipeline) Run() error {
	if p.Source == nil {
		return ErrNoSource
	}
	if p.Exporter == nil {
		return ErrNoExporter
	}
	sound, err := p.Source.Render()
	if err != nil {
		return err
	}
	for _, effect := range p.Effects {
		if effect == nil {
			continue
		}
		sound = effect.Apply(sound)
	}
	return p.Exporter.Export(sound)
}

// StaticSource adapts an already-rendered Sound (such as a preset) into a
// Source, letting existing synthesis flow through the pipeline unchanged.
type StaticSource struct {
	Sound audio.Sound
}

func (s StaticSource) Render() (audio.Sound, error) {
	return s.Sound, nil
}

// WAVExporter writes PCM16 mono WAV to an injected writer.
type WAVExporter struct {
	W io.Writer
}

func (e WAVExporter) Export(s audio.Sound) error {
	return wav.WritePCM16Mono(e.W, s.SampleRate(), s.Samples())
}
