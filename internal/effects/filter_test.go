package effects

import (
	"math"
	"testing"

	"beepboop/internal/audio"
)

// tone builds a full-scale sine at frequency for one second.
func tone(sampleRate int, frequency float64) audio.Sound {
	samples := make([]float64, sampleRate)
	for i := range samples {
		samples[i] = math.Sin(2 * math.Pi * frequency * float64(i) / float64(sampleRate))
	}
	return audio.NewSound(sampleRate, samples)
}

func peakOf(s audio.Sound) float64 {
	loudest := 0.0
	for _, v := range s.Samples() {
		loudest = math.Max(loudest, math.Abs(v))
	}
	return loudest
}

func TestHighPassAttenuatesBelowCutoff(t *testing.T) {
	// The whole point: a low drone under the cutoff should lose most of its
	// level while the material above it survives.
	low := peakOf(HighPass{Cutoff: 400}.Apply(tone(22050, 80)))
	high := peakOf(HighPass{Cutoff: 400}.Apply(tone(22050, 2000)))

	if low > 0.5 {
		t.Errorf("80Hz through a 400Hz high-pass = %v, want well attenuated", low)
	}
	if high < 0.8 {
		t.Errorf("2000Hz through a 400Hz high-pass = %v, want mostly intact", high)
	}
}

func TestHighPassPolesSteepenTheSlope(t *testing.T) {
	gentle := peakOf(HighPass{Cutoff: 400, Poles: 1}.Apply(tone(22050, 100)))
	steep := peakOf(HighPass{Cutoff: 400, Poles: 4}.Apply(tone(22050, 100)))

	if steep >= gentle {
		t.Errorf("four-pole output %v is not below one-pole %v", steep, gentle)
	}
}

func TestHighPassKeepsLoopsSeamless(t *testing.T) {
	// These run on looping beds, so the filter has to wrap its state rather
	// than starting cold and leaving a step at the buffer boundary.
	loop := audio.Loop(audio.LoopSpec{
		SampleRate: 22050,
		Duration:   1.0,
		Partials:   audio.SawPartials(110, 8000),
		Noise:      0.4,
		Seed:       5,
	})
	samples := HighPass{Cutoff: 300, Poles: 2}.Apply(loop).Samples()

	worst := 0.0
	for i := 1; i < len(samples); i++ {
		worst = math.Max(worst, math.Abs(samples[i]-samples[i-1]))
	}
	if seam := math.Abs(samples[0] - samples[len(samples)-1]); seam > worst {
		t.Errorf("seam jump %v exceeds largest in-buffer step %v", seam, worst)
	}
}

func TestHighPassNonPositiveCutoffIsPassThrough(t *testing.T) {
	in := tone(8000, 200)
	got := HighPass{Cutoff: 0}.Apply(in).Samples()
	for i, v := range in.Samples() {
		if math.Abs(got[i]-v) > 1e-9 {
			t.Fatalf("sample[%d] = %v, want %v", i, got[i], v)
		}
	}
}

func TestHighPassDoesNotMutateInput(t *testing.T) {
	in := tone(8000, 200)
	before := in.Samples()
	HighPass{Cutoff: 400, Poles: 2}.Apply(in)
	for i, v := range in.Samples() {
		if before[i] != v {
			t.Fatalf("input mutated at %d", i)
		}
	}
}

func TestHighPassIsDeterministic(t *testing.T) {
	in := tone(8000, 200)
	a := HighPass{Cutoff: 400, Poles: 2}.Apply(in).Samples()
	b := HighPass{Cutoff: 400, Poles: 2}.Apply(in).Samples()
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("nondeterministic at %d", i)
		}
	}
}
