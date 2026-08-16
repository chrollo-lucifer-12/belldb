package db

import (
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

func (db *DB) Open() error {

	db.kv = NewKV(5000)

	maxTs, err := db.recoverMetadata()
	if err != nil {
		return err
	}

	db.log, err = wal.NewLog()
	if err != nil {
		return err
	}

	return db.recover(maxTs)
}

func (db *DB) Close() error {
	//	db.kv.Flush()
	return db.log.Close()
}

func (db *DB) Put(metric string, timestamp int64, value float64) error {

	sp := wal.SavePoint{
		Metric: metric,
		Point:  storage.Point{Timestamp: timestamp, Value: value},
	}
	db.log.Append(sp)

	return db.kv.Put(metric, timestamp, value)
}

func (db *DB) Get(metric string, timestamp int64) (float64, error) {
	return db.kv.Get(metric, timestamp)
}

func (db *DB) Range(metric string, start, end int64) []storage.Point {
	return db.kv.Range(metric, start, end)
}

func (db *DB) recoverMetadata() (int64, error) {
	entries, err := os.ReadDir(config.DATA_DIR)
	if err != nil {
		if os.IsNotExist(err) {
			return -1, nil
		}
		return -1, err
	}

	maxTs := int64(0)

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
			return -1, err
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
			maxTs = max(maxTs, chunk.MaxTs)
		}
	}

	return maxTs, nil
}

func (db *DB) recover(maxTs int64) error {
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
			series = &Series{name: sp.Metric, cache: NewChunkCache(16)}

			series.activeChunk = &storage.Chunk{
				Points: make([]storage.Point, 0, ChunkSize),
			}

			db.kv.Series[sp.Metric] = series

		}

		if sp.Point.Timestamp <= maxTs {
			continue
		}

		if series.activeChunk == nil {
			series.activeChunk = &storage.Chunk{}
		}

		series.activeChunk.Points = append(series.activeChunk.Points, sp.Point)

		//	fmt.Println(sp)
	}

	return nil
}
