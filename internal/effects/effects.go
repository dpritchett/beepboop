// Package effects provides the core sample transforms that fill a pipeline's
// Effects slice: level, saturation, and normalization.
//
// Every effect here satisfies pipeline.Effect. They are pure and deterministic,
// they never mutate their input, and they preserve sample rate and length so
// they compose in any order. Invalid configuration (a zero threshold, a
// non-positive drive) degrades to pass-through rather than silence, because a
// misconfigured stage that eats the whole signal is harder to debug than one
// that quietly does nothing.
package effects

import (
	"math"

	"beepboop/internal/audio"
)

// Gain multiplies every sample by Factor. It deliberately does not clamp:
// pushing past 1.0 is how you feed a clip or fuzz stage, and Normalize or the
// WAV writer will deal with any remaining overshoot.
type Gain struct {
	Factor float64
}

func (e Gain) Apply(s audio.Sound) audio.Sound {
	return mapSamples(s, func(v float64) float64 { return v * e.Factor })
}

// HardClip clamps samples to +/- Threshold, the buzzy transistor-style
// distortion. A non-positive threshold is pass-through.
type HardClip struct {
	Threshold float64
}

func (e HardClip) Apply(s audio.Sound) audio.Sound {
	if e.Threshold <= 0 {
		return passThrough(s)
	}
	return mapSamples(s, func(v float64) float64 {
		return math.Max(-e.Threshold, math.Min(e.Threshold, v))
	})
}

// SoftClip saturates through tanh, rounding off the corners that HardClip
// leaves behind. Higher Drive means earlier, heavier compression.
//
// The curve is normalized by tanh(Drive) so a full-scale input stays full
// scale: the effect changes the shape of the waveform, not its peak level.
// That normalization means an input already past full scale (say, after a
// Gain stage) would land just above 1.0, so the result is clamped. A
// non-positive drive is pass-through.
type SoftClip struct {
	Drive float64
}

func (e SoftClip) Apply(s audio.Sound) audio.Sound {
	if e.Drive <= 0 {
		return passThrough(s)
	}
	ceiling := math.Tanh(e.Drive)
	return mapSamples(s, func(v float64) float64 {
		return clampUnit(math.Tanh(v*e.Drive) / ceiling)
	})
}

// Fuzz is aggressive asymmetric saturation: the Bias shifts the waveform
// before clipping so the positive and negative halves distort differently.
// That asymmetry is what makes it buzz rather than merely compress.
//
// The bias is removed after saturation so the result stays centered and does
// not introduce a DC offset into anything downstream. A non-positive drive is
// pass-through.
type Fuzz struct {
	Drive float64
	Bias  float64
}

func (e Fuzz) Apply(s audio.Sound) audio.Sound {
	if e.Drive <= 0 {
		return passThrough(s)
	}
	offset := math.Tanh(e.Bias)
	return mapSamples(s, func(v float64) float64 {
		return clampUnit(math.Tanh(v*e.Drive+e.Bias) - offset)
	})
}

// Normalize scales the whole buffer so its loudest sample sits exactly at
// Peak, bringing quiet material up and loud material down by one constant
// factor. Silence is returned unchanged rather than divided by zero, and a
// non-positive peak is pass-through.
type Normalize struct {
	Peak float64
}

func (e Normalize) Apply(s audio.Sound) audio.Sound {
	if e.Peak <= 0 {
		return passThrough(s)
	}
	in := s.Samples()
	loudest := 0.0
	for _, v := range in {
		loudest = math.Max(loudest, math.Abs(v))
	}
	if loudest == 0 {
		return audio.NewSound(s.SampleRate(), in)
	}
	scale := e.Peak / loudest
	return mapSamples(s, func(v float64) float64 { return v * scale })
}

// mapSamples builds a new Sound by applying fn to each sample. audio.NewSound
// copies, so the caller's buffer is never touched.
func mapSamples(s audio.Sound, fn func(float64) float64) audio.Sound {
	in := s.Samples()
	out := make([]float64, len(in))
	for i, v := range in {
		out[i] = fn(v)
	}
	return audio.NewSound(s.SampleRate(), out)
}

func passThrough(s audio.Sound) audio.Sound {
	return audio.NewSound(s.SampleRate(), s.Samples())
}

func clampUnit(v float64) float64 {
	return math.Max(-1, math.Min(1, v))
}
