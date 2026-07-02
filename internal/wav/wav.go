package wav

import (
	"encoding/binary"
	"io"
	"math"
)

func WritePCM16Mono(w io.Writer, sampleRate int, samples []float64) error {
	dataSize := uint32(len(samples) * 2)
	byteRate := uint32(sampleRate * 2)

	if _, err := w.Write([]byte("RIFF")); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(36)+dataSize); err != nil {
		return err
	}
	if _, err := w.Write([]byte("WAVEfmt ")); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(16)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint16(1)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint16(1)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(sampleRate)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, byteRate); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint16(2)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint16(16)); err != nil {
		return err
	}
	if _, err := w.Write([]byte("data")); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, dataSize); err != nil {
		return err
	}

	for _, sample := range samples {
		if sample > 1 {
			sample = 1
		}
		if sample < -1 {
			sample = -1
		}
		pcm := int16(math.Round(sample * 32767))
		if sample <= -1 {
			pcm = -32768
		}
		if err := binary.Write(w, binary.LittleEndian, pcm); err != nil {
			return err
		}
	}
	return nil
}
