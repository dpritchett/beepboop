package wav

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestWritePCM16Mono(t *testing.T) {
	var buf bytes.Buffer
	samples := []float64{-1, 0, 1}

	if err := WritePCM16Mono(&buf, 8000, samples); err != nil {
		t.Fatalf("WritePCM16Mono() error = %v", err)
	}

	got := buf.Bytes()
	if len(got) != 50 {
		t.Fatalf("len(wav) = %d, want 50", len(got))
	}
	if string(got[0:4]) != "RIFF" || string(got[8:12]) != "WAVE" {
		t.Fatalf("missing RIFF/WAVE header: %q %q", got[0:4], got[8:12])
	}
	if string(got[12:16]) != "fmt " || string(got[36:40]) != "data" {
		t.Fatalf("missing fmt/data chunks")
	}
	if channels := binary.LittleEndian.Uint16(got[22:24]); channels != 1 {
		t.Fatalf("channels = %d, want 1", channels)
	}
	if sampleRate := binary.LittleEndian.Uint32(got[24:28]); sampleRate != 8000 {
		t.Fatalf("sample rate = %d, want 8000", sampleRate)
	}
	if bits := binary.LittleEndian.Uint16(got[34:36]); bits != 16 {
		t.Fatalf("bits = %d, want 16", bits)
	}

	wantPCM := []int16{-32768, 0, 32767}
	for i, want := range wantPCM {
		offset := 44 + i*2
		if got := int16(binary.LittleEndian.Uint16(got[offset : offset+2])); got != want {
			t.Fatalf("pcm[%d] = %d, want %d", i, got, want)
		}
	}
}
