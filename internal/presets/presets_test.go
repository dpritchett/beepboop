package presets

import (
	"math"
	"testing"
)

func TestListIncludesMVPPresets(t *testing.T) {
	got := List()
	want := []string{
		"alarm-basic",
		"alarm-urgent",
		"soft-reminder",
		"turn-ready",
		"turn-ready-soft",
		"notify-blip",
		"notify-chime",
		"flight-fast",
		"flight-slow",
	}

	if len(got) != len(want) {
		t.Fatalf("len(List()) = %d, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("List()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestResolveAlarmBasic(t *testing.T) {
	preset, ok := Resolve("alarm-basic")
	if !ok {
		t.Fatal("Resolve(alarm-basic) ok = false")
	}
	if preset.Name != "alarm-basic" {
		t.Fatalf("preset name = %q, want alarm-basic", preset.Name)
	}
	if preset.Sound.SampleRate() != 44100 {
		t.Fatalf("sample rate = %d, want 44100", preset.Sound.SampleRate())
	}
	if samples := preset.Sound.Samples(); len(samples) != 35280 {
		t.Fatalf("len(samples) = %d, want 35280", len(samples))
	}
}

func TestResolveTurnReady(t *testing.T) {
	preset, ok := Resolve("turn-ready")
	if !ok {
		t.Fatal("Resolve(turn-ready) ok = false")
	}
	if preset.Name != "turn-ready" {
		t.Fatalf("preset name = %q, want turn-ready", preset.Name)
	}
	if preset.Sound.SampleRate() != 44100 {
		t.Fatalf("sample rate = %d, want 44100", preset.Sound.SampleRate())
	}
	if samples := preset.Sound.Samples(); len(samples) != 12348 {
		t.Fatalf("len(samples) = %d, want 12348", len(samples))
	}
}

func TestResolveTurnReadySoft(t *testing.T) {
	preset, ok := Resolve("turn-ready-soft")
	if !ok {
		t.Fatal("Resolve(turn-ready-soft) ok = false")
	}
	if preset.Name != "turn-ready-soft" {
		t.Fatalf("preset name = %q, want turn-ready-soft", preset.Name)
	}
	if samples := preset.Sound.Samples(); len(samples) != 4410 {
		t.Fatalf("len(samples) = %d, want 4410", len(samples))
	}
}

func TestFlightPresetsAreMarkedAsLoops(t *testing.T) {
	// A consumer has to know to set loop playback and skip a fade-out. The
	// one-shot alerts must not be flagged, or they will repeat forever.
	for _, tc := range []struct {
		name string
		loop bool
	}{
		{"flight-fast", true},
		{"flight-slow", true},
		{"notify-blip", false},
		{"alarm-basic", false},
	} {
		preset, ok := Resolve(tc.name)
		if !ok {
			t.Fatalf("Resolve(%s) ok = false", tc.name)
		}
		if preset.Loop != tc.loop {
			t.Errorf("%s Loop = %v, want %v", tc.name, preset.Loop, tc.loop)
		}
	}
}

func TestFlightPresetsWrapWithoutAClick(t *testing.T) {
	// These play on repeat while the camera moves. A discontinuity at the
	// wrap point is an audible tick on every single loop.
	for _, name := range []string{"flight-fast", "flight-slow"} {
		preset, ok := Resolve(name)
		if !ok {
			t.Fatalf("Resolve(%s) ok = false", name)
		}
		samples := preset.Sound.Samples()
		worst := 0.0
		for i := 1; i < len(samples); i++ {
			worst = math.Max(worst, math.Abs(samples[i]-samples[i-1]))
		}
		if seam := math.Abs(samples[0] - samples[len(samples)-1]); seam > worst {
			t.Errorf("%s seam jump %v exceeds largest in-buffer step %v", name, seam, worst)
		}
	}
}

func TestFlightPresetsDiffer(t *testing.T) {
	// Fast should read as brighter and busier than slow, or the two speeds
	// are not actually distinguishable in flight.
	step := func(name string) float64 {
		preset, _ := Resolve(name)
		samples := preset.Sound.Samples()
		total := 0.0
		for i := 1; i < len(samples); i++ {
			total += math.Abs(samples[i] - samples[i-1])
		}
		return total / float64(len(samples))
	}

	fast, slow := step("flight-fast"), step("flight-slow")
	if fast <= slow {
		t.Errorf("flight-fast average step %v is not above flight-slow %v", fast, slow)
	}
}

func TestFlightPresetsAreDeterministic(t *testing.T) {
	for _, name := range []string{"flight-fast", "flight-slow"} {
		first, _ := Resolve(name)
		second, _ := Resolve(name)
		a, b := first.Sound.Samples(), second.Sound.Samples()
		for i := range a {
			if a[i] != b[i] {
				t.Fatalf("%s nondeterministic at %d: %v vs %v", name, i, a[i], b[i])
			}
		}
	}
}

func TestResolveNotifyPresets(t *testing.T) {
	for _, tc := range []struct {
		name    string
		samples int
	}{
		{"notify-blip", 6174},
		{"notify-chime", 19845},
	} {
		preset, ok := Resolve(tc.name)
		if !ok {
			t.Fatalf("Resolve(%s) ok = false", tc.name)
		}
		if preset.Name != tc.name {
			t.Fatalf("preset name = %q, want %q", preset.Name, tc.name)
		}
		if got := len(preset.Sound.Samples()); got != tc.samples {
			t.Fatalf("%s len(samples) = %d, want %d", tc.name, got, tc.samples)
		}
	}
}
