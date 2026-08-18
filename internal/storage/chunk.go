package storage

import (
	"encoding/binary"
	"fmt"
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

type DODChunk struct {
	timestamps []byte
	values     []byte

	count int

	firstTimestamp int64
	lastTimestamp  int64
}

func DecompressChunk(dodChunk *DODChunk) (Chunk, error) {
	chunk := Chunk{
		Points: make([]Point, dodChunk.count),
	}

	if dodChunk.count == 0 {
		return chunk, nil
	}

	if len(dodChunk.timestamps) < 8 || len(dodChunk.values) < 8 {
		return Chunk{}, io.ErrUnexpectedEOF
	}

	timestampOffset := 0
	valueOffset := 0

	firstTimestamp := int64(
		binary.LittleEndian.Uint64(
			dodChunk.timestamps[timestampOffset : timestampOffset+8],
		),
	)
	timestampOffset += 8

	lastValue := binary.LittleEndian.Uint64(
		dodChunk.values[valueOffset : valueOffset+8],
	)
	valueOffset += 8

	chunk.Points[0] = Point{
		Timestamp: firstTimestamp,
		Value:     math.Float64frombits(lastValue),
	}

	if dodChunk.count == 1 {
		return chunk, nil
	}

	if len(dodChunk.timestamps) < timestampOffset+8 ||
		len(dodChunk.values) < valueOffset+8 {
		return Chunk{}, io.ErrUnexpectedEOF
	}

	lastDelta := int64(
		binary.LittleEndian.Uint64(
			dodChunk.timestamps[timestampOffset : timestampOffset+8],
		),
	)
	timestampOffset += 8

	lastTimestamp := firstTimestamp + lastDelta

	xor := binary.LittleEndian.Uint64(
		dodChunk.values[valueOffset : valueOffset+8],
	)
	valueOffset += 8

	lastValue ^= xor

	chunk.Points[1] = Point{
		Timestamp: lastTimestamp,
		Value:     math.Float64frombits(lastValue),
	}

	for i := 2; i < dodChunk.count; i++ {
		dod, n := binary.Varint(
			dodChunk.timestamps[timestampOffset:],
		)

		if n == 0 {
			return Chunk{}, io.ErrUnexpectedEOF
		}

		if n < 0 {
			return Chunk{}, fmt.Errorf("invalid delta-of-delta encoding")
		}

		timestampOffset += n

		delta := lastDelta + dod
		timestamp := lastTimestamp + delta

		if len(dodChunk.values) < valueOffset+8 {
			return Chunk{}, io.ErrUnexpectedEOF
		}

		xor := binary.LittleEndian.Uint64(
			dodChunk.values[valueOffset : valueOffset+8],
		)
		valueOffset += 8

		lastValue ^= xor

		chunk.Points[i] = Point{
			Timestamp: timestamp,
			Value:     math.Float64frombits(lastValue),
		}

		lastTimestamp = timestamp
		lastDelta = delta
	}

	return chunk, nil
}

func CompressChunk(chunk *Chunk) (*DODChunk, error) {
	if len(chunk.Points) == 0 {
		return nil, nil
	}

	count := len(chunk.Points)

	dod := &DODChunk{
		count:          count,
		firstTimestamp: chunk.Points[0].Timestamp,
		lastTimestamp:  chunk.Points[count-1].Timestamp,
		timestamps:     make([]byte, 0, count*8),
		values:         make([]byte, 0, count*8),
	}

	var buf [8]byte

	// First timestamp.
	binary.LittleEndian.PutUint64(
		buf[:],
		uint64(chunk.Points[0].Timestamp),
	)
	dod.timestamps = append(dod.timestamps, buf[:]...)

	// First value.
	lastValue := math.Float64bits(chunk.Points[0].Value)

	binary.LittleEndian.PutUint64(buf[:], lastValue)
	dod.values = append(dod.values, buf[:]...)

	if count == 1 {
		return dod, nil
	}

	// First delta.
	lastTimestamp := chunk.Points[0].Timestamp
	lastDelta := chunk.Points[1].Timestamp - lastTimestamp

	binary.LittleEndian.PutUint64(
		buf[:],
		uint64(lastDelta),
	)
	dod.timestamps = append(dod.timestamps, buf[:]...)

	lastTimestamp = chunk.Points[1].Timestamp

	// Second value encoded as XOR against the first.
	currValue := math.Float64bits(chunk.Points[1].Value)
	xor := currValue ^ lastValue

	binary.LittleEndian.PutUint64(buf[:], xor)
	dod.values = append(dod.values, buf[:]...)

	lastValue = currValue

	// Remaining points: delta-of-delta + XOR.
	for i := 2; i < count; i++ {
		ts := chunk.Points[i].Timestamp

		delta := ts - lastTimestamp
		dodValue := delta - lastDelta

		dod.timestamps = binary.AppendVarint(
			dod.timestamps,
			dodValue,
		)

		lastTimestamp = ts
		lastDelta = delta

		currValue := math.Float64bits(chunk.Points[i].Value)
		xor := currValue ^ lastValue

		binary.LittleEndian.PutUint64(buf[:], xor)
		dod.values = append(dod.values, buf[:]...)

		lastValue = currValue
	}

	return dod, nil
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

func DecodeDODChunk(
	r io.Reader,
	timestamps []byte,
	values []byte,
) (*DODChunk, error) {
	var (
		pointsCount    uint32
		firstTimestamp int64
		timestampsLen  uint32
		valuesLen      uint32
	)

	if err := binary.Read(r, binary.LittleEndian, &pointsCount); err != nil {
		return nil, fmt.Errorf("read count: %w", err)
	}

	if err := binary.Read(r, binary.LittleEndian, &firstTimestamp); err != nil {
		return nil, fmt.Errorf("read first timestamp: %w", err)
	}

	if err := binary.Read(r, binary.LittleEndian, &timestampsLen); err != nil {
		return nil, fmt.Errorf("read timestamps length: %w", err)
	}

	if cap(timestamps) < int(timestampsLen) {
		timestamps = make([]byte, int(timestampsLen))
	} else {
		timestamps = timestamps[:int(timestampsLen)]
	}

	if _, err := io.ReadFull(r, timestamps); err != nil {
		return nil, fmt.Errorf("read timestamps: %w", err)
	}

	if err := binary.Read(r, binary.LittleEndian, &valuesLen); err != nil {
		return nil, fmt.Errorf("read values length: %w", err)
	}

	if cap(values) < int(valuesLen) {
		values = make([]byte, int(valuesLen))
	} else {
		values = values[:int(valuesLen)]
	}

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
	buf := make([]byte, 4+len(points)*16)

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
	var pointsLen uint32

	if err := binary.Read(r, binary.LittleEndian, &pointsLen); err != nil {
		return nil, err
	}

	points := make([]Point, int(pointsLen))

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
