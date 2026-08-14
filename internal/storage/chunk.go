package storage

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
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

	if pointsLen < 0 {
		return nil, fmt.Errorf("invalid points length: %d", pointsLen)
	}

	points := make([]Point, pointsLen)
	var buf [16]byte

	for i := range points {
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, err
		}

		points[i] = Point{
			Timestamp: int64(binary.LittleEndian.Uint64(buf[0:8])),
			Value: math.Float64frombits(
				binary.LittleEndian.Uint64(buf[8:16]),
			),
		}
	}

	return points, nil
}

func FindPoint(path string, target int64, count int) (Point, error) {

	fp, err := os.Open(path)
	if err != nil {
		return Point{}, err
	}
	defer fp.Close()

	var buf [16]byte

	low := 0
	high := count - 1

	for high >= low {
		mid := (low + high) / 2

		offset := int64(4 + mid*16)

		if _, err := fp.Seek(offset, io.SeekStart); err != nil {
			return Point{}, err
		}

		if _, err := io.ReadFull(fp, buf[:]); err != nil {
			return Point{}, err
		}

		timestamp := int64(binary.LittleEndian.Uint64(buf[0:8]))

		if timestamp < target {
			low = mid + 1
			continue
		}

		if timestamp > target {
			high = mid - 1
			continue
		}

		return Point{
			Timestamp: timestamp,
			Value: math.Float64frombits(
				binary.LittleEndian.Uint64(buf[8:16]),
			),
		}, nil
	}

	return Point{}, fmt.Errorf("timestamp not found: %d", target)
}

func ReadPointAt(path string, index int) (Point, error) {
	fp, err := os.Open(path)
	if err != nil {
		return Point{}, err
	}
	defer fp.Close()

	offset := int64(4 + index*16)

	if _, err := fp.Seek(offset, io.SeekStart); err != nil {
		return Point{}, err
	}

	var buf [16]byte

	if _, err := io.ReadFull(fp, buf[:]); err != nil {
		return Point{}, err
	}

	return Point{
		Timestamp: int64(binary.LittleEndian.Uint64(buf[0:8])),
		Value: math.Float64frombits(
			binary.LittleEndian.Uint64(buf[8:16]),
		),
	}, nil
}
