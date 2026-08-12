package wal

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"

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

	return DecodePayload(bytes.NewReader(payload))
}

func DecodePayload(r io.Reader) (SavePoint, error) {
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
		Metric: string(metric),
		Point:  points[0],
	}, nil
}
