package main

import (
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"

	"beepboop/internal/pipeline"
	"beepboop/internal/player"
	"beepboop/internal/presets"
	"beepboop/internal/recipe"
	"beepboop/internal/voice"
	"beepboop/internal/wav"
)

// voiceModelEnv overrides a recipe's Piper voice model path.
const voiceModelEnv = "BEEPBOOP_VOICE_MODEL"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return 2
	}

	switch args[0] {
	case "list":
		for _, name := range presets.List() {
			fmt.Fprintln(stdout, name)
		}
		return 0
	case "render":
		if len(args) != 3 {
			fmt.Fprintln(stderr, "usage: beepboop render <preset> <output.wav>")
			return 2
		}
		if err := render(args[1], args[2]); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	case "bake":
		if len(args) != 3 {
			fmt.Fprintln(stderr, "usage: beepboop bake <recipe.json> <outdir>")
			return 2
		}
		if err := bake(args[1], args[2], stdout); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	case "inspect":
		if len(args) < 2 {
			fmt.Fprintln(stderr, "usage: beepboop inspect <file.wav>...")
			return 2
		}
		if err := inspect(args[1:], stdout); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	case "preview":
		if len(args) != 2 {
			fmt.Fprintln(stderr, "usage: beepboop preview <preset>")
			return 2
		}
		if err := preview(args[1]); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	default:
		usage(stderr)
		return 2
	}
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "usage: beepboop <list|render|bake|inspect|preview>")
	fmt.Fprintln(w, "  list                            list available presets")
	fmt.Fprintln(w, "  render <preset> <output.wav>    render one preset")
	fmt.Fprintln(w, "  bake <recipe.json> <outdir>     batch-render a recipe")
	fmt.Fprintln(w, "  inspect <file.wav>...           report rate, length, peak")
	fmt.Fprintln(w, "  preview <preset>                render and play locally")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "  BEEPBOOP_VOICE_MODEL overrides a recipe's Piper voice model path.")
}

// inspect reports rate, length, and peak for rendered WAVs, so verifying an
// artifact is a repo capability rather than a throwaway script each time.
func inspect(paths []string, stdout io.Writer) error {
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		rate, samples, err := wav.ReadPCM16Mono(file)
		file.Close()
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		peak := 0.0
		for _, sample := range samples {
			peak = math.Max(peak, math.Abs(sample))
		}
		fmt.Fprintf(stdout, "%s\trate=%d\tsamples=%d\tdur=%.3fs\tpeak=%.2f\n",
			path, rate, len(samples), float64(len(samples))/float64(rate), peak)
	}
	return nil
}

// bake renders every output in a recipe into outdir as <name>.wav. Spoken
// labels land beside their sound as <name>.label.wav.
func bake(recipePath, outDir string, stdout io.Writer) error {
	file, err := os.Open(recipePath)
	if err != nil {
		return err
	}
	defer file.Close()

	spec, err := recipe.Parse(file)
	if err != nil {
		return err
	}
	// Voice models are multi-megabyte files living wherever the operator put
	// them, so a checked-in recipe cannot name a portable path. The env var
	// keeps the recipe machine-independent.
	if model := os.Getenv(voiceModelEnv); model != "" {
		spec.Voice.Model = model
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	results, err := spec.Render(recipe.Options{
		Open: func(name string) (io.WriteCloser, error) {
			return os.Create(filepath.Join(outDir, name+".wav"))
		},
		Run:      voice.SystemRunner,
		LookPath: voice.SystemLookPath,
	})
	if err != nil {
		return err
	}
	for _, result := range results {
		fmt.Fprintf(stdout, "%s.wav\n", filepath.Join(outDir, result.Name))
		if result.Label != "" {
			fmt.Fprintf(stdout, "%s.wav\n", filepath.Join(outDir, result.Label))
		}
	}
	return nil
}

func render(name, output string) error {
	preset, ok := presets.Resolve(name)
	if !ok {
		return fmt.Errorf("unknown preset %q", name)
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return err
	}
	file, err := os.Create(output)
	if err != nil {
		return err
	}
	defer file.Close()
	return pipeline.Pipeline{
		Source:   pipeline.StaticSource{Sound: preset.Sound},
		Exporter: pipeline.WAVExporter{W: file},
	}.Run()
}

func preview(name string) error {
	tmp, err := os.CreateTemp("", "beepboop-*.wav")
	if err != nil {
		return err
	}
	path := tmp.Name()
	tmp.Close()
	defer os.Remove(path)

	if err := render(name, path); err != nil {
		return err
	}
	cmdSpec, ok := player.FirstAvailable(player.SystemLookPath)
	if !ok {
		return fmt.Errorf("no supported audio player found; tried aplay, paplay, ffplay")
	}
	args := append([]string{}, cmdSpec.Args...)
	args = append(args, path)
	cmd := exec.Command(cmdSpec.Name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
