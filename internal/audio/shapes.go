package audio

// Classic oscillator shapes as harmonic stacks.
//
// These are band-limited by construction rather than by a filter. A sawtooth
// is the harmonic series at 1/n amplitude, a square is its odd harmonics at
// 1/n, a triangle its odd harmonics at 1/n squared with alternating sign.
// Building them additively means never generating a partial above the limit
// in the first place, so there is nothing to alias and no anti-aliasing
// machinery to write. The naive approach, sampling an ideal ramp or step,
// would need exactly that machinery.
//
// The limit is the caller's brightness control as much as a safety rail:
// stopping the stack early is how you get a mellow saw. Pass something at or
// below Nyquist; Loop drops anything above it regardless.

// SawPartials returns a sawtooth's harmonic series: every harmonic at 1/n.
// The richest of the three shapes, and the backbone of most synth leads and
// basses.
func SawPartials(fundamental, limit float64) []Partial {
	return harmonics(fundamental, limit, 1, func(n float64) float64 {
		return 1 / n
	})
}

// SquarePartials returns a square wave's harmonic series: odd harmonics only,
// at 1/n. Hollow and woody next to a saw, because the even harmonics that
// fill a saw out are simply absent.
func SquarePartials(fundamental, limit float64) []Partial {
	return harmonics(fundamental, limit, 2, func(n float64) float64 {
		return 1 / n
	})
}

// TrianglePartials returns a triangle's harmonic series: odd harmonics at
// 1/n squared, alternating sign. The squared falloff makes it far mellower
// than a square, close to a sine with a little edge, which is what makes it
// useful for pads that should not dominate.
func TrianglePartials(fundamental, limit float64) []Partial {
	sign := 1.0
	return harmonics(fundamental, limit, 2, func(n float64) float64 {
		gain := sign / (n * n)
		sign = -sign
		return gain
	})
}

// harmonics builds a stack from fundamental up to limit, stepping the harmonic
// number by step (1 for every harmonic, 2 for odd only) and taking each
// amplitude from gain.
func harmonics(fundamental, limit float64, step int, gain func(n float64) float64) []Partial {
	if fundamental <= 0 || limit < fundamental {
		return nil
	}
	var partials []Partial
	for n := 1; ; n += step {
		frequency := fundamental * float64(n)
		if frequency > limit {
			break
		}
		partials = append(partials, Partial{
			Frequency: frequency,
			Gain:      gain(float64(n)),
		})
	}
	return partials
}
