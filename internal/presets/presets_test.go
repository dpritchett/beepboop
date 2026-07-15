package presets

import "testing"

func TestListIncludesMVPPresets(t *testing.T) {
	got := List()
	want := []string{
		"alarm-basic",
		"alarm-urgent",
		"soft-reminder",
		"turn-ready",
		"notify-blip",
		"notify-chime",
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
