package db

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/belldb/wal"
)

var DATA_DIR string

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

func (db *DB) Open(path string) error {

	db.kv = NewKV()

	fp, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return errors.Join(errors.New("db open"), err)
	}

	db.log = wal.NewLog(fp)

	if err := db.recoverMetrics(); err != nil {
		return err
	}

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
		sp, err := wal.DecodeRecord(db.log.Reader())

		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}

		if err != nil {
			return err
		}

		series, ok := db.kv.Series[sp.Metric]

		if !ok {
			series = &Series{name: sp.Metric}

			series.activeChunk = 0
			series.AddChunk()

			db.kv.Series[sp.Metric] = series
		}

		series.Append(Point{Timestamp: sp.Timestamp, Value: sp.Value})
	}

	return nil
}

func (db *DB) recoverMetrics() error {

	metrics, err := os.ReadDir(DATA_DIR)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}

	for _, metricDir := range metrics {
		if !metricDir.IsDir() {
			continue
		}

		metric := metricDir.Name()

		chunks, err := LoadChunks(filepath.Join(DATA_DIR, metric))
		if err != nil {
			return err
		}

		series := &Series{
			Chunks: chunks,
			name:   metric,
		}

		db.kv.Series[metric] = series
	}

	for _, series := range db.kv.Series {
		series.activeChunk = len(series.Chunks)
		series.AddChunk()
	}

	return nil
}
