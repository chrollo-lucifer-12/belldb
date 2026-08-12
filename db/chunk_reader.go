package db

import (
	"encoding/binary"
	"io"
	"math"
	"os"
	"path/filepath"
)

func DecodePoints(r io.Reader) ([]Point, error) {
	var pointsLen int32

	if err := binary.Read(r, binary.LittleEndian, &pointsLen); err != nil {
		return nil, err
	}

	points := make([]Point, pointsLen)

	for i := 0; i < int(pointsLen); i++ {

		var timestamp int64

		if err := binary.Read(r, binary.LittleEndian, &timestamp); err != nil {
			return nil, err

		}

		var valueBits uint64

		if err := binary.Read(r, binary.LittleEndian, &valueBits); err != nil {
			return nil, err
		}

		points[i] = Point{Timestamp: timestamp, Value: math.Float64frombits(valueBits)}
	}

	return points, nil
}

func LoadChunks(dir string) ([]*Chunk, error) {
	entries, err := os.ReadDir(dir)

	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	chunks := make([]*Chunk, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		path := filepath.Join(dir, entry.Name())

		fp, err := os.Open(path)
		if err != nil {
			return nil, err
		}

		points, err := DecodePoints(fp)
		fp.Close()

		if err != nil {
			return nil, err
		}

		chunks = append(chunks, &Chunk{
			Points: points,
		})
	}

	return chunks, nil
}
