package db

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/belldb/internal/config"
	"github.com/belldb/internal/storage"
	"github.com/belldb/internal/wal"
)

type DB struct {
	kv  *KV
	log *wal.Log
	lsn uint64
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

func (db *DB) Open() error {

	db.kv = NewKV(5000)
	db.kv.onFlush = func(u uint64) {
		db.log.Checkpoint(u)
	}

	err := db.recoverMetadata()
	if err != nil {
		return err
	}

	if err := db.recover(); err != nil {
		return err
	}

	if err := wal.Reset(); err != nil {
		return err
	}

	db.log = wal.NewLog()
	return db.log.Start()

}

func (db *DB) Close() error {
	db.kv.Flush()
	return db.log.Close()
}

func (db *DB) Put(metric string, timestamp int64, value float64) error {
	db.lsn++

	sp := wal.SavePoint{
		Metric: metric,
		Point:  storage.Point{Timestamp: timestamp, Value: value},
	}

	db.log.Append(sp)

	return db.kv.Put(metric, timestamp, value, db.lsn)
}

func (db *DB) Get(metric string, timestamp int64) (float64, error) {
	return db.kv.Get(metric, timestamp)
}

func (db *DB) Range(metric string, start, end int64) []storage.Point {
	return db.kv.Range(metric, start, end)
}

func (db *DB) recoverMetadata() error {
	entries, err := os.ReadDir(config.DATA_DIR)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		metric := entry.Name()

		metaFilePath := filepath.Join(
			config.DATA_DIR,
			metric,
			"meta.json",
		)

		metadata, err := storage.LoadMeta(metaFilePath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}

		series, ok := db.kv.Series[metric]
		if !ok {
			series = &Series{
				name:        metric,
				activeChunk: &storage.Chunk{},
			}

			db.kv.Series[metric] = series
		}

		for _, chunk := range metadata.Chunks {
			series.chunks = append(
				series.chunks,
				storage.ChunkMetaData{
					MinTs: chunk.MinTs,
					MaxTs: chunk.MaxTs,
					Path:  chunk.Path,
				},
			)

		}
	}

	return nil
}

func (db *DB) recover() error {

	records, err := db.log.Recover()
	if err != nil {
		return err
	}

	for _, record := range records {
		series := db.kv.GetOrCreateSeries(record.Metric)
		series.Append(record.Point)
	}

	return nil
}
