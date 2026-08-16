package presets

import "beepboop/internal/audio"

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
		// Jet turbine for fast camera movement. Three ingredients do the
		// work: a low spool rumble on 110 Hz and its harmonics, a pair of
		// high blade-pass partials for the characteristic turbine shriek,
		// and a thick noise bed for moving air. The noise carries most of
		// the level; the partials give it a pitch to sit on.
		return Preset{
			Name: name,
			Loop: true,
			Sound: audio.Loop(audio.LoopSpec{
				SampleRate: 22050,
				Duration:   2.0,
				Partials: []audio.Partial{
					{Frequency: 110, Gain: 0.50},
					{Frequency: 220, Gain: 0.28},
					{Frequency: 330, Gain: 0.16},
					{Frequency: 2180, Gain: 0.11},
					{Frequency: 3270, Gain: 0.07},
				},
				Noise:     0.85,
				NoiseTone: 0.30,
				Seed:      1101,
			}),
		}, true
	case "flight-slow":
		// Droning buzz for slow movement: darker, calmer, and tonal rather
		// than airy. The 61 and 62 Hz pair beat against each other about
		// once a second, which keeps a long loop from sounding frozen, and
		// the noise bed is rolled well off so it reads as a hum, not wind.
		return Preset{
			Name: name,
			Loop: true,
			Sound: audio.Loop(audio.LoopSpec{
				SampleRate: 22050,
				Duration:   2.0,
				Partials: []audio.Partial{
					{Frequency: 61, Gain: 0.50},
					{Frequency: 62, Gain: 0.42, Phase: 0.25},
					{Frequency: 122, Gain: 0.20},
					{Frequency: 183, Gain: 0.09},
				},
				Noise:     0.22,
				NoiseTone: 0.08,
				Seed:      2202,
			}),
		}, true
	default:
		return Preset{}, false
	}
}
