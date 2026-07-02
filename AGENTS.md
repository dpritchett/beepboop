# beepboop

Go-based sound pipeline lab for designing, previewing, processing, and
exporting reusable synthetic and voice-based audio assets.

## Ground Rules

- Use TDD for synthesis primitives, effects, WAV reading/writing, preset and
  recipe resolution, Piper integration seams, and CLI behavior.
- Keep steps small and commits focused.
- Inject ambient state: filesystem paths, process spawning, clocks, and player commands should be passed through seams.
- Keep Piper, playback tools, and encoders optional. Report missing tools
  cleanly instead of making core tests depend on them.
- Prefer deterministic generated samples over checked-in opaque binaries while the sound design is changing.
- Use exported WAV/MP3 assets only as distribution artifacts once presets stabilize.
- Keep Timerbox integration one-way: Timerbox may consume exported sounds, but beepboop should not depend on Timerbox.
- Keep the CLI thin. The durable product is reusable sound pipelines, effects,
  voice recipes, and rendered artifacts.

## Expected Shape

```text
cmd/beepboop
internal/audio
internal/effects
internal/pipeline
internal/presets
internal/player
internal/voice
internal/wav
recipes/
dist/
```

## Stop Rules

Stop and report if tests fail, generated output is nondeterministic without an explicit reason, or a playback/export dependency is missing and cannot be cleanly optional.
