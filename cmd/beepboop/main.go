package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"beepboop/internal/player"
	"beepboop/internal/presets"
	"beepboop/internal/wav"
)

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
	fmt.Fprintln(w, "usage: beepboop <list|render|preview>")
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
	return wav.WritePCM16Mono(file, preset.Sound.SampleRate(), preset.Sound.Samples())
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
