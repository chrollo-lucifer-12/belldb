package db

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/belldb/wal"
)

type DB struct {
	kv  *KV
	log *wal.Log
}

func errMetricNotFound(metric string) error {
	return fmt.Errorf("metric not found: %s", metric)
}

func errTimestampNotFound(timestamp int64) error {
	return fmt.Errorf("timestamp not found: %d", timestamp)
}

func NewDB() *DB {
	return &DB{}
}

func (db *DB) Open(filepath string) error {

	db.kv = NewKV()

	fp, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return errors.Join(errors.New("db open"), err)
	}

	db.log = wal.NewLog(fp)

	return db.recover()
}

func (db *DB) Close() error {
	return db.log.Close()
}

func (db *DB) Put(metric string, timestamp int64, value float64) error {

	err := db.log.Write(wal.EncodeRecord(wal.SavePoint{Metric: metric, Timestamp: timestamp, Value: value}))
	if err != nil {
		return err
	}

	return db.kv.Put(metric, timestamp, value)
}

func (db *DB) Get(metric string, timestamp int64) (float64, error) {
	return db.kv.Get(metric, timestamp)
}

func (db *DB) Range(metric string, start, end int64) []Point {
	return db.kv.Range(metric, start, end)
}

func (db *DB) recover() error {
	for {
		start, err := db.log.Seek(0, io.SeekCurrent)
		if err != nil {
			return err
		}

		sp, err := wal.DecodeRecord(db.log.Reader())

		if err == io.EOF {
			break
		}

		if errors.Is(err, io.ErrUnexpectedEOF) {
			if err := db.log.Truncate(start); err != nil {
				return err
			}
			break
		}

		if err != nil {
			return err
		}

		db.kv.Put(sp.Metric, sp.Timestamp, sp.Value)
	}

	return nil
}
