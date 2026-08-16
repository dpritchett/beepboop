package presets

import (
	"beepboop/internal/audio"
	"beepboop/internal/effects"
)

// bedLevel sets a continuous bed's peak. audio.Loop renders at full scale so
// callers can decide, and a flight bed has to sit well under the voice lines
// it plays behind rather than competing with them.
func bedLevel(sound audio.Sound, peak float64) audio.Sound {
	return effects.Normalize{Peak: peak}.Apply(sound)
}

type Preset struct {
	Name  string
	Sound audio.Sound
	// Loop marks a sound built to repeat seamlessly for as long as some
	// state holds, rather than a one-shot that plays once and stops.
	Loop bool
}

func List() []string {
	return []string{
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
	case "turn-ready-soft":
		// Low-attention companion to turn-ready: a single short, quiet E5
		// blip for "Claude moved again, you're aware" moments where you are
		// not actually blocking anyone. Softer, shorter, and lower-energy
		// than turn-ready so it registers without demanding a response.
		return Preset{
			Name: name,
			Sound: audio.AlternatingTone(audio.SoundSpec{
				SampleRate:  44100,
				Duration:    0.10,
				Gain:        0.18,
				Frequencies: []float64{659.25},
				Step:        0.10,
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
	case "flight-fast":
		// Turbo version of flight-slow, not a different sound. Same airflow
		// recipe with the tone filter opened up (0.16 to 0.30) and dropped
		// to two poles, which lets through the harmonics that read as speed,
		// plus a third partial for body. Keeping the family shared is what
		// lets the two crossfade as one source changing speed.
		return Preset{
			Name: name,
			Loop: true,
			Sound: bedLevel(audio.Loop(audio.LoopSpec{
				SampleRate: 22050,
				Duration:   2.0,
				Partials: []audio.Partial{
					{Frequency: 90, Gain: 0.30},
					{Frequency: 180, Gain: 0.18},
					{Frequency: 270, Gain: 0.10},
				},
				Noise:      0.95,
				NoiseTone:  0.30,
				NoisePoles: 2,
				Seed:       1102,
			}), 0.38),
		}, true
	case "flight-slow":
		// The baseline movement bed: airflow, chosen by ear over rumble,
		// turbine, and engine candidates. flight-fast is a hotter version of
		// this same recipe rather than a different sound, so the two
		// crossfade as one source changing speed instead of swapping.
		return Preset{
			Name: name,
			Loop: true,
			Sound: bedLevel(audio.Loop(audio.LoopSpec{
				SampleRate: 22050,
				Duration:   2.0,
				Partials: []audio.Partial{
					{Frequency: 90, Gain: 0.22},
					{Frequency: 180, Gain: 0.12},
				},
				Noise:      0.95,
				NoiseTone:  0.16,
				NoisePoles: 3,
				Seed:       1102,
			}), 0.26),
		}, true
	default:
		return Preset{}, false
	}
}
