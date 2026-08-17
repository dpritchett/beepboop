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
beepboop bake recipes/navigator.json dist/navigator   # 23 cues: 19 spoken, 4 preset
```

All 14 `dist/` artifacts verified byte-identical across re-bakes, and
`dist/alarm-basic.wav` still matches the version committed in July.

Two Piper behaviors are load-bearing and easy to undo by accident, both
guarded by tests in `internal/voice`:

- Without `--output_file -` piper plays to the speakers and writes nothing.
- Without `--noise-scale 0 --noise-w-scale 0` the same line renders
  differently every run (observed 30764, 26668, and 30252 bytes for one line).

## Music moved to beatshop (2026-08-17)

**Do not build music here.** `../beatshop` renders it now, by driving
SuperCollider's `scsynth` in non-realtime mode from Python, and Daniel's verdict
is that it does the job vastly better than this engine did. Its `apollo-v1.flac`
has already shipped into callscape and replaced the pair of eight-second loops
that came out of the music lab.

That settles the split recorded in the README: beepboop bakes voice lines,
earcons, SFX, and the state-reactive loop beds; beatshop composes music.

What is dead here as a result:

- **The "Next for music" list.** Drums, filter movement, and longer ambient
  loops were all beatshop's problem the moment it existed. Filter movement in
  particular is already built there -- an LFO through `LinExp.kr` into cascaded
  `LPF.ar`, with the range and period exposed as recipe dials -- so building it
  again here would be duplicating the better version.
- **`recipes/music-lab.json`.** The four candidates it bakes are superseded.
  Left on disk rather than deleted, since it costs nothing and documents how the
  verdict below was reached, but nothing should be built on it. Fair game to
  delete whenever Daniel wants the repo tidier.

What is *not* dead: `audio.Sequence`, the harmonic shapes, recipe `layers`, and
the loop-safe `HighPass` all stay. Multi-note earcons want envelopes and
sequencing exactly as much as music did, and the flight beds still live here.

### Findings worth carrying to beatshop

These came from listening, not from code, so they survive the move and are the
one part of the music work worth keeping:

- **Commit to a groove or commit to ambient, and stay out of the
  one-event-per-second middle.** `music-3` grooved at a 0.5s pulse and worked;
  `music-4` was ambient at two seconds or more and worked; both rejected tracks
  sat at exactly 1.0s spacing, where the ear tracks the repetition but nothing
  drives it.
- **Triangle stacks read as single drones.** The 1/n squared falloff puts nearly
  all the energy in the fundamental, so a triangle stack is barely
  distinguishable from a sine. Saw at 1/n is what makes a chord sound like a
  chord.
- **Stacked voices pile up under 200Hz and the sum reads as mud.** Raising the
  register and high-passing at 180 to 220Hz fixed it. Turning voices down did
  not.
- Length is the cheapest cure for hearing a loop repeat, and a sparse ambient
  bed carries a long one cheaply.

## Callscape's two asks are settled (2026-08-17)

`make sounds` in callscape bakes `recipes/navigator.json` into
`web/public/sounds`. Both asks are now resolved.

### The remote handover: done

`remote-on` and `remote-off` are committed. They fill the two slugs callscape
had wired and silent since July -- the remote taking the wheel and giving it
back. Deliberately presets rather than `say`: the wheel changing hands is a
state change, and a state change wants an earcon, not a sentence read out while
somebody is flying.

The direction carries the meaning. Falling when control leaves you
(`notify-chime`, C6 down to E5); rising when it comes back (`turn-ready`, E5 up
to A5, whose own note here is "it's your turn", which is exactly what getting
the wheel back is).

**The determinism claim is now properly tested.** Baking the whole recipe into
an empty scratch directory and diffing against callscape's committed tree
produced all 23 WAVs byte-identical, with `apollo-v1.flac` correctly untouched.
That is a cold render matching a months-old artifact set, which is a stronger
result than the earlier same-tree rebake.

### The `flip` whoosh: went to beatshop

Callscape's flip -- half a turn to see what is behind you -- wants "something
wooshy that evokes movement and maybe thrusters", one-shot, under about half a
second. Nothing on the app side changes when it lands; it already logs
`voice.missing {cue: "flip"}`.

No preset here fits, and the gap was a primitive rather than a recipe:
`AlternatingTone` is tonal, `Loop` is deliberately envelope-free so it can wrap,
and `lowpass` takes one fixed coefficient. A whoosh is a filter that **moves**.

Beatshop already has that primitive, so Daniel routed the work there. The brief
is `../beatshop/HANDOFF.md`, including how to get noise without breaking its
no-server-RNG rule. Nothing is owed from this repo.

The cost of the routing, recorded honestly: callscape asked for the whoosh to
share the `flight-slow`/`flight-fast` noise family so it reads as those engines,
and that does not survive the move to a different synthesis engine. If it lands
sounding like a stock effect, that is why, and the fix would be building sweep
support here after all.

## Next up

1. **#6 MP3 export** -- an `Exporter` shelling out to an injected encoder,
   mirroring how Piper is wired. The navigator set is 764K of WAV for a
   browser project, so this is the next real need, not a nice-to-have.
2. **Earcons to retire voice lines.** Callscape is moving away from spoken
   lines toward earcons as it becomes a fluid interactive game, starting with
   the high-frequency events (`select`, `capture`, `release`, `fast-on/off`)
   where a one-second line overlaps itself. Which cues change is callscape's
   call, not this repo's; the job here is having earcons ready when asked.
   See "Kinds of sound" in the README for the tradeoff.
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
