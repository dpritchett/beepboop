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
