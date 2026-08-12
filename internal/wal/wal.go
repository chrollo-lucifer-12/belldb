package wal

import (
	"io"
	"os"

	"github.com/belldb/internal/storage"
)

type Log struct {
	aof *os.File
}

type SavePoint struct {
	Metric string
	Point  storage.Point
}

func NewLog(aof *os.File) *Log {
	return &Log{aof: aof}
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
