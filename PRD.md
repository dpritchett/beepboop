# Beepboop PRD

## Summary

Beepboop is a small local tool for designing synthetic alert sounds in Go and exporting them as reusable audio files.

The first consumer is likely Timerbox, but the project should stand alone as a general-purpose personal sound-design utility.

## User Stories

- As a developer, I can define an alert sound in code.
- As a developer, I can preview a named sound from the terminal.
- As a developer, I can render a named sound to WAV.
- As a developer, I can keep generated assets reproducible from source presets.
- As a developer, I can compare variations without hand-editing binary files.

## MVP Presets

- `alarm-basic`: short alternating two-tone alarm.
- `alarm-urgent`: louder/faster alarm for expired timers.
- `soft-reminder`: gentler short chime for non-critical reminders.

## MVP Acceptance

- `go test ./...` passes.
- `beepboop list` shows available presets.
- `beepboop render alarm-basic dist/alarm-basic.wav` creates a valid WAV.
- `beepboop preview alarm-basic` plays locally when a supported player exists.
