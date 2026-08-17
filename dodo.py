"""Dev tasks for beepboop. Nothing here is part of the library.

The one job so far is auditioning: getting rendered candidates in front of a
human ear with as little ceremony as possible. That is deliberately not a Go
concern -- `beepboop preview` shells out to `aplay`/`paplay`/`ffplay`, and WSL2's
audio stack is unreliable enough that judging a cue through it is worse than
useless. So playback goes to Windows, the same way beatshop does it, and stays
out of the test suite entirely.

    uv run doit audition                                  # the infobox lab
    uv run doit audition:recipes/flight-lab.json          # any other lab
    uv run doit bake:recipes/navigator.json
"""

from __future__ import annotations

import json
import shutil
import subprocess
from pathlib import Path

DOIT_CONFIG = {"default_tasks": ["audition"]}

REPO = Path(__file__).parent
DEFAULT_RECIPE = "recipes/infobox-lab.json"
# Gap between candidates. Long enough to stop them blurring into one another,
# short enough that a whole lab is still one sitting.
GAP_MS = 700


def _windows_temp() -> Path:
    """Windows %TEMP% as a WSL path.

    Resolved at runtime rather than hardcoded: nothing here should assume a
    Windows username. Playback needs the file on the Windows side, because
    SoundPlayer cannot be trusted with a \\\\wsl$ path.
    """
    out = subprocess.run(
        ["powershell.exe", "-NoProfile", "-Command", "Write-Output $env:TEMP"],
        capture_output=True,
        text=True,
        check=True,
    ).stdout.strip()
    return Path(
        subprocess.run(["wslpath", out], capture_output=True, text=True, check=True)
        .stdout.strip()
    )


def _outdir(recipe: str) -> Path:
    return REPO / "dist" / Path(recipe).stem


def _bake(recipe: str) -> None:
    subprocess.run(
        ["go", "run", "./cmd/beepboop", "bake", recipe, str(_outdir(recipe))],
        cwd=REPO,
        check=True,
    )


def _order(recipe: str) -> list[str]:
    """Play in the order the recipe lists, which is the order it was authored
    in. A lab is usually written reference-first so the comparison lands."""
    spec = json.loads((REPO / recipe).read_text())
    return [o["name"] for o in spec["outputs"]]


def _audition(recipe: str) -> None:
    _bake(recipe)
    src = _outdir(recipe)
    stage = _windows_temp() / f"beepboop-{Path(recipe).stem}"
    stage.mkdir(parents=True, exist_ok=True)

    names = _order(recipe)
    for name in names:
        wav = src / f"{name}.wav"
        if not wav.exists():  # a `say` output with no voice model configured
            print(f"  (skipped {name}: not rendered)")
            continue
        shutil.copy(wav, stage / wav.name)

    print(f"\n{len(names)} candidates, playing in recipe order:\n")
    for name in names:
        if not (stage / f"{name}.wav").exists():
            continue
        print(f"  {name}")
        # PlaySync so the gap is real rather than a race between overlapping
        # async plays.
        subprocess.run(
            [
                "powershell.exe",
                "-NoProfile",
                "-Command",
                f'$p = New-Object System.Media.SoundPlayer "$env:TEMP\\'
                f'beepboop-{Path(recipe).stem}\\{name}.wav"; $p.PlaySync(); '
                f"Start-Sleep -Milliseconds {GAP_MS}",
            ],
            check=False,
        )
    print("\nStaged at:", stage)


def task_audition():
    """Bake a recipe and play every output through Windows, in recipe order."""
    return {
        "actions": [(lambda recipe=DEFAULT_RECIPE: _audition(recipe))],
        "params": [
            {
                "name": "recipe",
                "short": "r",
                "long": "recipe",
                "default": DEFAULT_RECIPE,
                "help": "recipe to bake and play",
            }
        ],
        "uptodate": [False],  # auditioning is never "already done"
        "verbosity": 2,
    }


def task_bake():
    """Bake a recipe to dist/<recipe-stem>/ without playing anything."""
    return {
        "actions": [(lambda recipe=DEFAULT_RECIPE: _bake(recipe))],
        "params": [
            {
                "name": "recipe",
                "short": "r",
                "long": "recipe",
                "default": DEFAULT_RECIPE,
                "help": "recipe to bake",
            }
        ],
        "uptodate": [False],
        "verbosity": 2,
    }
