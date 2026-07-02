package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunList(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"list"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run(list) code = %d, stderr = %s", code, stderr.String())
	}
	for _, want := range []string{"alarm-basic", "alarm-urgent", "soft-reminder"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, missing %q", stdout.String(), want)
		}
	}
}

func TestRunRender(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "alarm-basic.wav")
	var stdout, stderr bytes.Buffer

	code := run([]string{"render", "alarm-basic", out}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run(render) code = %d, stderr = %s", code, stderr.String())
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read rendered wav: %v", err)
	}
	if string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		t.Fatalf("rendered file is not a WAV")
	}
}

func TestRunUnknownPreset(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"render", "missing", "out.wav"}, &stdout, &stderr)

	if code == 0 {
		t.Fatal("run(render missing) code = 0, want failure")
	}
	if !strings.Contains(stderr.String(), "unknown preset") {
		t.Fatalf("stderr = %q, want unknown preset", stderr.String())
	}
}
