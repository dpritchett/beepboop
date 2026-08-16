# beepboop

Daniel's reusable sound pipeline lab.

Beepboop is a small Go audio playground for building deterministic sound
pipelines. The first slice renders synthetic alert sounds, but the project is
intended to grow into a place where agents can define reusable sources,
effects chains, voice recipes, and batch exports without hand-editing binary
audio files.

## Goals

- Define sounds and processing chains as inspectable source.
- Render deterministic WAV assets from source recipes.
- Support pure-Go synthesis primitives for alarms, reminders, and UI sounds.
- Support local voice generation through optional external engines such as
  Piper.
- Feed generated or imported audio through reusable effects chains: gain,
  clipping, fuzz, distortion, filters, delay, limiting, and related tools.
- Batch-render named lines or presets into individual files.
- Keep the CLI thin; it is an execution surface, not the product center.

## Non-goals

- Be a full DAW.
- Ship a GUI before the library and recipe workflow is useful.
- Make Timerbox depend on a large synthesis runtime.
- Require Piper, MP3 encoders, or playback tools for the core test suite.
- Check in large voice models or unstable generated experiments.
- Optimize for professional audio production before the generated sounds are
  pleasant and useful.

## Current Slice

The repository currently includes:

- A Go module.
- A thin `beepboop` CLI with `list`, `render`, `bake`, and `preview`.
- Deterministic sample generation for basic alert presets.
- A pure-Go PCM16 mono WAV reader and writer.
- A `source -> effects -> export` pipeline with gain, clipping, fuzz, and
  normalization effects.
- An optional Piper voice source behind an injected command runner.
- JSON recipes for batch rendering, with spoken companion labels.
- Optional local preview through `aplay`, `paplay`, or `ffplay`.
- `dist/alarm-basic.wav`, rendered from source as the first artifact.

## Recipes

A recipe is a JSON file describing a batch of sounds, the effects chain
applied to them, and where the voice comes from:

```json
{
  "voice": {"model": "voices/en_US-lessac-medium.onnx"},
  "labels": true,
  "effects": [{"type": "normalize", "peak": 0.8}],
  "outputs": [
    {"name": "turn-ready", "preset": "turn-ready"},
    {"name": "build-done", "say": "build finished",
     "effects": [{"type": "fuzz", "drive": 8, "bias": 0.3}]}
  ]
}
```

Each output names a `preset` or a line to `say`, never both. Recipe-level
`effects` apply to every output; an output's own `effects` replace them, and an
explicit empty list opts that output out. Effect types are `gain`, `hardclip`,
`softclip`, `fuzz`, and `normalize`.

`recipes/dist.json` rebuilds every preset in `dist/`.

## Spoken Labels

With `"labels": true`, every output gets a companion file with Piper speaking
the sound's name, written beside it as `<name>.label.wav`:

```text
dist/turn-ready.wav
dist/turn-ready.label.wav
```

The name is spoken with `-`, `_`, and `.` as word breaks, so `turn-ready-soft`
is read "turn ready soft". Set an output's `label` to override the text, or to
`""` to skip that one. Labels deliberately skip the effects chain so the spoken
name stays intelligible next to a heavily distorted sound.

Labels need Piper. Point `BEEPBOOP_VOICE_MODEL` at a `.onnx` voice to override
the path in a recipe, which keeps checked-in recipes machine-independent.

## Target Shape

```text
cmd/beepboop          thin CLI wrapper
internal/audio        sample buffers, synthesis primitives, WAV loading helpers
internal/effects      gain, clipping, fuzz, filters, delay, limiting
internal/pipeline     reusable source -> effects -> export orchestration
internal/presets      named synthetic sounds
internal/voice        optional TTS adapters, starting with Piper
internal/player       optional local preview helpers
internal/wav          WAV read/write
recipes/              checked-in pipeline and voice recipes
dist/                 stable rendered artifacts
```

## Example Commands

```sh
beepboop list
beepboop render alarm-basic dist/alarm-basic.wav
beepboop bake recipes/dist.json dist
beepboop preview alarm-basic
```

## Design Notes

- Prefer pure Go synthesis and standard-library WAV output for core features.
- Model sounds as presets made of oscillators, envelopes, gates, sweeps, and optional noise.
- Model voice work as optional source adapters around external tools. Piper is
  the first expected adapter.
- Tests should assert structure, bounds, duration, and determinism, not
  subjective taste.
- Exported audio assets are build artifacts; source presets are the durable truth.
- External tools should be injected through narrow interfaces so missing
  players, TTS engines, or encoders can be reported cleanly.
