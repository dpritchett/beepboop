package presets

import "beepboop/internal/audio"

type Preset struct {
	Name  string
	Sound audio.Sound
}

func List() []string {
	return []string{
		"alarm-basic",
		"alarm-urgent",
		"soft-reminder",
		"turn-ready",
		"notify-blip",
		"notify-chime",
	}
}

func Resolve(name string) (Preset, bool) {
	switch name {
	case "alarm-basic":
		return Preset{
			Name: name,
			Sound: audio.AlternatingTone(audio.SoundSpec{
				SampleRate:  44100,
				Duration:    0.8,
				Gain:        0.55,
				Frequencies: []float64{880, 660},
				Step:        0.12,
			}),
		}, true
	case "alarm-urgent":
		return Preset{
			Name: name,
			Sound: audio.AlternatingTone(audio.SoundSpec{
				SampleRate:  44100,
				Duration:    0.9,
				Gain:        0.8,
				Frequencies: []float64{1040, 780},
				Step:        0.08,
			}),
		}, true
	case "soft-reminder":
		return Preset{
			Name: name,
			Sound: audio.AlternatingTone(audio.SoundSpec{
				SampleRate:  44100,
				Duration:    0.7,
				Gain:        0.32,
				Frequencies: []float64{660, 880},
				Step:        0.22,
			}),
		}, true
	case "turn-ready":
		// Soft "it's your turn" boop: a gentle rising E5 -> A5 (a warm
		// perfect fourth). Low gain and mid frequencies keep it calm
		// rather than alarming; duration is exactly two steps so it plays
		// a clean two-note "doo-doo" and stops.
		return Preset{
			Name: name,
			Sound: audio.AlternatingTone(audio.SoundSpec{
				SampleRate:  44100,
				Duration:    0.28,
				Gain:        0.28,
				Frequencies: []float64{659.25, 880.0},
				Step:        0.14,
			}),
		}, true
	case "notify-blip":
		// Minimal single soft blip for subtle, low-attention nudges.
		return Preset{
			Name: name,
			Sound: audio.AlternatingTone(audio.SoundSpec{
				SampleRate:  44100,
				Duration:    0.14,
				Gain:        0.25,
				Frequencies: []float64{880.0},
				Step:        0.14,
			}),
		}, true
	case "notify-chime":
		// Gentle three-note descending chime (C6 -> G5 -> E5) for a
		// slightly richer, still-soft completion cue.
		return Preset{
			Name: name,
			Sound: audio.AlternatingTone(audio.SoundSpec{
				SampleRate:  44100,
				Duration:    0.45,
				Gain:        0.3,
				Frequencies: []float64{1046.5, 783.99, 659.25},
				Step:        0.15,
			}),
		}, true
	default:
		return Preset{}, false
	}
}
