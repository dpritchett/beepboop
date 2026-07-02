package presets

import "beepboop/internal/audio"

type Preset struct {
	Name  string
	Sound audio.Sound
}

func List() []string {
	return []string{"alarm-basic", "alarm-urgent", "soft-reminder"}
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
	default:
		return Preset{}, false
	}
}
