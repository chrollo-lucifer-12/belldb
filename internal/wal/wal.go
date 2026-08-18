package wal

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/belldb/internal/config"
	"github.com/belldb/internal/storage"
)

const WalLimit = 100

type Log struct {
	file     *os.File
	buf      *bufio.Writer
	nrecords int

	segmentID int

	queue chan SavePoint
	done  chan struct{}

	wg sync.WaitGroup

	errMu sync.Mutex
	err   error
}

type SavePoint struct {
	LSN    uint64
	Metric string
	Point  storage.Point
}

func NewLog() *Log {

	return &Log{
		segmentID: 0,
		nrecords:  0,
		queue:     make(chan SavePoint, 8192),
		done:      make(chan struct{}),
	}

}

func Reset() error {
	dir := filepath.Join(config.LOG_DIR)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		err := os.RemoveAll(filepath.Join(dir, entry.Name()))
		if err != nil {
			return err
		}
	}

	path := filepath.Join(dir, "checkpoint")

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func (log *Log) Start() error {
	if err := log.rotate(); err != nil {
		return err
	}

	log.wg.Add(1)
	go log.startBackgroundSync(10 * time.Millisecond)

	return nil
}

func (log *Log) Close() error {
	close(log.done)

	log.wg.Wait()

	if err := log.getError(); err != nil {
		_ = storage.CloseFile(log.file)
		return err
	}

	if err := log.sync(); err != nil {
		_ = storage.CloseFile(log.file)
		return err
	}

	return storage.CloseFile(log.file)
}

func (log *Log) Append(sp SavePoint) error {
	select {
	case log.queue <- sp:
		return nil
	case <-log.done:
		return os.ErrClosed
	}
}

func (log *Log) write(data []byte) error {
	if _, err := log.buf.Write(data); err != nil {
		return fmt.Errorf("write WAL: %w", err)
	}

	return nil
}

func (log *Log) Reader() io.Reader {
	return log.file
}

func (log *Log) sync() error {
	if err := log.buf.Flush(); err != nil {
		return fmt.Errorf("flush WAL buffer: %w", err)
	}

	if err := storage.SyncFile(log.file); err != nil {
		return err
	}

	return nil
}

func (log *Log) startBackgroundSync(interval time.Duration) {
	defer log.wg.Done()

	var buf []byte

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	dirty := false

	for {
		select {
		case <-log.done:
			if err := log.drain(); err != nil {
				log.setError(err)
				return
			}

			if err := log.sync(); err != nil {
				log.setError(err)
			}

			return

		case sp := <-log.queue:
			buf = EncodeRecord(buf, sp)

			if err := log.write(buf); err != nil {
				log.setError(err)
				return
			}

			dirty = true
			log.nrecords++

			for i := 0; i < 256; i++ {
				select {
				case sp := <-log.queue:
					buf = EncodeRecord(buf, sp)

					if err := log.write(buf); err != nil {
						log.setError(err)
						return
					}
					dirty = true
					log.nrecords++
				default:
					i = 256
				}
			}

			if log.nrecords >= WalLimit {
				if err := log.drain(); err != nil {
					log.setError(err)
					return
				}

				if err := log.sync(); err != nil {
					log.setError(err)
					return
				}

				if err := log.rotate(); err != nil {
					log.setError(err)
					return
				}

				dirty = false
			}

		case <-ticker.C:
			if !dirty {
				continue
			}

			if err := log.sync(); err != nil {
				log.setError(err)
				return
			}

			dirty = true
		}
	}
}

func (log *Log) drain() error {

	var buf []byte

	for {
		select {
		case sp := <-log.queue:
			buf = EncodeRecord(buf, sp)

			if err := log.write(buf); err != nil {
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
