package audio

import (
	"math"
	"testing"
)

func blip(start, duration, frequency, gain float64) Note {
	return Note{
		Start:    start,
		Duration: duration,
		Partials: []Partial{{Frequency: frequency, Gain: 1}},
		Gain:     gain,
	}
}

func TestSequenceLength(t *testing.T) {
	got := Sequence(SequenceSpec{
		SampleRate: 8000,
		Duration:   2.0,
		Notes:      []Note{blip(0, 0.5, 440, 1)},
	})
	if want := 16000; len(got.Samples()) != want {
		t.Errorf("samples = %d, want %d", len(got.Samples()), want)
	}
}

func TestSequencePlacesNotesInTime(t *testing.T) {
	// A note at 1s must leave the first second silent and the second not.
	samples := Sequence(SequenceSpec{
		SampleRate: 8000,
		Duration:   2.0,
		Notes:      []Note{blip(1.0, 0.5, 440, 1)},
	}).Samples()

	if got := peakOf(samples[:8000]); got != 0 {
		t.Errorf("peak before the note = %v, want silence", got)
	}
	if got := peakOf(samples[8000:]); got == 0 {
		t.Error("note did not sound")
	}
}

func TestSequenceMixesOverlappingNotes(t *testing.T) {
	// Chords and overlapping tails are the point of a mixer.
	single := Sequence(SequenceSpec{
		SampleRate: 8000,
		Duration:   1.0,
		Notes:      []Note{blip(0, 0.5, 440, 1)},
	}).Samples()
	double := Sequence(SequenceSpec{
		SampleRate: 8000,
		Duration:   1.0,
		Notes: []Note{
			blip(0, 0.5, 440, 1),
			blip(0, 0.5, 660, 1),
		},
	}).Samples()

	if peakOf(double) <= peakOf(single) {
		t.Errorf("two notes peak %v is not above one note %v", peakOf(double), peakOf(single))
	}
}

func TestSequenceEnvelopeAvoidsClicks(t *testing.T) {
	// A note that jumps straight to full amplitude clicks on every hit. The
	// attack ramp is what stops that, so the first sample must be near zero.
	samples := Sequence(SequenceSpec{
		SampleRate: 22050,
		Duration:   1.0,
		Notes:      []Note{blip(0, 0.5, 440, 1)},
	}).Samples()

	if math.Abs(samples[0]) > 0.01 {
		t.Errorf("first sample = %v, want a ramp from near zero", samples[0])
	}
	// And it must decay away rather than being cut off mid-cycle.
	tail := peakOf(samples[10584:11466])  // 0.48s to 0.52s
	body := peakOf(samples[1102:3307])    // 0.05s to 0.15s
	if tail > body*0.5 {
		t.Errorf("note tail %v is not decaying against body %v", tail, body)
	}
}

func TestSequenceWrapsNotesAroundTheLoop(t *testing.T) {
	// A note near the end has a tail that belongs at the start of the next
	// repeat. Truncating it puts a hole and a click at the wrap; wrapping it
	// around is what makes a sequenced loop seamless.
	samples := Sequence(SequenceSpec{
		SampleRate: 8000,
		Duration:   2.0,
		Notes:      []Note{blip(1.9, 0.5, 440, 1)},
	}).Samples()

	if got := peakOf(samples[:800]); got == 0 {
		t.Error("note tail did not wrap to the start of the loop")
	}
}

func TestSequenceIsSeamless(t *testing.T) {
	notes := []Note{
		blip(0.0, 0.4, 440, 0.8),
		blip(0.5, 0.4, 660, 0.8),
		blip(1.0, 0.4, 880, 0.8),
		blip(1.85, 0.4, 550, 0.8), // deliberately overhangs the wrap
	}
	samples := Sequence(SequenceSpec{
		SampleRate: 22050,
		Duration:   2.0,
		Notes:      notes,
	}).Samples()

	if seam, step := seamStep(samples), maxStep(samples); seam > step {
		t.Errorf("seam jump %v exceeds largest in-buffer step %v", seam, step)
	}
}

func TestSequenceIsDeterministic(t *testing.T) {
	spec := SequenceSpec{
		SampleRate: 22050,
		Duration:   1.0,
		Notes:      []Note{blip(0, 0.5, 440, 1), blip(0.5, 0.4, 660, 0.7)},
	}
	a, b := Sequence(spec).Samples(), Sequence(spec).Samples()
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("nondeterministic at %d: %v vs %v", i, a[i], b[i])
		}
	}
}

func TestSequenceDecayControlsPluckiness(t *testing.T) {
	// High decay is a pluck, low decay sustains. Compare energy late in the
	// note.
	late := func(decay float64) float64 {
		note := blip(0, 1.0, 440, 1)
		note.Decay = decay
		samples := Sequence(SequenceSpec{
			SampleRate: 22050, Duration: 1.0, Notes: []Note{note},
		}).Samples()
		return peakOf(samples[int(0.7*22050):])
	}
	if late(12) >= late(2) {
		t.Errorf("plucky decay tail %v is not below sustained %v", late(12), late(2))
	}
}

func TestSequenceEmptyIsSilence(t *testing.T) {
	samples := Sequence(SequenceSpec{SampleRate: 8000, Duration: 0.5}).Samples()
	if len(samples) != 4000 {
		t.Fatalf("samples = %d, want 4000", len(samples))
	}
	if peakOf(samples) != 0 {
		t.Errorf("peak = %v, want silence", peakOf(samples))
	}
}

func TestSequenceIgnoresNonsenseNotes(t *testing.T) {
	samples := Sequence(SequenceSpec{
		SampleRate: 8000,
		Duration:   1.0,
		Notes: []Note{
			{Start: 0, Duration: 0, Partials: []Partial{{Frequency: 440, Gain: 1}}, Gain: 1},
			{Start: 0, Duration: 0.5, Gain: 1}, // no partials
		},
	}).Samples()
	if peakOf(samples) != 0 {
		t.Errorf("peak = %v, want silence from unusable notes", peakOf(samples))
	}
}
