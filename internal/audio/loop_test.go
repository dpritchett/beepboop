package audio

import (
	"math"
	"sort"
	"testing"
)

// maxStep is the largest jump between neighbouring samples inside the buffer.
func maxStep(samples []float64) float64 {
	worst := 0.0
	for i := 1; i < len(samples); i++ {
		worst = math.Max(worst, math.Abs(samples[i]-samples[i-1]))
	}
	return worst
}

// seamStep is the jump a looping player hears when the buffer wraps.
func seamStep(samples []float64) float64 {
	return math.Abs(samples[0] - samples[len(samples)-1])
}

func peakOf(samples []float64) float64 {
	loudest := 0.0
	for _, v := range samples {
		loudest = math.Max(loudest, math.Abs(v))
	}
	return loudest
}

func TestLoopLength(t *testing.T) {
	got := Loop(LoopSpec{
		SampleRate: 8000,
		Duration:   0.5,
		Partials:   []Partial{{Frequency: 100, Gain: 1}},
	})
	if want := 4000; len(got.Samples()) != want {
		t.Errorf("samples = %d, want %d", len(got.Samples()), want)
	}
	if got.SampleRate() != 8000 {
		t.Errorf("SampleRate() = %d, want 8000", got.SampleRate())
	}
}

func TestLoopIsSeamlessForPartials(t *testing.T) {
	// A flight sound plays on repeat for as long as the camera moves. If the
	// buffer does not wrap cleanly, every loop boundary is an audible click.
	// Frequencies that do not divide the loop evenly are the usual cause, so
	// this uses deliberately awkward ones.
	got := Loop(LoopSpec{
		SampleRate: 44100,
		Duration:   0.5,
		Partials: []Partial{
			{Frequency: 97.3, Gain: 0.5},
			{Frequency: 213.7, Gain: 0.3},
			{Frequency: 1471.9, Gain: 0.2},
		},
	}).Samples()

	if seam, step := seamStep(got), maxStep(got); seam > step {
		t.Errorf("seam jump %v exceeds largest in-buffer step %v", seam, step)
	}
}

func TestLoopIsSeamlessWithNoise(t *testing.T) {
	// Raw noise cannot wrap on its own; the generator has to crossfade it.
	got := Loop(LoopSpec{
		SampleRate: 44100,
		Duration:   0.5,
		Noise:      1.0,
		Seed:       7,
	}).Samples()

	if seam, step := seamStep(got), maxStep(got); seam > step {
		t.Errorf("seam jump %v exceeds largest in-buffer step %v", seam, step)
	}
}

// windowRMS returns the RMS level of each fixed-size window in order.
func windowRMS(samples []float64, window int) []float64 {
	var out []float64
	for start := 0; start+window <= len(samples); start += window {
		sum := 0.0
		for _, v := range samples[start : start+window] {
			sum += v * v
		}
		out = append(out, math.Sqrt(sum/float64(window)))
	}
	return out
}

func TestLoopNoiseHasNoLevelDip(t *testing.T) {
	// Crossfading two uncorrelated noise streams with linear gains loses 3dB
	// of power at the midpoint, which is audible as the level sagging and
	// recovering once per loop. Equal-power gains are required here, and
	// nothing about the seam-continuity test catches this.
	samples := Loop(LoopSpec{
		SampleRate: 22050,
		Duration:   2.0,
		Noise:      1,
		NoiseTone:  1,
		Seed:       11,
	}).Samples()

	levels := windowRMS(samples, 551) // 25ms
	sorted := append([]float64(nil), levels...)
	sort.Float64s(sorted)
	median := sorted[len(sorted)/2]
	quietest := sorted[0]

	if quietest/median < 0.85 {
		t.Errorf("quietest 25ms window is %.3f of median (%.4f vs %.4f); the loop dips",
			quietest/median, quietest, median)
	}
}

func TestLoopNoiseLevelIsSteadyAcrossTheSeam(t *testing.T) {
	// The dip lands wherever the crossfade sits, so compare the windows
	// bracketing the wrap against the body of the loop specifically.
	samples := Loop(LoopSpec{
		SampleRate: 22050,
		Duration:   1.0,
		Noise:      1,
		NoiseTone:  1,
		Seed:       12,
	}).Samples()

	window := 551
	levels := windowRMS(samples, window)
	body := levels[len(levels)/2]
	for _, i := range []int{0, 1, 2, len(levels) - 1} {
		if ratio := levels[i] / body; ratio < 0.85 || ratio > 1.18 {
			t.Errorf("window %d level is %.3f of mid-loop level, want near 1", i, ratio)
		}
	}
}

// windowStep returns the mean absolute sample-to-sample step per window, a
// proxy for how much high-frequency energy each stretch of the buffer holds.
func windowStep(samples []float64, window int) []float64 {
	var out []float64
	for start := 1; start+window <= len(samples); start += window {
		sum := 0.0
		for i := start; i < start+window; i++ {
			sum += math.Abs(samples[i] - samples[i-1])
		}
		out = append(out, sum/float64(window))
	}
	return out
}

func TestLoopNoiseTextureIsUniform(t *testing.T) {
	// Level being flat is not enough. Any stretch of the loop whose noise is
	// built differently from the rest, such as a blend of two streams across
	// a crossfade, has different statistics even at matched level. On a
	// heavily filtered bed that reads as a lurch once per loop.
	samples := Loop(LoopSpec{
		SampleRate: 22050,
		Duration:   2.0,
		Noise:      1,
		NoiseTone:  0.10,
		NoisePoles: 4,
		Seed:       1101,
	}).Samples()

	steps := windowStep(samples, 2205) // 100ms
	sorted := append([]float64(nil), steps...)
	sort.Float64s(sorted)
	median := sorted[len(sorted)/2]

	for i, step := range steps {
		if ratio := step / median; ratio < 0.6 || ratio > 1.6 {
			t.Errorf("window %d texture is %.2f of median; the loop is not uniform", i, ratio)
		}
	}
}

func TestLoopSnapsPartialsToTheLoopPeriod(t *testing.T) {
	// Over a 1s loop only whole-Hz partials complete an integer number of
	// cycles, so 100.4 Hz has to become 100 Hz.
	got := Loop(LoopSpec{
		SampleRate: 8000,
		Duration:   1.0,
		Partials:   []Partial{{Frequency: 100.4, Gain: 1}},
	}).Samples()
	want := Loop(LoopSpec{
		SampleRate: 8000,
		Duration:   1.0,
		Partials:   []Partial{{Frequency: 100, Gain: 1}},
	}).Samples()

	for i := range want {
		if math.Abs(got[i]-want[i]) > 1e-9 {
			t.Fatalf("sample[%d] = %v, want %v (frequency was not snapped)", i, got[i], want[i])
		}
	}
}

func TestLoopKeepsPartialsBelowNyquist(t *testing.T) {
	// A partial above half the sample rate aliases down into an audible
	// whistle that has nothing to do with the intended timbre.
	got := Loop(LoopSpec{
		SampleRate: 8000,
		Duration:   0.25,
		Partials: []Partial{
			{Frequency: 200, Gain: 0.5},
			{Frequency: 6000, Gain: 0.5},
		},
	}).Samples()
	want := Loop(LoopSpec{
		SampleRate: 8000,
		Duration:   0.25,
		Partials:   []Partial{{Frequency: 200, Gain: 0.5}},
	}).Samples()

	for i := range want {
		if math.Abs(got[i]-want[i]) > 1e-9 {
			t.Fatalf("sample[%d] = %v, want %v (aliasing partial was not dropped)", i, got[i], want[i])
		}
	}
}

func TestLoopIsDeterministic(t *testing.T) {
	spec := LoopSpec{
		SampleRate: 22050,
		Duration:   0.3,
		Partials:   []Partial{{Frequency: 120, Gain: 0.4}},
		Noise:      0.5,
		Seed:       42,
	}
	a, b := Loop(spec).Samples(), Loop(spec).Samples()
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("nondeterministic at %d: %v vs %v", i, a[i], b[i])
		}
	}
}

func TestLoopSeedChangesNoise(t *testing.T) {
	base := LoopSpec{SampleRate: 22050, Duration: 0.3, Noise: 1, Seed: 1}
	other := base
	other.Seed = 2

	a, b := Loop(base).Samples(), Loop(other).Samples()
	same := true
	for i := range a {
		if a[i] != b[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("different seeds produced identical noise")
	}
}

func TestLoopStaysInRange(t *testing.T) {
	got := Loop(LoopSpec{
		SampleRate: 22050,
		Duration:   0.4,
		Partials: []Partial{
			{Frequency: 90, Gain: 1},
			{Frequency: 180, Gain: 1},
			{Frequency: 360, Gain: 1},
		},
		Noise: 1,
		Seed:  3,
	}).Samples()

	if got := peakOf(got); got > 1 {
		t.Errorf("peak = %v, want within [-1, 1]", got)
	}
}

func TestLoopNoiseToneDarkensTheNoise(t *testing.T) {
	// A turbine is not white hiss. Lower NoiseTone has to actually remove
	// high frequencies, which shows up as smaller sample-to-sample steps.
	bright := Loop(LoopSpec{
		SampleRate: 22050, Duration: 0.3, Noise: 1, NoiseTone: 1, Seed: 5,
	}).Samples()
	dark := Loop(LoopSpec{
		SampleRate: 22050, Duration: 0.3, Noise: 1, NoiseTone: 0.05, Seed: 5,
	}).Samples()

	if maxStep(dark) >= maxStep(bright) {
		t.Errorf("dark step %v not below bright step %v", maxStep(dark), maxStep(bright))
	}
}

func TestLoopNoisePolesSteepenTheRolloff(t *testing.T) {
	// One pole is only 6dB per octave, which leaves plenty of hiss on top of
	// what should read as a low roar. Cascading poles is what turns the bed
	// from "white noise" into "moving air".
	spec := LoopSpec{SampleRate: 22050, Duration: 0.3, Noise: 1, NoiseTone: 0.25, Seed: 9}
	one, four := spec, spec
	one.NoisePoles, four.NoisePoles = 1, 4

	gentle := maxStep(Loop(one).Samples())
	steep := maxStep(Loop(four).Samples())
	if steep >= gentle {
		t.Errorf("four-pole step %v is not below one-pole %v", steep, gentle)
	}
}

func TestLoopNoisePolesStaySeamless(t *testing.T) {
	got := Loop(LoopSpec{
		SampleRate: 22050, Duration: 0.5, Noise: 1, NoiseTone: 0.2,
		NoisePoles: 4, Seed: 9,
	}).Samples()

	if seam, step := seamStep(got), maxStep(got); seam > step {
		t.Errorf("seam jump %v exceeds largest in-buffer step %v", seam, step)
	}
}

func TestLoopEmptySpecIsSilence(t *testing.T) {
	got := Loop(LoopSpec{SampleRate: 8000, Duration: 0.1}).Samples()
	if len(got) != 800 {
		t.Fatalf("samples = %d, want 800", len(got))
	}
	if peakOf(got) != 0 {
		t.Errorf("peak = %v, want silence", peakOf(got))
	}
}

func TestLoopZeroDurationIsEmpty(t *testing.T) {
	if got := Loop(LoopSpec{SampleRate: 8000}).Samples(); len(got) != 0 {
		t.Errorf("samples = %d, want 0", len(got))
	}
}
