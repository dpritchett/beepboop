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
- Compose or render background music. That is beatshop's job; see below.

## Kinds of sound

Four different jobs, often confused because they all come out of a speaker.
Picking the wrong kind is the most common way an app's audio ends up annoying,
so it is worth naming them. Which kind a given event should use is the calling
app's decision, not beepboop's -- this is a menu, not a policy.

**Voice lines.** Synthesized speech, roughly a second each. Self-documenting: a
first-time user needs no training to understand "no match". The catch is that
speech does not layer. Two lines at once are mush and you can parse neither, so
any event that can fire twice in a second will talk over itself. Expensive in
bytes, in render time, and in the user's attention. Best for rare, consequential,
or genuinely ambiguous events. Piper renders these.

**Earcons.** Short abstract tones carrying a conventional meaning, roughly 0.1
to 0.5s. The opposite trade: they mean nothing until learned, but they repeat and
overlap cleanly, and they land fast enough to feel like part of the interaction
rather than a report about it. Direction does most of the work -- rising for
arrival or a grant, falling for departure or a loss -- which is why `notify-chime`
descends for control leaving and `turn-ready` rises for it coming back. Best for
frequent, repeated, self-evident events. Built from `AlternatingTone` and the
harmonic shapes.

**Generic SFX.** Texture tied to a physical event: whooshes, thumps, clicks. Not
a message and not a symbol -- the job is sensation, selling motion or impact so a
transition feels like it happened to something. Usually sub-second, one-shot, and
noise-based rather than tonal, which means they want a *moving* filter far more
often than a fixed pitch.

**Beds and background music.** Continuous rather than event-driven, with no
meaning attached to any moment. It has to survive hours without fatiguing, which
is a completely different constraint from the three above. Two sub-kinds worth
separating: state-reactive loops like `flight-slow`/`flight-fast`, which are
seamless texture that responds to what the app is doing, and composed music,
which is written rather than parameterized.

### Which tool

Beepboop bakes voice lines, earcons, generic SFX, and the state-reactive loop
beds. Composed music goes to beatshop, which rents SuperCollider's DSP and is a
far better machine for it.

The one live exception is the `flip` whoosh, which is generic SFX but was routed
to beatshop because a whoosh needs a filter that travels over its duration and
beatshop already has that primitive. If more SFX end up wanting sweeps, the
honest fix is to build sweep support here rather than to keep exporting them.

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
`""` to skip that one.

Labels skip the effects chain, since a spoken name run through a fuzz pedal
cannot do its job. They do get level-matched: each label is normalized to the
peak of the sound it names, because Piper normalizes to full scale and an
unmatched label lands about three times louder than a gentle notification
boop.

Labels need Piper. Point `BEEPBOOP_VOICE_MODEL` at a `.onnx` voice to override
the path in a recipe, which keeps checked-in recipes machine-independent:

```sh
export BEEPBOOP_VOICE_MODEL=~/.local/share/piper/voices/en_US-lessac-medium.onnx
beepboop bake recipes/dist.json dist
beepboop inspect dist/*.wav
```

Piper's VITS models sample noise per run, so beepboop pins `--noise-scale` and
`--noise-w-scale` to zero. Without that the same line renders differently every
time and no artifact is reproducible. Override a recipe's `voice.args` to trade
that back for prosody variation.

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
