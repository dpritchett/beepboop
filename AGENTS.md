# beepboop

Go-based noise generation studio for designing, previewing, and exporting small synthetic alert sounds.

## Ground Rules

- Use TDD for synthesis primitives, WAV writing, preset resolution, and CLI behavior.
- Keep steps small and commits focused.
- Inject ambient state: filesystem paths, process spawning, clocks, and player commands should be passed through seams.
- Prefer deterministic generated samples over checked-in opaque binaries while the sound design is changing.
- Use exported WAV/MP3 assets only as distribution artifacts once presets stabilize.
- Keep Timerbox integration one-way: Timerbox may consume exported sounds, but beepboop should not depend on Timerbox.

## Expected Shape

```text
cmd/beepboop
internal/audio
internal/presets
internal/player
internal/wav
dist/
```

## Stop Rules

Stop and report if tests fail, generated output is nondeterministic without an explicit reason, or a playback/export dependency is missing and cannot be cleanly optional.
