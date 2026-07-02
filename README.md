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
- A thin `beepboop` CLI with `list`, `render`, and `preview`.
- Deterministic sample generation for basic alert presets.
- A pure-Go PCM16 mono WAV writer.
- Optional local preview through `aplay`, `paplay`, or `ffplay`.
- `dist/alarm-basic.wav`, rendered from source as the first artifact.

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
