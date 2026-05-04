package audio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
)

func ReadWAVLevel(path string) (float64, float64, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil
		}
		return 0, 0, fmt.Errorf("open wav file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return 0, 0, fmt.Errorf("stat wav file: %w", err)
	}
	if info.Size() < 12 {
		return 0, 0, nil
	}

	dataOffset, dataSize, err := readWAVDataChunk(file, info.Size())
	if err != nil {
		return 0, 0, err
	}
	if dataSize < 2 {
		return 0, 0, nil
	}

	const window = int64(4096)
	windowSize := minInt64(window, dataSize)
	if windowSize%2 != 0 {
		windowSize--
	}
	if windowSize < 2 {
		return 0, 0, nil
	}

	start := dataOffset + dataSize - windowSize
	data := make([]byte, windowSize)
	_, err = file.ReadAt(data, start)
	if err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return 0, 0, nil
		}
		return 0, 0, fmt.Errorf("read wav file: %w", err)
	}

	var sumSquares float64
	var peak float64
	var count int
	for i := 0; i+1 < len(data); i += 2 {
		sample := int16(binary.LittleEndian.Uint16(data[i : i+2]))
		normalized := math.Abs(float64(sample) / 32768.0)
		sumSquares += normalized * normalized
		if normalized > peak {
			peak = normalized
		}
		count++
	}
	if count == 0 {
		return 0, 0, nil
	}

	rms := math.Sqrt(sumSquares / float64(count))
	return rms * 100, peak * 100, nil
}

func readWAVDataChunk(file *os.File, size int64) (int64, int64, error) {
	header := make([]byte, 12)
	if _, err := file.ReadAt(header, 0); err != nil {
		return 0, 0, fmt.Errorf("read wav header: %w", err)
	}
	if !bytes.Equal(header[0:4], []byte("RIFF")) || !bytes.Equal(header[8:12], []byte("WAVE")) {
		return 0, 0, nil
	}

	var offset int64 = 12
	for offset+8 <= size {
		chunkHeader := make([]byte, 8)
		if _, err := file.ReadAt(chunkHeader, offset); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return 0, 0, nil
			}
			return 0, 0, fmt.Errorf("read wav chunk header: %w", err)
		}

		chunkSize := int64(binary.LittleEndian.Uint32(chunkHeader[4:8]))
		chunkDataOffset := offset + 8
		if chunkDataOffset > size {
			return 0, 0, nil
		}

		remaining := size - chunkDataOffset
		if chunkSize > remaining {
			chunkSize = remaining
		}

		if bytes.Equal(chunkHeader[0:4], []byte("data")) {
			return chunkDataOffset, chunkSize, nil
		}

		offset = chunkDataOffset + chunkSize
		if chunkSize%2 != 0 {
			offset++
		}
	}

	return 0, 0, nil
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
