package db

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/belldb/internal/config"
	"github.com/belldb/internal/storage"
	"github.com/belldb/internal/wal"
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

func (db *DB) Open(path string) error {

	db.kv = NewKV()

	fp, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return errors.Join(errors.New("db open"), err)
	}

	if err := db.recoverMetadata(); err != nil {
		return err
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

			series.activeChunk = &storage.Chunk{}

			db.kv.Series[sp.Metric] = series
		}

		series.Append(storage.Point{Timestamp: sp.Timestamp, Value: sp.Value})
	}

	return nil
}
