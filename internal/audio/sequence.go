package audio

import "math"

// Note is one sounding event: a timbre, a place in time, and an envelope.
type Note struct {
	// Start is seconds from the loop start. Duration is how long the note
	// sounds, envelope included.
	Start    float64
	Duration float64
	// Partials is the timbre. Build it by hand or with SawPartials and
	// friends. Frequencies here are absolute, not relative to a fundamental.
	Partials []Partial
	Gain     float64
	// Attack is the ramp up from silence, defaulting to 5ms. Without one a
	// note starts mid-jump and clicks.
	Attack float64
	// Decay is the exponential falloff rate across the note. Low values
	// sustain, high values pluck. Defaults to 5, which lands the tail near
	// silence by the end of the note.
	Decay float64
}

// SequenceSpec is a loop's worth of notes.
//
// Duration is the loop length, not the length of the material: notes are
// placed on that grid and anything overhanging the end wraps back to the
// beginning, so the pattern repeats without a hole or a click at the seam.
type SequenceSpec struct {
	SampleRate int
	Duration   float64
	Notes      []Note
}

// Sequence renders notes into a single seamlessly looping buffer.
//
// Every note is summed rather than replacing what is already there, so
// overlapping notes make chords and decaying tails run under later hits. The
// result is not normalized; a dense pattern can sum past full scale, and
// setting final level is the caller's job through an effects chain.
func Sequence(spec SequenceSpec) Sound {
	if spec.SampleRate <= 0 {
		spec.SampleRate = 44100
	}
	total := int(math.Round(spec.Duration * float64(spec.SampleRate)))
	if total <= 0 {
		return generatedSound{sampleRate: spec.SampleRate}
	}
	samples := make([]float64, total)
	for _, note := range spec.Notes {
		addNote(samples, note, spec.SampleRate, total)
	}
	return generatedSound{sampleRate: spec.SampleRate, samples: samples}
}

func addNote(samples []float64, note Note, sampleRate, total int) {
	if note.Duration <= 0 || len(note.Partials) == 0 || note.Gain == 0 {
		return
	}
	attack := note.Attack
	if attack <= 0 {
		attack = 0.005
	}
	decay := note.Decay
	if decay <= 0 {
		decay = 5
	}

	length := int(math.Round(note.Duration * float64(sampleRate)))
	start := int(math.Round(note.Start * float64(sampleRate)))
	attackSamples := math.Max(1, attack*float64(sampleRate))

	for i := 0; i < length; i++ {
		t := float64(i) / float64(sampleRate)

		// Linear attack into exponential decay. The decay is measured across
		// whatever remains after the attack so a short note still resolves.
		envelope := 1.0
		if float64(i) < attackSamples {
			envelope = float64(i) / attackSamples
		} else if remaining := note.Duration - attack; remaining > 0 {
			envelope = math.Exp(-decay * (t - attack) / remaining)
		}

		value := 0.0
		for _, partial := range note.Partials {
			// Phase is relative to the note, not the buffer, so every hit of
			// the same note is identical wherever it lands.
			value += math.Sin(2*math.Pi*partial.Frequency*t) * partial.Gain
		}

		// Wrap rather than truncate: a tail hanging past the loop end belongs
		// at the start of the next repeat, which is the same place in a loop.
		samples[(start+i)%total] += value * envelope * note.Gain
	}
}
