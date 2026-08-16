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
  sound with Piper reading the name aloud. Labels skip the effects chain.

`go build ./... && go vet ./... && go test ./...` all green.

## Blocked

**Piper is not installed on this host**, so no label has actually been
rendered. `beepboop bake recipes/dist.json dist` fails the preflight with a
clear message and writes nothing. Installing Piper plus one `.onnx` voice is
the only thing between here and a fully labeled `dist/`.

Notable: `uv` and `espeak` are already on this box; `apt` has no `piper-tts`.

## Next up

1. **Install Piper**, then `beepboop bake recipes/dist.json dist` to produce
   the labeled artifact set.
2. **#6 MP3 export** — an `Exporter` shelling out to an injected encoder,
   mirroring how Piper is wired.
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
