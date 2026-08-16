package wav

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"testing"
)

// buildWAV assembles a PCM16 mono file with optional extra chunks inserted
// between fmt and data, so tests can cover readers that must walk chunks.
func buildWAV(t *testing.T, sampleRate int, pcm []int16, extraChunks ...[]byte) []byte {
	t.Helper()
	var body bytes.Buffer

	fmtChunk := new(bytes.Buffer)
	binary.Write(fmtChunk, binary.LittleEndian, uint16(1))                // PCM
	binary.Write(fmtChunk, binary.LittleEndian, uint16(1))                // mono
	binary.Write(fmtChunk, binary.LittleEndian, uint32(sampleRate))       //
	binary.Write(fmtChunk, binary.LittleEndian, uint32(sampleRate*2))     // byte rate
	binary.Write(fmtChunk, binary.LittleEndian, uint16(2))                // block align
	binary.Write(fmtChunk, binary.LittleEndian, uint16(16))               // bits
	body.WriteString("fmt ")
	binary.Write(&body, binary.LittleEndian, uint32(fmtChunk.Len()))
	body.Write(fmtChunk.Bytes())

	for _, chunk := range extraChunks {
		body.Write(chunk)
	}

	data := new(bytes.Buffer)
	for _, s := range pcm {
		binary.Write(data, binary.LittleEndian, s)
	}
	body.WriteString("data")
	binary.Write(&body, binary.LittleEndian, uint32(data.Len()))
	body.Write(data.Bytes())

	var out bytes.Buffer
	out.WriteString("RIFF")
	binary.Write(&out, binary.LittleEndian, uint32(4+body.Len()))
	out.WriteString("WAVE")
	out.Write(body.Bytes())
	return out.Bytes()
}

func TestReadPCM16Mono(t *testing.T) {
	in := buildWAV(t, 8000, []int16{-32768, 0, 32767})

	rate, samples, err := ReadPCM16Mono(bytes.NewReader(in))
	if err != nil {
		t.Fatalf("ReadPCM16Mono() error = %v", err)
	}
	if rate != 8000 {
		t.Errorf("sample rate = %d, want 8000", rate)
	}
	want := []float64{-1, 0, 1}
	if len(samples) != len(want) {
		t.Fatalf("sample count = %d, want %d", len(samples), len(want))
	}
	for i := range want {
		if math.Abs(samples[i]-want[i]) > 1e-4 {
			t.Errorf("sample[%d] = %v, want %v", i, samples[i], want[i])
		}
	}
}

func TestReadPCM16MonoSkipsUnknownChunks(t *testing.T) {
	// Piper and most encoders emit a LIST/INFO chunk before data. A reader
	// that assumes a flat 44-byte header would read metadata as audio.
	list := new(bytes.Buffer)
	list.WriteString("LIST")
	binary.Write(list, binary.LittleEndian, uint32(10))
	list.WriteString("INFOhello!")

	in := buildWAV(t, 22050, []int16{100, -100}, list.Bytes())

	rate, samples, err := ReadPCM16Mono(bytes.NewReader(in))
	if err != nil {
		t.Fatalf("ReadPCM16Mono() error = %v", err)
	}
	if rate != 22050 {
		t.Errorf("sample rate = %d, want 22050", rate)
	}
	if len(samples) != 2 {
		t.Fatalf("sample count = %d, want 2", len(samples))
	}
}

func TestReadPCM16MonoOddSizedChunkIsPadded(t *testing.T) {
	// RIFF chunks are word-aligned: an odd-length chunk carries a pad byte
	// that is not counted in its size field.
	odd := new(bytes.Buffer)
	odd.WriteString("junk")
	binary.Write(odd, binary.LittleEndian, uint32(3))
	odd.WriteString("abc")
	odd.WriteByte(0)

	in := buildWAV(t, 8000, []int16{500, -500, 250}, odd.Bytes())

	_, samples, err := ReadPCM16Mono(bytes.NewReader(in))
	if err != nil {
		t.Fatalf("ReadPCM16Mono() error = %v", err)
	}
	if len(samples) != 3 {
		t.Fatalf("sample count = %d, want 3", len(samples))
	}
}

func TestReadPCM16MonoRoundTrip(t *testing.T) {
	original := []float64{0, 0.25, -0.5, 0.75, -1}

	var buf bytes.Buffer
	if err := WritePCM16Mono(&buf, 44100, original); err != nil {
		t.Fatalf("WritePCM16Mono() error = %v", err)
	}
	rate, got, err := ReadPCM16Mono(&buf)
	if err != nil {
		t.Fatalf("ReadPCM16Mono() error = %v", err)
	}
	if rate != 44100 {
		t.Errorf("sample rate = %d, want 44100", rate)
	}
	if len(got) != len(original) {
		t.Fatalf("sample count = %d, want %d", len(got), len(original))
	}
	// PCM16 quantization is lossy; one LSB of tolerance is the real contract.
	for i := range original {
		if math.Abs(got[i]-original[i]) > 1.0/32767.0 {
			t.Errorf("sample[%d] = %v, want %v", i, got[i], original[i])
		}
	}
}

func TestReadPCM16MonoRejectsNonRIFF(t *testing.T) {
	_, _, err := ReadPCM16Mono(bytes.NewReader([]byte("NOPEnot a wav file at all")))
	if !errors.Is(err, ErrNotRIFF) {
		t.Errorf("err = %v, want ErrNotRIFF", err)
	}
}

func TestReadPCM16MonoRejectsStereo(t *testing.T) {
	in := buildWAV(t, 8000, []int16{1, 2})
	// Patch the channel count in the fmt chunk to 2.
	binary.LittleEndian.PutUint16(in[22:24], 2)

	_, _, err := ReadPCM16Mono(bytes.NewReader(in))
	if !errors.Is(err, ErrUnsupportedFormat) {
		t.Errorf("err = %v, want ErrUnsupportedFormat", err)
	}
}

func TestReadPCM16MonoRejectsNon16Bit(t *testing.T) {
	in := buildWAV(t, 8000, []int16{1, 2})
	binary.LittleEndian.PutUint16(in[34:36], 24)

	_, _, err := ReadPCM16Mono(bytes.NewReader(in))
	if !errors.Is(err, ErrUnsupportedFormat) {
		t.Errorf("err = %v, want ErrUnsupportedFormat", err)
	}
}

func TestReadPCM16MonoRejectsMissingData(t *testing.T) {
	in := buildWAV(t, 8000, []int16{1, 2})
	// Rename the data chunk so no audio is ever found.
	copy(in[36:40], "junk")

	_, _, err := ReadPCM16Mono(bytes.NewReader(in))
	if !errors.Is(err, ErrNoData) {
		t.Errorf("err = %v, want ErrNoData", err)
	}
}

func TestReadPCM16MonoRejectsTruncated(t *testing.T) {
	in := buildWAV(t, 8000, []int16{1, 2, 3})

	_, _, err := ReadPCM16Mono(bytes.NewReader(in[:len(in)-3]))
	if err == nil {
		t.Error("err = nil, want a truncation error")
	}
}
