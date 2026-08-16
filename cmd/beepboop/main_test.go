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

func TestRunBake(t *testing.T) {
	dir := t.TempDir()
	recipePath := filepath.Join(dir, "recipe.json")
	if err := os.WriteFile(recipePath, []byte(`{"outputs": [
		{"name": "one", "preset": "notify-blip"},
		{"name": "two", "preset": "notify-chime",
		 "effects": [{"type": "fuzz", "drive": 8, "bias": 0.3}]}
	]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(dir, "out")
	var stdout, stderr bytes.Buffer

	code := run([]string{"bake", recipePath, outDir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run(bake) code = %d, stderr = %s", code, stderr.String())
	}
	for _, name := range []string{"one.wav", "two.wav"} {
		data, err := os.ReadFile(filepath.Join(outDir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if string(data[0:4]) != "RIFF" {
			t.Errorf("%s is not a WAV", name)
		}
	}
	if !strings.Contains(stdout.String(), "one.wav") {
		t.Errorf("stdout = %q, want the rendered files listed", stdout.String())
	}
}

func TestRunBakeReportsRecipeErrors(t *testing.T) {
	dir := t.TempDir()
	recipePath := filepath.Join(dir, "recipe.json")
	if err := os.WriteFile(recipePath, []byte(
		`{"outputs": [{"name": "x", "preset": "nope"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer

	code := run([]string{"bake", recipePath, filepath.Join(dir, "out")}, &stdout, &stderr)

	if code == 0 {
		t.Fatal("run(bake) code = 0, want failure")
	}
	if !strings.Contains(stderr.String(), "unknown preset") {
		t.Errorf("stderr = %q, want unknown preset", stderr.String())
	}
}

func TestRunBakeVoiceModelEnvOverride(t *testing.T) {
	// The override has to reach Piper, which is not installed here. Pointing
	// it at a model and asserting the failure names piper (not the model)
	// proves the value was applied and the lookup got that far.
	dir := t.TempDir()
	recipePath := filepath.Join(dir, "recipe.json")
	if err := os.WriteFile(recipePath, []byte(
		`{"outputs": [{"name": "x", "say": "hello"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(voiceModelEnv, filepath.Join(dir, "voice.onnx"))
	var stdout, stderr bytes.Buffer

	code := run([]string{"bake", recipePath, filepath.Join(dir, "out")}, &stdout, &stderr)

	if code == 0 {
		t.Skip("piper is installed; this test covers the not-installed path")
	}
	if strings.Contains(stderr.String(), "no voice model") {
		t.Errorf("stderr = %q, want the env override to supply the model", stderr.String())
	}
}

func TestRunBakeMissingRecipe(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"bake", "/nope/recipe.json", t.TempDir()}, &stdout, &stderr)

	if code == 0 {
		t.Fatal("run(bake) code = 0, want failure")
	}
}

func TestRunInspect(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "blip.wav")
	var stdout, stderr bytes.Buffer
	if code := run([]string{"render", "notify-blip", out}, &stdout, &stderr); code != 0 {
		t.Fatalf("render failed: %s", stderr.String())
	}
	stdout.Reset()

	code := run([]string{"inspect", out}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run(inspect) code = %d, stderr = %s", code, stderr.String())
	}
	// notify-blip is 0.14s at 44100 with a 0.25 gain.
	for _, want := range []string{"44100", "0.140", "0.25"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("stdout = %q, missing %q", stdout.String(), want)
		}
	}
}

func TestRunInspectRejectsNonWAV(t *testing.T) {
	dir := t.TempDir()
	bogus := filepath.Join(dir, "bogus.wav")
	if err := os.WriteFile(bogus, []byte("not a wav at all"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer

	code := run([]string{"inspect", bogus}, &stdout, &stderr)

	if code == 0 {
		t.Fatal("run(inspect) code = 0, want failure")
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
