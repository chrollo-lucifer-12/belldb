package db

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"math"
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
	if err != nil {
		return err
	}

	err = log.aof.Sync()
	if err != nil {
		return err
	}

	return nil
}

func (log *Log) Read(buf []byte, offset int64) error {
	_, err := log.aof.ReadAt(buf, offset)
	return err
}

func Encode(p SavePoint) []byte {
	paylod := make([]byte, 2+len(p.metric)+8+8)

	binary.LittleEndian.PutUint16(paylod[0:2], uint16(len(p.metric)))

	copy(paylod[2:2+len(p.metric)], p.metric)

	offset := 2 + len(p.metric)

	binary.LittleEndian.PutUint64(
		paylod[offset:offset+8],
		uint64(p.timestamp),
	)

	offset += 8

	binary.LittleEndian.PutUint64(
		paylod[offset:offset+8],
		math.Float64bits(p.value),
	)

	return paylod
}

func EncodeRecord(sp SavePoint) []byte {

	payload := Encode(sp)

	checksum := crc32.ChecksumIEEE(payload)

	buf := make([]byte, 4+len(payload)+4)

	binary.LittleEndian.PutUint32(buf[0:4], uint32(len(payload)))

	copy(buf[4:4+len(payload)], payload)

	binary.LittleEndian.PutUint32(buf[4+len(payload):], checksum)

	return buf
}

func DecodeRecord(r io.Reader) (SavePoint, error) {
	var length uint32

	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		return SavePoint{}, err
	}

	payload := make([]byte, length)

	if _, err := io.ReadFull(r, payload); err != nil {
		return SavePoint{}, err
	}

	var storedChecksum uint32

	if err := binary.Read(r, binary.LittleEndian, &storedChecksum); err != nil {
		return SavePoint{}, err
	}

	calculatedChecksum := crc32.ChecksumIEEE(payload)

	if calculatedChecksum != storedChecksum {
		return SavePoint{}, fmt.Errorf("checksum mismatch")
	}

	return DecodePoint(bytes.NewReader(payload))
}

func DecodePoint(r io.Reader) (SavePoint, error) {
	var metricLen uint16

	if err := binary.Read(r, binary.LittleEndian, &metricLen); err != nil {
		return SavePoint{}, err
	}

	metric := make([]byte, metricLen)

	if _, err := io.ReadFull(r, metric); err != nil {
		return SavePoint{}, err
	}

	var timestamp int64
	if err := binary.Read(r, binary.LittleEndian, &timestamp); err != nil {
		return SavePoint{}, err
	}

	var valueBits uint64
	if err := binary.Read(r, binary.LittleEndian, &valueBits); err != nil {
		return SavePoint{}, err
	}

	return SavePoint{
		metric:    string(metric),
		timestamp: timestamp,
		value:     math.Float64frombits(valueBits),
	}, nil
}
