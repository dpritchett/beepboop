package audio

import "math"

type Sound interface {
	SampleRate() int
	Samples() []float64
}

type SoundSpec struct {
	SampleRate  int
	Duration    float64
	Gain        float64
	Frequencies []float64
	Step        float64
}

type generatedSound struct {
	sampleRate int
	samples    []float64
}

func (s generatedSound) SampleRate() int {
	return s.sampleRate
}

func (s generatedSound) Samples() []float64 {
	out := make([]float64, len(s.samples))
	copy(out, s.samples)
	return out
}

func AlternatingTone(spec SoundSpec) Sound {
	if spec.SampleRate <= 0 {
		spec.SampleRate = 44100
	}
	total := int(math.Round(spec.Duration * float64(spec.SampleRate)))
	samples := make([]float64, total)
	if total == 0 || len(spec.Frequencies) == 0 || spec.Step <= 0 {
		return generatedSound{sampleRate: spec.SampleRate, samples: samples}
	}

	stepSamples := max(1, int(math.Round(spec.Step*float64(spec.SampleRate))))
	gain := math.Max(0, math.Min(1, spec.Gain))
	attack := max(1, int(0.005*float64(spec.SampleRate)))
	release := max(1, int(0.020*float64(spec.SampleRate)))

	for i := range samples {
		step := i / stepSamples
		frequency := spec.Frequencies[step%len(spec.Frequencies)]
		phase := 2 * math.Pi * frequency * float64(i) / float64(spec.SampleRate)
		envelope := 1.0
		if i < attack {
			envelope = float64(i) / float64(attack)
		}
		if remaining := total - 1 - i; remaining < release {
			envelope *= float64(remaining) / float64(release)
		}
		samples[i] = math.Sin(phase) * gain * envelope
	}

	return generatedSound{sampleRate: spec.SampleRate, samples: samples}
}
