package wal

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/belldb/internal/storage"
)

type Log struct {
	dir          string
	aof          *os.File
	checkpointID uint64
}

type SavePoint struct {
	Metric string
	Point  storage.Point
}

func NewLog(dir string) (*Log, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	path := filepath.Join(dir, "wal.db")

	aof, err := os.OpenFile(
		path,
		os.O_CREATE|os.O_RDWR|os.O_APPEND,
		0644,
	)
	if err != nil {
		return nil, err
	}

	return &Log{
		dir: dir,
		aof: aof,
	}, nil
}

func (log *Log) Close() error {
	return log.aof.Close()
}

func (log *Log) Write(data []byte) error {
	_, err := log.aof.Write(data)
	if err != nil {
		return err
	}

	return log.aof.Sync()
}

func (log *Log) Read(buf []byte, offset int64) error {
	_, err := log.aof.ReadAt(buf, offset)
	return err
}

func (log *Log) Seek(offfset int64, whence int) (int64, error) {
	return log.aof.Seek(offfset, whence)
}

func (log *Log) Truncate(size int64) error {
	return log.aof.Truncate(size)
}

func (log *Log) Reader() io.Reader {
	return log.aof
}

func (log *Log) Offset() (int64, error) {
	return log.aof.Seek(0, io.SeekEnd)
}

func (log *Log) Checkpoint() error {
	log.checkpointID++

	offset, err := log.Offset()
	if err != nil {
		log.checkpointID--
		return err
	}

	checkpointDir := filepath.Join(
		log.dir,
		fmt.Sprintf("checkpoint.%08d", log.checkpointID),
	)

	if err := os.MkdirAll(checkpointDir, 0755); err != nil {
		log.checkpointID--
		return err
	}

	checkpointPath := filepath.Join(
		checkpointDir,
		"00000000",
	)

	fp, err := os.OpenFile(
		checkpointPath,
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		0644,
	)
	if err != nil {
		log.checkpointID--
		return err
	}

	if err := binary.Write(
		fp,
		binary.LittleEndian,
		offset,
	); err != nil {
		fp.Close()
		log.checkpointID--
		return err
	}

	if err := fp.Sync(); err != nil {
		fp.Close()
		log.checkpointID--
		return err
	}

	if err := fp.Close(); err != nil {
		log.checkpointID--
		return err
	}

	return nil
}
