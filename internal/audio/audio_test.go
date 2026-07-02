package audio

import "testing"

func TestAlternatingToneIsDeterministic(t *testing.T) {
	sound := AlternatingTone(SoundSpec{
		SampleRate:  8000,
		Duration:    0.5,
		Gain:        0.5,
		Frequencies: []float64{440, 660},
		Step:        0.125,
	})

	first := sound.Samples()
	second := sound.Samples()

	if len(first) != 4000 {
		t.Fatalf("len(samples) = %d, want 4000", len(first))
	}
	if len(second) != len(first) {
		t.Fatalf("second render len = %d, want %d", len(second), len(first))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("sample %d differs between renders: %v != %v", i, first[i], second[i])
		}
		if first[i] < -1 || first[i] > 1 {
			t.Fatalf("sample %d = %v, outside [-1, 1]", i, first[i])
		}
	}
	if first[0] != 0 {
		t.Fatalf("first sample = %v, want 0", first[0])
	}
	if first[1] <= 0 {
		t.Fatalf("second sample = %v, want positive sine rise", first[1])
	}
}
