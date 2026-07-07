# Handoff

Pickup notes for the next beepboop session. Delete or trim once absorbed.

## Where things stand (2026-07-07)

The `source -> effects -> export` pipeline spine (issue #4) is **done and
committed** (`Add source-effects-export pipeline abstraction (closes #4)`).

- `internal/pipeline` defines `Source`, `Effect`, `Exporter` interfaces and a
  `Pipeline{Source, Effects, Exporter}.Run()` composer. Effects apply
  left-to-right, nil effects skip, missing source/exporter return
  `ErrNoSource`/`ErrNoExporter`.
- Adapters: `StaticSource` (wraps any `audio.Sound`), `WAVExporter` (injected
  `io.Writer`).
- `audio.NewSound(rate, samples)` is a new public constructor so sources and
  effects can build `Sound` values.
- CLI `render` now composes `StaticSource -> WAVExporter`. Output is
  byte-identical to `dist/alarm-basic.wav`.
- `go build ./... && go vet ./... && go test ./...` all green.

## Next up

Two zero-dependency leaves now slot straight into the spine:

1. **#1 core effects** — implement `Gain`, `HardClip`, `SoftClip`, `Fuzz`,
   `Normalize` as `Effect`s in `internal/effects`. This fills the `Effects`
   slice that only test doubles exercise today. Recommended next.
2. **#2 WAV readback** — a `WAVSource` implementing `Source`, so rendered or
   imported PCM16 mono audio can re-enter pipelines. Std-lib only; pairs with
   the existing `internal/wav` writer.

Then #3 (recipes) and #5/#6 (Piper, MP3) build on top: recipes drive the
pipeline, Piper is a `Source`, MP3 is an `Exporter`.

## Conventions to keep

- TDD: write the red test first (see `internal/pipeline/pipeline_test.go` for
  the pattern — test-local effect doubles, ordering + determinism + error paths).
- Std-lib only for core; keep Piper/players/encoders optional and injected.
- One-line commit messages.
- Deterministic output; exported WAV/MP3 are artifacts, source presets are the
  durable truth.
