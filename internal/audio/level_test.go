package audio

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestReadWAVLevelReadsDataChunkBeyond44Bytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.wav")
	if err := writeTestWAV(path, []int16{0, 12000, -24000, 30000}, true); err != nil {
		t.Fatalf("writeTestWAV() error = %v", err)
	}

	level, peak, err := ReadWAVLevel(path)
	if err != nil {
		t.Fatalf("ReadWAVLevel() error = %v", err)
	}
	if level <= 0 {
		t.Fatalf("level = %v, want > 0", level)
	}
	if peak < 90 {
		t.Fatalf("peak = %v, want >= 90", peak)
	}
}

func writeTestWAV(path string, samples []int16, addJunkChunk bool) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	dataSize := uint32(len(samples) * 2)
	junkSize := uint32(0)
	if addJunkChunk {
		junkSize = 6
	}
	riffSize := uint32(4 + (8 + 16) + (8 + junkSize) + (8 + dataSize))

	if _, err := file.Write([]byte("RIFF")); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, riffSize); err != nil {
		return err
	}
	if _, err := file.Write([]byte("WAVE")); err != nil {
		return err
	}

	if _, err := file.Write([]byte("fmt ")); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, uint32(16)); err != nil {
		return err
	}
	for _, value := range []uint16{1, 1} {
		if err := binary.Write(file, binary.LittleEndian, value); err != nil {
			return err
		}
	}
	for _, value := range []uint32{16000, 32000} {
		if err := binary.Write(file, binary.LittleEndian, value); err != nil {
			return err
		}
	}
	for _, value := range []uint16{2, 16} {
		if err := binary.Write(file, binary.LittleEndian, value); err != nil {
			return err
		}
	}

	if addJunkChunk {
		if _, err := file.Write([]byte("JUNK")); err != nil {
			return err
		}
		if err := binary.Write(file, binary.LittleEndian, junkSize); err != nil {
			return err
		}
		if _, err := file.Write([]byte{1, 2, 3, 4, 5, 6}); err != nil {
			return err
		}
	}

	if _, err := file.Write([]byte("data")); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, dataSize); err != nil {
		return err
	}
	for _, sample := range samples {
		if err := binary.Write(file, binary.LittleEndian, sample); err != nil {
			return err
		}
	}

	return nil
}
