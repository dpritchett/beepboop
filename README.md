# beepboop

Daniel\s cool noise studio.

Beepboop is a small Go audio playground for defining synthetic alarm/noise ideas in code, previewing them locally, and exporting durable audio assets for other projects.

## Goals

- Define sounds as inspectable Go code rather than one-off DAW sessions.
- Preview generated tones quickly while tuning envelopes, rhythms, layers, and timbre.
- Export deterministic WAV files for apps such as Timerbox to vendor as low-resource playback assets.
- Keep synthesis code boring, testable, and easy to delete or replace.

## Non-goals

- Be a full DAW.
- Ship a GUI before the CLI/library workflow is useful.
- Make Timerbox depend on a large synthesis runtime.
- Optimize for professional audio production before the generated sounds are pleasant and useful.

## First Slice

1. Create a Go module and CLI.
2. Implement a tiny PCM/WAV writer.
3. Add one preset alarm tone similar to the Timerbox prototype.
4. Add `preview` using a local player such as `aplay`.
5. Add `render` to write a `.wav` file under `dist/`.
6. Add deterministic tests for WAV headers and sample generation.

## Possible Command Shape

```sh
beepboop list
beepboop preview alarm1
beepboop render alarm1 dist/alarm1.wav
```

## Design Notes

- Prefer pure Go synthesis and standard-library WAV output at first.
- Model sounds as presets made of oscillators, envelopes, gates, sweeps, and optional noise.
- Tests should assert structure and determinism, not subjective taste.
- Exported audio assets are build artifacts; source presets are the durable truth.
