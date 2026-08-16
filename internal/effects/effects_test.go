package effects

import (
	"math"
	"testing"

	"beepboop/internal/audio"
	"beepboop/internal/pipeline"
)

// The whole point of these types is to drop into a Pipeline's Effects slice,
// so assert the contract at compile time rather than trusting the docs.
var (
	_ pipeline.Effect = Gain{}
	_ pipeline.Effect = HardClip{}
	_ pipeline.Effect = SoftClip{}
	_ pipeline.Effect = Fuzz{}
	_ pipeline.Effect = Normalize{}
)

const tolerance = 1e-9

func sound(samples ...float64) audio.Sound {
	return audio.NewSound(8000, samples)
}

// assertSamples compares within tolerance so float math in soft-knee curves
// does not make tests brittle.
func assertSamples(t *testing.T, got audio.Sound, want []float64) {
	t.Helper()
	samples := got.Samples()
	if len(samples) != len(want) {
		t.Fatalf("sample count = %d, want %d", len(samples), len(want))
	}
	for i := range want {
		if math.Abs(samples[i]-want[i]) > tolerance {
			t.Errorf("sample[%d] = %v, want %v", i, samples[i], want[i])
		}
	}
}

func TestGainScalesSamples(t *testing.T) {
	got := Gain{Factor: 2}.Apply(sound(0.1, -0.25, 0))
	assertSamples(t, got, []float64{0.2, -0.5, 0})
}

func TestGainDoesNotClamp(t *testing.T) {
	// Gain is a pure multiply; keeping headroom past 1.0 lets a later
	// Normalize or clip stage decide what to do with the overshoot.
	got := Gain{Factor: 4}.Apply(sound(0.5, -0.5))
	assertSamples(t, got, []float64{2, -2})
}

func TestGainPreservesSampleRate(t *testing.T) {
	if got := (Gain{Factor: 2}).Apply(sound(0.1)).SampleRate(); got != 8000 {
		t.Errorf("SampleRate() = %d, want 8000", got)
	}
}

func TestHardClipClampsToThreshold(t *testing.T) {
	got := HardClip{Threshold: 0.5}.Apply(sound(0.9, -0.9, 0.25, -0.25))
	assertSamples(t, got, []float64{0.5, -0.5, 0.25, -0.25})
}

func TestHardClipNonPositiveThresholdIsPassThrough(t *testing.T) {
	// A zero threshold would silence everything, which is never what a
	// caller means; treat invalid config as a no-op instead.
	got := HardClip{Threshold: 0}.Apply(sound(0.9, -0.3))
	assertSamples(t, got, []float64{0.9, -0.3})
}

func TestSoftClipSaturatesWithoutHardCorners(t *testing.T) {
	// tanh drive: output stays inside (-1, 1) and compresses loud samples
	// more than quiet ones.
	got := SoftClip{Drive: 2}.Apply(sound(0.5, -0.5, 1.0))
	assertSamples(t, got, []float64{
		math.Tanh(1.0) / math.Tanh(2.0),
		-math.Tanh(1.0) / math.Tanh(2.0),
		1.0,
	})
}

func TestSoftClipStaysInRange(t *testing.T) {
	out := SoftClip{Drive: 8}.Apply(sound(3, -3, 0.2)).Samples()
	for i, v := range out {
		if v > 1 || v < -1 {
			t.Errorf("sample[%d] = %v, want within [-1, 1]", i, v)
		}
	}
}

func TestSoftClipNonPositiveDriveIsPassThrough(t *testing.T) {
	got := SoftClip{Drive: 0}.Apply(sound(0.4, -0.7))
	assertSamples(t, got, []float64{0.4, -0.7})
}

func TestFuzzAddsAsymmetricGrit(t *testing.T) {
	// Fuzz drives hard and clips positive and negative halves differently,
	// which is what gives it a buzzier character than SoftClip.
	out := Fuzz{Drive: 10, Bias: 0.3}.Apply(sound(0.4, -0.4)).Samples()
	if math.Abs(out[0]) == math.Abs(out[1]) {
		t.Errorf("fuzz is symmetric: %v vs %v, want asymmetric", out[0], out[1])
	}
	for i, v := range out {
		if v > 1 || v < -1 {
			t.Errorf("sample[%d] = %v, want within [-1, 1]", i, v)
		}
	}
}

func TestFuzzIsDeterministic(t *testing.T) {
	a := Fuzz{Drive: 6, Bias: 0.2}.Apply(sound(0.3, -0.6, 0.9)).Samples()
	b := Fuzz{Drive: 6, Bias: 0.2}.Apply(sound(0.3, -0.6, 0.9)).Samples()
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("nondeterministic at %d: %v vs %v", i, a[i], b[i])
		}
	}
}

func TestFuzzNonPositiveDriveIsPassThrough(t *testing.T) {
	got := Fuzz{Drive: 0, Bias: 0.5}.Apply(sound(0.4, -0.7))
	assertSamples(t, got, []float64{0.4, -0.7})
}

func TestNormalizeScalesPeakToTarget(t *testing.T) {
	got := Normalize{Peak: 1.0}.Apply(sound(0.25, -0.5, 0.125))
	assertSamples(t, got, []float64{0.5, -1.0, 0.25})
}

func TestNormalizeAttenuatesTooLoudInput(t *testing.T) {
	got := Normalize{Peak: 0.5}.Apply(sound(2.0, -1.0))
	assertSamples(t, got, []float64{0.5, -0.25})
}

func TestNormalizeSilenceIsUnchanged(t *testing.T) {
	// Dividing by a zero peak would produce NaN; silence must stay silence.
	got := Normalize{Peak: 1.0}.Apply(sound(0, 0, 0))
	assertSamples(t, got, []float64{0, 0, 0})
}

func TestNormalizeNonPositivePeakIsPassThrough(t *testing.T) {
	got := Normalize{Peak: 0}.Apply(sound(0.4, -0.7))
	assertSamples(t, got, []float64{0.4, -0.7})
}

func TestEffectsDoNotMutateInput(t *testing.T) {
	in := sound(0.4, -0.8, 0.6)
	before := in.Samples()
	for name, effect := range map[string]interface {
		Apply(audio.Sound) audio.Sound
	}{
		"Gain":      Gain{Factor: 3},
		"HardClip":  HardClip{Threshold: 0.2},
		"SoftClip":  SoftClip{Drive: 4},
		"Fuzz":      Fuzz{Drive: 6, Bias: 0.3},
		"Normalize": Normalize{Peak: 1},
	} {
		effect.Apply(in)
		after := in.Samples()
		for i := range before {
			if before[i] != after[i] {
				t.Errorf("%s mutated input at %d: %v -> %v", name, i, before[i], after[i])
			}
		}
	}
}
