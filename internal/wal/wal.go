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

const WalLimit = 100

type Log struct {
	aof      *os.File
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

	return os.Remove(filepath.Join(dir, "checkpoint"))
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
		log.aof.Close()
		return err
	}

	if err := log.sync(); err != nil {
		log.aof.Close()
		return err
	}

	return log.aof.Close()
}

func (log *Log) Append(sp SavePoint) error {
	select {
	case log.queue <- sp:
		log.nrecords++
		return nil
	case <-log.done:
		return os.ErrClosed
	}
}

func (log *Log) write(data []byte) error {
	_, err := log.buf.Write(data)
	return err
}

func (log *Log) Reader() io.Reader {
	return log.aof
}

func (log *Log) sync() error {
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

			for i := 0; i < 256; i++ {
				select {
				case sp := <-log.queue:
					buf = EncodeRecord(buf, sp)

					if err := log.write(buf); err != nil {
						log.setError(err)
						return
					}
				default:
					i = 256
				}
			}

			if log.nrecords > WalLimit {
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
			}

		case <-ticker.C:
			if err := log.sync(); err != nil {
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
