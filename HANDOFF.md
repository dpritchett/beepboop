# Handoff

Pickup notes for the next beepboop session. Delete or trim once absorbed.

## Where things stand (2026-08-15)

Every issue on the near-term roadmap is closed except MP3 export (#6). The
library now covers the whole `source -> effects -> export` path with a recipe
layer on top.

- **#1 core effects** — `internal/effects` has `Gain`, `HardClip`, `SoftClip`,
  `Fuzz`, `Normalize`. Pure, non-mutating, invalid config degrades to
  pass-through rather than silence.
- **#2 WAV readback** — `wav.ReadPCM16Mono` walks RIFF chunks (so a Piper
  `LIST` chunk is not decoded as audio) and `pipeline.WAVSource` feeds it back
  into a pipeline.
- **#5 Piper voice** — `internal/voice` wraps Piper behind an injected
  `Runner` and `LookPath`. `Piper.Available()` probes config and PATH without
  synthesizing.
- **#3 recipes** — `internal/recipe` parses JSON recipes and batch-renders
  them. `beepboop bake <recipe.json> <outdir>` is the CLI surface.
- **Spoken labels** — `"labels": true` renders `<name>.label.wav` beside each
  sound with Piper reading the name aloud. Labels skip the effects chain but
  are normalized to the peak of the sound they name.
- **`beepboop inspect`** — reports rate, length, and peak for rendered WAVs,
  so checking an artifact is a repo capability instead of a throwaway script.

`go build ./... && go vet ./... && go test ./...` all green.

## Voice is live

Piper is installed (`pip install piper-tts`, binary at the mise python 3.12
prefix) with `en_US-lessac-medium` in `~/.local/share/piper/voices/`. Both
recipes render end to end:

```sh
export BEEPBOOP_VOICE_MODEL=~/.local/share/piper/voices/en_US-lessac-medium.onnx
beepboop bake recipes/dist.json dist          # 7 presets + 7 spoken labels
beepboop bake recipes/navigator.json dist/navigator   # 19 UI voice lines
```

All 14 `dist/` artifacts verified byte-identical across re-bakes, and
`dist/alarm-basic.wav` still matches the version committed in July.

Two Piper behaviors are load-bearing and easy to undo by accident, both
guarded by tests in `internal/voice`:

- Without `--output_file -` piper plays to the speakers and writes nothing.
- Without `--noise-scale 0 --noise-w-scale 0` the same line renders
  differently every run (observed 30764, 26668, and 30252 bytes for one line).

## Music: where it stands (2026-08-16)

The engine now has what music needs except rhythm and movement:
`audio.Sequence` (notes in time with attack/decay envelopes, wrapping across
the loop boundary), harmonic shapes (`SawPartials`, `SquarePartials`,
`TrianglePartials`), recipe `layers` for mixing a pad under a pattern, and a
loop-safe `HighPass`. All drivable from JSON, no Go edits: see
`recipes/music-lab.json`.

**Daniel's verdict: `music-3-both` and `music-4-sparse` work, `music-1` and
`music-2` do not.** The two that work are opposites, which is the useful part:
`music-3` grooves at a 0.5s pulse, roughly 120 BPM, and `music-4` is ambient
with events every two seconds or more. Both rejected tracks sit at exactly 1.0s
spacing.

Read that as a rule: **commit to a groove or commit to ambient, and stay out of
the one-event-per-second middle**, where the ear tracks the repetition but
nothing drives. It also means the two survivors map onto the two things Daniel
originally asked for, a chill bed (`music-4`) and a driving one (`music-3`),
rather than one being better than the other.

Two rounds of feedback got us here, both worth remembering:

- Triangle stacks read as single drones. The 1/n squared falloff puts nearly
  all energy in the fundamental, so a triangle stack is barely distinguishable
  from a sine. Saw at 1/n is what makes a chord sound like a chord.
- Stacked voices pile up under 200Hz and the sum reads as mud. Raising the
  register and high-passing at 180 to 220Hz fixed it; turning voices down did
  not.

### Next for music

1. **Drums**, on the `music-3` branch only. Noise bursts with fast envelopes:
   kick is a pitch-swept sine, snare noise plus tone, hats filtered noise.
   Small now that sequencing exists. Start from `music-3-both` and put a kick
   on the 0.5s grid it already has. Leave `music-4` percussion-free; sparse is
   what makes it work.
2. **Filter movement over time.** A resonant filter with a moving cutoff is
   what makes electronic music breathe rather than repeat. This is the one
   genuinely new piece of DSP left, and the payoff sound of the genre.
3. Longer loops for the ambient branch. Eight seconds is short enough to
   notice; `music-4` would carry 30 to 60 seconds cheaply since it is sparse,
   and length is the cheapest cure for hearing the repeat.

## Next up

1. **#6 MP3 export** — an `Exporter` shelling out to an injected encoder,
   mirroring how Piper is wired. The navigator set is 764K of WAV for a
   browser project, so this is the next real need, not a nice-to-have.
2. Consider synthetic SFX rather than speech for the navigator's
   high-frequency events (`select`, `capture`, `release`, `fast-on/off`); a
   one-second spoken line per click will overlap itself.
3. Consider an espeak adapter as a zero-install fallback voice; the `Runner`
   interface already accommodates it.

## Conventions to keep

- TDD: write the red test first, confirm it fails for the right reason.
- Std-lib only for core; keep Piper/players/encoders optional and injected.
- Batch operations validate everything before writing anything. A run that
  cannot finish must not leave a half-rendered directory behind.
- One-line commit messages.
- Deterministic output; exported WAV/MP3 are artifacts, source presets are the
  durable truth. `dist/*.wav` is gitignored except `alarm-basic.wav`.
