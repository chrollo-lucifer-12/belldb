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
	size := int32(len(points))

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
