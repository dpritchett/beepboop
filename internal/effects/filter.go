package effects

import (
	"math"

	"beepboop/internal/audio"
)

// HighPass removes energy below Cutoff, in Hz.
//
// This exists because stacked harmonic voices pile up in the low end: several
// pad voices, a noise bed, and a bass line all put energy under 200Hz, and the
// sum reads as one muddy drone rather than as a chord. Cutting that build-up is
// usually a better fix than turning voices down, because it keeps their
// character while removing only the part that was colliding.
//
// It is a cascade of one-pole high-passes, not a low-pass subtracted from the
// input. Subtraction works for a single pole but falls apart past that: each
// pole adds phase shift, so the low-passed copy is no longer aligned with the
// original and subtracting it stops cancelling the bass instead of removing
// more of it.
//
// Each pole wraps its state across the buffer boundary, so applying this to a
// looping bed leaves the loop seamless. Filtering from a cold zero state would
// leave a step at the wrap.
type HighPass struct {
	Cutoff float64
	// Poles steepens the slope. One pole is a gentle 6dB per octave, which
	// often is not enough to clear space; three or four cuts decisively.
	// Defaults to 1.
	Poles int
}

func (e HighPass) Apply(s audio.Sound) audio.Sound {
	in := s.Samples()
	if e.Cutoff <= 0 || len(in) == 0 {
		return audio.NewSound(s.SampleRate(), in)
	}

	// Standard one-pole RC high-pass coefficient. Always in (0, 1), so the
	// filter stays stable for any cutoff the caller asks for.
	alpha := 1 / (1 + 2*math.Pi*e.Cutoff/float64(s.SampleRate()))

	poles := e.Poles
	if poles < 1 {
		poles = 1
	}

	out := make([]float64, len(in))
	copy(out, in)
	for i := 0; i < poles; i++ {
		circularHighPass(out, alpha)
	}
	return audio.NewSound(s.SampleRate(), out)
}

// circularHighPass applies one pole in place, wrapping so a looping buffer
// stays seamless. The first pass only settles the filter state into its
// periodic value; the second pass is the one that is kept.
func circularHighPass(samples []float64, alpha float64) {
	var previousIn, previousOut float64
	for pass := 0; pass < 2; pass++ {
		for i := range samples {
			// Capture before writing: pass two overwrites as it goes, and the
			// recurrence needs the original previous input.
			in := samples[i]
			out := alpha * (previousOut + in - previousIn)
			previousIn, previousOut = in, out
			if pass == 1 {
				samples[i] = out
			}
		}
	}
}
