package wav

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

var (
	ErrNotRIFF           = errors.New("wav: not a RIFF/WAVE stream")
	ErrUnsupportedFormat = errors.New("wav: not PCM16 mono")
	ErrNoData            = errors.New("wav: no data chunk")
)

// ReadPCM16Mono decodes a PCM16 mono WAV stream into normalized samples in
// [-1, 1], the same shape audio.Sound uses, so decoded audio can re-enter a
// pipeline as a Source.
//
// It walks the RIFF chunk list rather than assuming a flat 44-byte header:
// Piper and most encoders emit a LIST/INFO chunk before the audio, and a
// fixed-offset reader would decode that metadata as samples. Only PCM16 mono
// is accepted, which matches what WritePCM16Mono produces and what Piper is
// configured to emit; anything else is a clear error rather than a silent
// misread.
func ReadPCM16Mono(r io.Reader) (sampleRate int, samples []float64, err error) {
	var riff struct {
		ID     [4]byte
		Size   uint32
		Format [4]byte
	}
	if err := binary.Read(r, binary.LittleEndian, &riff); err != nil {
		return 0, nil, fmt.Errorf("wav: read RIFF header: %w", err)
	}
	if string(riff.ID[:]) != "RIFF" || string(riff.Format[:]) != "WAVE" {
		return 0, nil, ErrNotRIFF
	}

	haveFormat := false
	for {
		var header struct {
			ID   [4]byte
			Size uint32
		}
		if err := binary.Read(r, binary.LittleEndian, &header); err != nil {
			if errors.Is(err, io.EOF) {
				return 0, nil, ErrNoData
			}
			return 0, nil, fmt.Errorf("wav: read chunk header: %w", err)
		}
		size := int64(header.Size)

		switch string(header.ID[:]) {
		case "fmt ":
			rate, err := readFormatChunk(r, size)
			if err != nil {
				return 0, nil, err
			}
			sampleRate, haveFormat = rate, true
		case "data":
			if !haveFormat {
				// A data chunk before fmt leaves us no way to interpret it.
				return 0, nil, ErrUnsupportedFormat
			}
			samples, err := readDataChunk(r, size)
			if err != nil {
				return 0, nil, err
			}
			return sampleRate, samples, nil
		default:
			// Chunks are word-aligned: an odd size carries a trailing pad
			// byte that the size field does not count.
			if _, err := io.CopyN(io.Discard, r, size+size%2); err != nil {
				return 0, nil, fmt.Errorf("wav: skip %q chunk: %w", header.ID, err)
			}
		}
	}
}

func readFormatChunk(r io.Reader, size int64) (sampleRate int, err error) {
	var format struct {
		AudioFormat   uint16
		Channels      uint16
		SampleRate    uint32
		ByteRate      uint32
		BlockAlign    uint16
		BitsPerSample uint16
	}
	if size < 16 {
		return 0, ErrUnsupportedFormat
	}
	if err := binary.Read(r, binary.LittleEndian, &format); err != nil {
		return 0, fmt.Errorf("wav: read fmt chunk: %w", err)
	}
	// Extensible formats append bytes past the 16 we understand.
	if extra := size - 16; extra > 0 {
		if _, err := io.CopyN(io.Discard, r, extra+extra%2); err != nil {
			return 0, fmt.Errorf("wav: skip fmt extension: %w", err)
		}
	}
	if format.AudioFormat != 1 || format.Channels != 1 || format.BitsPerSample != 16 {
		return 0, fmt.Errorf("%w: format=%d channels=%d bits=%d",
			ErrUnsupportedFormat, format.AudioFormat, format.Channels, format.BitsPerSample)
	}
	return int(format.SampleRate), nil
}

func readDataChunk(r io.Reader, size int64) ([]float64, error) {
	pcm := make([]int16, size/2)
	if err := binary.Read(r, binary.LittleEndian, pcm); err != nil {
		return nil, fmt.Errorf("wav: read data chunk: %w", err)
	}
	samples := make([]float64, len(pcm))
	for i, v := range pcm {
		// Divide by 32767 so full-scale positive maps to exactly 1.0 and
		// round-trips through WritePCM16Mono unchanged. That leaves the most
		// negative code slightly past -1, so clamp it.
		sample := float64(v) / 32767
		if sample < -1 {
			sample = -1
		}
		samples[i] = sample
	}
	return samples, nil
}
