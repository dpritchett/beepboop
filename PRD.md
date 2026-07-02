# Beepboop PRD

## Summary

Beepboop is a local sound pipeline workspace. It lets agents and developers
define audio sources, effects chains, voice recipes, and batch exports in
source, then render deterministic audio assets.

The first artifact is a synthetic alert WAV. The broader direction is reusable
pipeline work: synthetic sounds, local Piper-generated voices, aggressive
distortion chains, and repeatable export batches.

## User Stories

- As a developer, I can define a sound pipeline in source.
- As a developer, I can render a named preset or recipe to WAV.
- As a developer, I can preview rendered output when a local player exists.
- As a developer, I can keep generated assets reproducible from source.
- As a developer, I can batch-render multiple spoken lines as separate files.
- As a developer, I can use Piper as an optional local TTS source.
- As a developer, I can run voice output through reusable distortion and
  rock-and-roll effects chains.

## Completed First Artifact

- `alarm-basic`: short alternating two-tone alarm.
- `dist/alarm-basic.wav`: generated from source with the CLI.

## Near-Term Roadmap

1. Add reusable effects primitives: gain, hard clip, soft clip, fuzz, and
   normalization.
2. Add WAV reading so generated TTS output can re-enter the pipeline.
3. Add a pipeline abstraction for `source -> effects -> export`.
4. Add a Piper voice adapter behind an injected command runner.
5. Add recipe files for batch-rendering multiple lines.
6. Add optional MP3 export through an injected external encoder.

## MVP Acceptance

- `go test ./...` passes.
- `beepboop list` shows available presets.
- `beepboop render alarm-basic dist/alarm-basic.wav` creates a valid WAV.
- `beepboop preview alarm-basic` plays locally when a supported player exists.
- Missing optional tools fail with clear messages instead of breaking core
  tests.
