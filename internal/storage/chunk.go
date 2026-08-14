package storage

import (
	"encoding/binary"
	"io"
	"math"
)

type Point struct {
	Timestamp int64
	Value     float64
}

type Chunk struct {
	Points []Point
}

func EncodePoints(w io.Writer, points []Point) error {
	size := 4 + len(points)*16
	buf := make([]byte, size)

	binary.LittleEndian.PutUint32(
		buf[0:4],
		uint32(len(points)),
	)

	offset := 4

	for _, point := range points {
		binary.LittleEndian.PutUint64(
			buf[offset:offset+8],
			uint64(point.Timestamp),
		)
		offset += 8

		binary.LittleEndian.PutUint64(
			buf[offset:offset+8],
			math.Float64bits(point.Value),
		)
		offset += 8
	}

	_, err := w.Write(buf)
	return err
}

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
