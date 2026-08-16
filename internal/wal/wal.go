package wal

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/belldb/internal/config"
	"github.com/belldb/internal/storage"
)

type Log struct {
	dir string

	aof *os.File
	buf *bufio.Writer

	queue chan SavePoint
	done  chan struct{}

	wg sync.WaitGroup

	errMu sync.Mutex
	err   error
}

type SavePoint struct {
	Metric string
	Point  storage.Point
}

func NewLog() (*Log, error) {

	dir := config.LOG_DIR

	path := filepath.Join(dir, "wal.db")

	aof, err := os.OpenFile(
		path,
		os.O_CREATE|os.O_RDWR|os.O_APPEND,
		0644,
	)
	if err != nil {
		return nil, err
	}

	l := &Log{
		dir:   dir,
		aof:   aof,
		buf:   bufio.NewWriterSize(aof, 1024*1024),
		queue: make(chan SavePoint, 8192),
		done:  make(chan struct{}),
	}

	l.wg.Add(1)
	go l.startBackgroundSync(10 * time.Millisecond)

	return l, nil
}

func (log *Log) Close() error {
	close(log.done)

	log.wg.Wait()

	if err := log.getError(); err != nil {
		log.aof.Close()
		return err
	}

	if err := log.Sync(); err != nil {
		log.aof.Close()
		return err
	}

	return log.aof.Close()
}

func (log *Log) Append(sp SavePoint) error {
	select {
	case log.queue <- sp:
		return nil
	case <-log.done:
		return os.ErrClosed
	}
}

func (log *Log) Write(data []byte) error {
	_, err := log.buf.Write(data)
	return err
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

func (log *Log) Sync() error {
	if err := log.buf.Flush(); err != nil {
		return err
	}
	return log.aof.Sync()
}

func (log *Log) startBackgroundSync(interval time.Duration) {
	defer log.wg.Done()

	var buf []byte

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-log.done:
			if err := log.drain(); err != nil {
				log.setError(err)
				return
			}

			if err := log.Sync(); err != nil {
				log.setError(err)
			}

			return

		case sp := <-log.queue:
			buf = EncodeRecord(buf, sp)

			if err := log.Write(buf); err != nil {
				log.setError(err)
				return
			}

			for i := 0; i < 256; i++ {
				select {
				case sp := <-log.queue:
					buf = EncodeRecord(buf, sp)

					if err := log.Write(buf); err != nil {
						log.setError(err)
						return
					}
				default:
					i = 256
				}
			}

		case <-ticker.C:
			if err := log.Sync(); err != nil {
				log.setError(err)
				return
			}
		}
	}
}

func (log *Log) drain() error {

	var buf []byte

	for {
		select {
		case sp := <-log.queue:
			buf = EncodeRecord(buf, sp)

			if err := log.Write(buf); err != nil {
				return err
			}
		default:
			return nil
		}
	}
}

func (log *Log) setError(err error) {
	log.errMu.Lock()
	log.err = err
	log.errMu.Unlock()
}

func (log *Log) getError() error {
	log.errMu.Lock()
	defer log.errMu.Unlock()

	return log.err
}
