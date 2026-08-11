package db

import (
	"io"
	"os"
)

type Log struct {
	aof *os.File
}

type SavePoint struct {
	metric    string
	timestamp int64
	value     float64
}

func NewLog(aof *os.File) *Log {
	return &Log{aof: aof}
}

func (log *Log) Write(data []byte) error {
	_, err := log.aof.Write(data)
	return err
}

func (log *Log) Read(buf []byte, offset int64) error {
	_, err := log.aof.ReadAt(buf, offset)
	return err
}



func EncodePoint(p SavePoint) []byte {
	buf := make([]byte, 2+len(p.metric)+8+8)

	binary.LittleEndian.PutUint16(buf[0:2], uint16(len(p.metric)))

	copy(buf[2:2+len(p.metric)], p.metric)

	offset := 2 + len(p.metric)

	binary.LittleEndian.PutUint64(
		buf[offset:offset+8],
		uint64(p.timestamp),
	)

	offset += 8

	binary.LittleEndian.PutUint64(
		buf[offset:offset+8],
		math.Float64bits(p.value)
	)

	return buf
}

func DecodePoint(r io.Reader) (SavePoint, error) {

	var metricLen uint16
	if err := binary.Read(r, binary.LittleEndian, &metricLen); err != nil {
		return SavePoint{}, nil
	}

	metric := make([]byte, metricLen)

	if _, err := io.ReadFull(r, metric); err != nil {
		return err
	}

	var timestamp int64
	if err := binary.Read(r, binary.LittleEndian, &timestamp); err != nil {
		return SavePoint{}, nil
	}

	var valueBits uint64
	if err := binary.Read(r, binary.LittleEndian, &valueBits); err != nil {
		return SavePoint{}, nil
	}

	return SavePoint{
		metric:    string(metric),
		timestamp: timestamp,
		value:     math.Float64frombits(valueBits),
	}
}
