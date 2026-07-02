package presets

import "testing"

func TestListIncludesMVPPresets(t *testing.T) {
	got := List()
	want := []string{"alarm-basic", "alarm-urgent", "soft-reminder"}

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
