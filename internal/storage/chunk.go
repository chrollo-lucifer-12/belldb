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

type DODChunk struct {
	timestamps []byte
	values     []byte

	count int

	firstTimestamp int64
	lastTimestamp  int64
	lastDelta      int64
}

func DecompressChunk(dodChunk *DODChunk) (Chunk, error) {

	chunk := Chunk{
		Points: make([]Point, 0, dodChunk.count),
	}

	if dodChunk.count == 0 {
		return chunk, nil
	}

	if len(dodChunk.timestamps) < 8 {
		return Chunk{}, io.ErrUnexpectedEOF
	}

	offset := 0

	firstTimestamp := int64(
		binary.LittleEndian.Uint64(
			dodChunk.timestamps[offset : offset+8],
		),
	)
	offset += 8

	if len(dodChunk.values) < 8 {
		return Chunk{}, io.ErrUnexpectedEOF
	}

	value := math.Float64frombits(
		binary.LittleEndian.Uint64(
			dodChunk.values[:8],
		),
	)

	chunk.Points = append(chunk.Points, Point{
		Timestamp: firstTimestamp,
		Value:     value,
	})

	if dodChunk.count == 1 {
		return chunk, nil
	}

	if len(dodChunk.timestamps) < offset+8 {
		return Chunk{}, io.ErrUnexpectedEOF
	}

	lastDelta := int64(
		binary.LittleEndian.Uint64(
			dodChunk.timestamps[offset : offset+8],
		),
	)
	offset += 8

	lastTimestamp := firstTimestamp + lastDelta

	if len(dodChunk.values) < 16 {
		return Chunk{}, io.ErrUnexpectedEOF
	}

	value = math.Float64frombits(
		binary.LittleEndian.Uint64(
			dodChunk.values[8:16],
		),
	)

	chunk.Points = append(chunk.Points, Point{
		Timestamp: lastTimestamp,
		Value:     value,
	})

	valueOffset := 16

	for i := 2; i < dodChunk.count; i++ {
		dod, n := binary.Varint(dodChunk.timestamps[offset:])
		if n == 0 {
			return Chunk{}, io.ErrUnexpectedEOF
		}
		if n < 0 {
			return Chunk{}, fmt.Errorf("invalid delta-of-delta encoding")
		}

		offset += n

		delta := lastDelta + dod
		timestamp := lastTimestamp + delta

		if len(dodChunk.values) < valueOffset+8 {
			return Chunk{}, io.ErrUnexpectedEOF
		}

		value = math.Float64frombits(
			binary.LittleEndian.Uint64(
				dodChunk.values[valueOffset : valueOffset+8],
			),
		)

		chunk.Points = append(chunk.Points, Point{
			Timestamp: timestamp,
			Value:     value,
		})

		lastTimestamp = timestamp
		lastDelta = delta
		valueOffset += 8
	}

	return chunk, nil
}

func CompressChunk(chunk *Chunk) (*DODChunk, error) {
	if len(chunk.Points) == 0 {
		return nil, nil
	}

	dod := &DODChunk{
		count:          len(chunk.Points),
		firstTimestamp: chunk.Points[0].Timestamp,
		lastTimestamp:  chunk.Points[len(chunk.Points)-1].Timestamp,
	}

	var buf [8]byte
	binary.LittleEndian.PutUint64(
		buf[:],
		uint64(chunk.Points[0].Timestamp),
	)

	dod.timestamps = append(dod.timestamps, buf[:]...)

	if len(chunk.Points) == 1 {
		dod.values = encodeValues(chunk.Points)
		return dod, nil
	}

	lastTimestamp := chunk.Points[0].Timestamp
	lastDelta := chunk.Points[1].Timestamp - lastTimestamp

	binary.LittleEndian.PutUint64(
		buf[:],
		uint64(lastDelta),
	)

	dod.timestamps = append(dod.timestamps, buf[:]...)

	lastTimestamp = chunk.Points[1].Timestamp

	for i := 2; i < len(chunk.Points); i++ {
		ts := chunk.Points[i].Timestamp

		delta := ts - lastTimestamp
		dodValue := delta - lastDelta

		dod.timestamps = binary.AppendVarint(
			dod.timestamps,
			dodValue,
		)

		lastTimestamp = ts
		lastDelta = delta
	}

	dod.lastTimestamp = lastTimestamp
	dod.lastDelta = lastDelta

	dod.values = encodeValues(chunk.Points)

	return dod, nil
}

func encodeValues(points []Point) []byte {
	buf := make([]byte, 0, len(points)*8)

	var tmp [8]byte

	for _, p := range points {
		binary.LittleEndian.PutUint64(tmp[:], math.Float64bits(p.Value))

		buf = append(buf, tmp[:]...)
	}

	return buf
}

func EncodeDODChunk(w io.Writer, chunk DODChunk) error {
	if err := binary.Write(w, binary.LittleEndian, uint32(chunk.count)); err != nil {
		return err
	}

	if err := binary.Write(w, binary.LittleEndian, chunk.firstTimestamp); err != nil {
		return err
	}

	if err := binary.Write(w, binary.LittleEndian, uint32(len(chunk.timestamps))); err != nil {
		return err
	}

	if _, err := w.Write(chunk.timestamps); err != nil {
		return err
	}

	if err := binary.Write(w, binary.LittleEndian, uint32(len(chunk.values))); err != nil {
		return err
	}

	if _, err := w.Write(chunk.values); err != nil {
		return err
	}

	return nil
}

func DecodeDODChunk(r io.Reader) (*DODChunk, error) {
	var (
		pointsCount    int32
		firstTimestamp int64
		timestampsLen  int32
		valuesLen      int32
	)

	if err := binary.Read(r, binary.LittleEndian, &pointsCount); err != nil {
		return nil, fmt.Errorf("read count: %w", err)
	}

	if pointsCount < 0 {
		return nil, fmt.Errorf("invalid points count")
	}

	if err := binary.Read(r, binary.LittleEndian, &firstTimestamp); err != nil {
		return nil, fmt.Errorf("read first timestamp: %w", err)
	}

	if err := binary.Read(r, binary.LittleEndian, &timestampsLen); err != nil {
		return nil, fmt.Errorf("read timestamps length: %w", err)
	}

	if timestampsLen < 0 {
		return nil, fmt.Errorf("invalid timestamps len")
	}

	timestamps := make([]byte, timestampsLen)

	if _, err := io.ReadFull(r, timestamps); err != nil {
		return nil, fmt.Errorf("read timestamps: %w", err)
	}

	if err := binary.Read(r, binary.LittleEndian, &valuesLen); err != nil {
		return nil, fmt.Errorf("read values length: %w", err)
	}

	values := make([]byte, valuesLen)

	if _, err := io.ReadFull(r, values); err != nil {
		return nil, fmt.Errorf("read values: %w", err)
	}

	return &DODChunk{
		timestamps:     timestamps,
		values:         values,
		count:          int(pointsCount),
		firstTimestamp: firstTimestamp,
	}, nil
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
