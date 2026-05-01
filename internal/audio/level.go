package audio

import (
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
	if info.Size() <= 44 {
		return 0, 0, nil
	}

	const window = int64(4096)
	start := info.Size() - window
	if start < 44 {
		start = 44
	}
	if _, err := file.Seek(start, io.SeekStart); err != nil {
		return 0, 0, fmt.Errorf("seek wav file: %w", err)
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return 0, 0, fmt.Errorf("read wav file: %w", err)
	}
	if len(data) < 2 {
		return 0, 0, nil
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
