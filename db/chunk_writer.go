package db

import (
	"encoding/binary"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
)

func EncodePoints(w io.Writer, points []Point) error {
	size := len(points)

	if err := binary.Write(w, binary.LittleEndian, size); err != nil {
		return err
	}

	for _, point := range points {
		ts := point.Timestamp
		v := point.Value

		if err := binary.Write(w, binary.LittleEndian, ts); err != nil {
			return err
		}

		if err := binary.Write(w, binary.LittleEndian, math.Float64bits(v)); err != nil {
			return err
		}
	}

	return nil
}

func (s *Series) Flush() error {
	points := s.Chunks[s.activeChunk].Points

	idx := strconv.Itoa(s.activeChunk)

	dir := filepath.Join("data", "chunks", idx)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	startTimestamp := points[0].Timestamp
	timestampStr := strconv.Itoa(int(startTimestamp))

	path := filepath.Join("data", "chunks", idx, timestampStr)

	fp, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer fp.Close()

	if err := EncodePoints(fp, points); err != nil {
		return err
	}

	return fp.Sync()
}
