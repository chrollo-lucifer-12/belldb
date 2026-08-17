package wal

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"math"

	"github.com/belldb/internal/storage"
)

func Encode(p SavePoint) []byte {
	var buf bytes.Buffer

	binary.Write(
		&buf,
		binary.LittleEndian,
		uint16(len(p.Metric)),
	)

	buf.WriteString(p.Metric)

	storage.EncodePoints(&buf, []storage.Point{p.Point})

	return buf.Bytes()
}

func EncodeRecord(buf []byte, sp SavePoint) []byte {
	metricLen := len(sp.Metric)

	payloadLen := 2 + metricLen + 4 + 8 + 8
	recordLen := 8 + 4 + payloadLen + 4

	if cap(buf) < recordLen {
		buf = make([]byte, recordLen)
	} else {
		buf = buf[:recordLen]
	}

	binary.LittleEndian.PutUint64(buf[0:8], sp.LSN)

	binary.LittleEndian.PutUint32(
		buf[8:12],
		uint32(payloadLen),
	)

	offset := 12

	binary.LittleEndian.PutUint16(
		buf[offset:offset+2],
		uint16(metricLen),
	)
	offset += 2

	copy(buf[offset:], sp.Metric)
	offset += metricLen

	binary.LittleEndian.PutUint32(buf[offset:offset+4], 1)
	offset += 4

	binary.LittleEndian.PutUint64(
		buf[offset:offset+8],
		uint64(sp.Point.Timestamp),
	)
	offset += 8

	binary.LittleEndian.PutUint64(
		buf[offset:offset+8],
		math.Float64bits(sp.Point.Value),
	)
	offset += 8

	checksum := crc32.ChecksumIEEE(buf[12 : 12+payloadLen])

	binary.LittleEndian.PutUint32(
		buf[12+payloadLen:12+payloadLen+4],
		checksum,
	)

	return buf
}

func DecodeRecord(r io.Reader) (SavePoint, error) {

	var lsn uint64

	if err := binary.Read(r, binary.LittleEndian, &lsn); err != nil {
		return SavePoint{}, err
	}

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

	return DecodePayload(bytes.NewReader(payload), lsn)
}

func DecodePayload(r io.Reader, lsn uint64) (SavePoint, error) {
	var metricLen uint16

	if err := binary.Read(r, binary.LittleEndian, &metricLen); err != nil {
		return SavePoint{}, err
	}

	metric := make([]byte, metricLen)

	if _, err := io.ReadFull(r, metric); err != nil {
		return SavePoint{}, err
	}

	points, err := storage.DecodePoints(r)
	if err != nil {
		return SavePoint{}, err
	}

	return SavePoint{
		LSN:    lsn,
		Metric: string(metric),
		Point:  points[0],
	}, nil
}
