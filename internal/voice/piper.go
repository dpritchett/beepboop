// Package voice provides optional text-to-speech sources. Piper is the first
// adapter.
//
// Piper is an external binary and a large voice model, neither of which the
// core is allowed to depend on. So the process runner and the binary lookup
// are both injected: tests exercise the adapter with a fake runner, and a
// machine without Piper installed gets a clear error instead of a broken
// build or a hanging test.
package voice

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"beepboop/internal/audio"
	"beepboop/internal/wav"
)

var (
	ErrPiperNotFound = errors.New("voice: piper not found in PATH")
	ErrNoModel       = errors.New("voice: no piper voice model configured")
	ErrNoText        = errors.New("voice: no text to speak")
)

// Command is an external program invocation, mirroring player.Command.
type Command struct {
	Name string
	Args []string
}

// Runner executes a command, feeding it stdin and collecting stdout. Injecting
// this is what keeps the package testable without spawning processes.
type Runner func(cmd Command, stdin io.Reader, stdout io.Writer) error

// LookPath reports whether a binary is available, matching exec.LookPath.
type LookPath func(string) (string, error)

// Piper renders text to speech with a local Piper install, implementing
// pipeline.Source so spoken audio can feed straight into an effects chain.
type Piper struct {
	// Binary is the Piper executable; defaults to "piper".
	Binary string
	// Model is the path to a .onnx voice. Required.
	Model string
	// Text is the line to speak. Required.
	Text string
	// Args overrides the full argument list. Piper's flag names have shifted
	// across releases, so callers pin their own rather than waiting on us.
	Args []string

	Run      Runner
	LookPath LookPath
}

// Available reports whether this Piper is configured and its binary is on
// PATH, without synthesizing anything. Batch callers use it to fail before
// writing the first file instead of part way through a directory.
func (p Piper) Available() error {
	if p.Model == "" {
		return ErrNoModel
	}
	if p.Run == nil {
		return errors.New("voice: no command runner configured")
	}
	if p.LookPath == nil {
		return nil
	}
	if _, err := p.LookPath(p.binary()); err != nil {
		return fmt.Errorf("%w: install piper or set Binary (%q): %v",
			ErrPiperNotFound, p.binary(), err)
	}
	return nil
}

// defaultArgs builds the argv for a deterministic render to stdout.
//
// The noise scales matter more than they look. Piper's VITS models sample
// noise per run, so the same line synthesized three times produced 30764,
// 26668, and 30252 bytes of audio here. Pinning both to zero makes the output
// byte-identical run to run, which is the whole contract this project sells:
// artifacts reproducible from source. It costs some prosody variation, which
// for a spoken file label is no loss at all. Override Args to get it back.
//
// "-f -" is required, not decorative. With no output flag piper plays the
// audio through the speakers instead of writing anything to stdout.
func defaultArgs(model string) []string {
	return []string{
		"--model", model,
		"--noise-scale", "0",
		"--noise-w-scale", "0",
		"--output_file", "-",
	}
}

func (p Piper) binary() string {
	if p.Binary == "" {
		return "piper"
	}
	return p.Binary
}

// Render speaks Text and decodes the resulting WAV.
//
// Piper writes a WAV stream to stdout, which we buffer and hand to the same
// reader used for artifacts on disk. Buffering costs memory proportional to
// the line length, which for notification-sized lines is nothing, and it
// keeps the decode path identical to every other source.
func (p Piper) Render() (audio.Sound, error) {
	if err := p.Available(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(p.Text) == "" {
		return nil, ErrNoText
	}

	args := p.Args
	if args == nil {
		args = defaultArgs(p.Model)
	}

	// Piper reads one line of text per utterance from stdin. Keeping the text
	// out of argv avoids flag injection from arbitrary lines and argv limits.
	stdin := strings.NewReader(strings.TrimSpace(p.Text) + "\n")
	var stdout bytes.Buffer
	if err := p.Run(Command{Name: p.binary(), Args: args}, stdin, &stdout); err != nil {
		return nil, fmt.Errorf("voice: piper failed: %w", err)
	}

	rate, samples, err := wav.ReadPCM16Mono(&stdout)
	if err != nil {
		return nil, fmt.Errorf("voice: decode piper output: %w", err)
	}
	return audio.NewSound(rate, samples), nil
}

// SystemRunner runs a command with the real os/exec. Stderr is captured and
// folded into the error so a failing Piper explains itself.
func SystemRunner(cmd Command, stdin io.Reader, stdout io.Writer) error {
	var stderr bytes.Buffer
	c := exec.Command(cmd.Name, cmd.Args...)
	c.Stdin = stdin
	c.Stdout = stdout
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return fmt.Errorf("%w: %s", err, msg)
		}
		return err
	}
	return nil
}

// SystemLookPath resolves a binary against the real PATH.
func SystemLookPath(name string) (string, error) {
	return exec.LookPath(name)
}
