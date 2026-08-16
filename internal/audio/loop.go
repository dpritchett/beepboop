package audio

import (
	"math"
	"math/rand"
)

// Partial is one sine component of a looping timbre.
type Partial struct {
	Frequency float64
	Gain      float64
	// Phase offsets the component in turns (0 to 1). Detuning two partials
	// by a fraction of a turn is what gives a drone its slow beating.
	Phase float64
}

// LoopSpec describes a continuous sound built to repeat forever: engine
// rumble, turbine whine, a drone under slow movement.
//
// Loops are a different problem from one-shot alerts. A preset can fade in
// and out, but anything played on repeat while a camera moves has to wrap
// without a click, which constrains both halves of the recipe. Partials are
// snapped to frequencies that complete a whole number of cycles per loop, and
// the noise bed is crossfaded across the seam.
type LoopSpec struct {
	SampleRate int
	Duration   float64
	Partials   []Partial
	// Noise is the amount of broadband noise mixed under the partials, 0 to 1.
	Noise float64
	// NoiseTone darkens that noise, 0 to 1. Near 1 is open hiss; low values
	// roll off the top for the thicker roar of moving air.
	NoiseTone float64
	// NoisePoles cascades the tone filter. One pole is a gentle 6dB per
	// octave that still leaves audible hiss; three or four is what makes a
	// noise bed read as a low roar rather than static. Defaults to 1.
	NoisePoles int
	// Seed fixes the noise so a given spec always renders the same bytes.
	Seed int64
}

// Loop renders a seamlessly repeating sound.
//
// The result is scaled so the loudest sample sits at full scale rather than
// clipping, since a stack of partials plus noise routinely sums past 1.0.
// Callers set final level with an effects chain.
func Loop(spec LoopSpec) Sound {
	if spec.SampleRate <= 0 {
		spec.SampleRate = 44100
	}
	total := int(math.Round(spec.Duration * float64(spec.SampleRate)))
	if total <= 0 {
		return generatedSound{sampleRate: spec.SampleRate}
	}
	samples := make([]float64, total)

	addPartials(samples, spec, total)
	if spec.Noise > 0 {
		addNoise(samples, spec, total)
	}
	normalize(samples)

	return generatedSound{sampleRate: spec.SampleRate, samples: samples}
}

func addPartials(samples []float64, spec LoopSpec, total int) {
	nyquist := float64(spec.SampleRate) / 2
	loops := float64(total) / float64(spec.SampleRate)

	for _, partial := range spec.Partials {
		// Snap to the nearest frequency completing whole cycles in the loop.
		// Anything else leaves a phase discontinuity at the wrap point, which
		// is the click these sounds must not have.
		cycles := math.Round(partial.Frequency * loops)
		if cycles < 1 {
			continue
		}
		frequency := cycles / loops
		// A partial above Nyquist folds back down as an unrelated whistle.
		if frequency >= nyquist {
			continue
		}
		phase := partial.Phase * 2 * math.Pi
		step := 2 * math.Pi * frequency / float64(spec.SampleRate)
		for i := range samples {
			samples[i] += math.Sin(step*float64(i)+phase) * partial.Gain
		}
	}
}

// addNoise mixes in a broadband bed that also wraps cleanly.
//
// No crossfade is involved, deliberately. White noise has no continuity to
// preserve: neighbouring samples are uncorrelated everywhere, so the wrap
// point is statistically identical to every other sample boundary. Once the
// tone filter runs circularly, carrying its state across the boundary, the
// filtered bed is genuinely periodic.
//
// An earlier version crossfaded an overhang back over the start, which put a
// stretch of blended, more-correlated noise at the head of every loop. On a
// heavily filtered bed that stretch has a different character from the rest
// and lurches once per repeat. Generating exactly one loop of noise and
// filtering it circularly has no such region.
func addNoise(samples []float64, spec LoopSpec, total int) {
	source := rand.New(rand.NewSource(spec.Seed))
	noise := make([]float64, total)
	for i := range noise {
		noise[i] = source.Float64()*2 - 1
	}

	if tone := spec.NoiseTone; tone > 0 && tone < 1 {
		poles := spec.NoisePoles
		if poles < 1 {
			poles = 1
		}
		for i := 0; i < poles; i++ {
			lowpass(noise, tone)
		}
	}
	for i := range samples {
		samples[i] += noise[i] * spec.Noise
	}
}

// lowpass applies a one-pole filter in place, wrapping so the loop stays
// seamless. The first pass only settles the filter state into its periodic
// value; the second pass is the one that is kept. Filtering from a cold zero
// state would leave the start of the buffer quieter than the end.
func lowpass(samples []float64, coefficient float64) {
	state := 0.0
	for pass := 0; pass < 2; pass++ {
		for i, v := range samples {
			state += coefficient * (v - state)
			if pass == 1 {
				samples[i] = state
			}
		}
	}
	// A one-pole filter costs a lot of level; restore it so NoiseTone reads
	// as a tone control rather than a volume control.
	normalize(samples)
}

func normalize(samples []float64) {
	loudest := 0.0
	for _, v := range samples {
		loudest = math.Max(loudest, math.Abs(v))
	}
	if loudest == 0 {
		return
	}
	for i := range samples {
		samples[i] /= loudest
	}
}
