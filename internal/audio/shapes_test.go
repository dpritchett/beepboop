package audio

import (
	"math"
	"testing"
)

func TestSawPartialsFollowOneOverN(t *testing.T) {
	got := SawPartials(100, 550)

	// 100 through 500 Hz: every harmonic, amplitude 1/n.
	if len(got) != 5 {
		t.Fatalf("partials = %d, want 5: %v", len(got), got)
	}
	for i, p := range got {
		n := float64(i + 1)
		if p.Frequency != 100*n {
			t.Errorf("partial %d frequency = %v, want %v", i, p.Frequency, 100*n)
		}
		if math.Abs(p.Gain-1/n) > 1e-9 {
			t.Errorf("partial %d gain = %v, want %v", i, p.Gain, 1/n)
		}
	}
}

func TestSquarePartialsAreOddHarmonicsOnly(t *testing.T) {
	got := SquarePartials(100, 1050)

	// 100, 300, 500, 700, 900 Hz at 1, 1/3, 1/5, 1/7, 1/9.
	want := []float64{100, 300, 500, 700, 900}
	if len(got) != len(want) {
		t.Fatalf("partials = %d, want %d: %v", len(got), len(want), got)
	}
	for i, p := range got {
		if p.Frequency != want[i] {
			t.Errorf("partial %d frequency = %v, want %v", i, p.Frequency, want[i])
		}
		n := float64(2*i + 1)
		if math.Abs(p.Gain-1/n) > 1e-9 {
			t.Errorf("partial %d gain = %v, want %v", i, p.Gain, 1/n)
		}
	}
}

func TestTrianglePartialsFallOffFaster(t *testing.T) {
	// Odd harmonics at 1/n squared, which is why a triangle is so much
	// mellower than a square built on the same fundamental.
	got := TrianglePartials(100, 1050)

	if len(got) != 5 {
		t.Fatalf("partials = %d, want 5: %v", len(got), got)
	}
	for i, p := range got {
		n := float64(2*i + 1)
		if math.Abs(math.Abs(p.Gain)-1/(n*n)) > 1e-9 {
			t.Errorf("partial %d gain = %v, want +/-%v", i, p.Gain, 1/(n*n))
		}
	}
}

func TestShapesRespectTheLimit(t *testing.T) {
	// The limit is what keeps a stack band-limited. Every partial above it
	// would fold back down as an unrelated whistle.
	for name, got := range map[string][]Partial{
		"saw":      SawPartials(1000, 3500),
		"square":   SquarePartials(1000, 3500),
		"triangle": TrianglePartials(1000, 3500),
	} {
		if len(got) == 0 {
			t.Errorf("%s produced no partials", name)
		}
		for _, p := range got {
			if p.Frequency > 3500 {
				t.Errorf("%s partial at %v exceeds the 3500 limit", name, p.Frequency)
			}
		}
	}
}

func TestShapesRejectNonsense(t *testing.T) {
	for name, got := range map[string][]Partial{
		"zero fundamental": SawPartials(0, 1000),
		"limit below f0":   SawPartials(1000, 500),
		"negative limit":   SawPartials(100, -1),
	} {
		if len(got) != 0 {
			t.Errorf("%s produced %d partials, want none", name, len(got))
		}
	}
}

func TestSawIsBrighterThanTriangle(t *testing.T) {
	// The whole point of offering shapes: they have to actually differ in
	// harmonic content, not just in name.
	energy := func(partials []Partial) float64 {
		loop := Loop(LoopSpec{SampleRate: 22050, Duration: 0.5, Partials: partials})
		samples := loop.Samples()
		total := 0.0
		for i := 1; i < len(samples); i++ {
			total += math.Abs(samples[i] - samples[i-1])
		}
		return total / float64(len(samples))
	}

	saw := energy(SawPartials(110, 11000))
	triangle := energy(TrianglePartials(110, 11000))
	if saw <= triangle {
		t.Errorf("saw step energy %v is not above triangle %v", saw, triangle)
	}
}

func TestShapesStaySeamlessThroughLoop(t *testing.T) {
	// Awkward fundamental on purpose: the loop snapping has to hold for a
	// whole harmonic stack, not just one partial.
	samples := Loop(LoopSpec{
		SampleRate: 22050,
		Duration:   2.0,
		Partials:   SawPartials(97.3, 8000),
	}).Samples()

	if seam, step := seamStep(samples), maxStep(samples); seam > step {
		t.Errorf("seam jump %v exceeds largest in-buffer step %v", seam, step)
	}
}
