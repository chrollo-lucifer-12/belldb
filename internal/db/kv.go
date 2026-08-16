package db

import (
	"os"
	"path/filepath"

	"github.com/belldb/internal/config"
	"github.com/belldb/internal/storage"
)

type FlushTask struct {
	SeriesName string
	Chunk      *storage.Chunk
}

type KV struct {
	Series     map[string]*Series
	flushQueue chan *FlushTask
}

func NewKV(flushQueueSize int) *KV {
	kv := &KV{Series: make(map[string]*Series), flushQueue: make(chan *FlushTask, flushQueueSize)}
	go kv.startFlushWorkers(8)

	return kv
}

func (kv *KV) startFlushWorkers(numWorkers int) {
	for i := 0; i < numWorkers; i++ {
		go func() {
			for task := range kv.flushQueue {
				dodChunk, err := storage.CompressChunk(task.Chunk)
				if err != nil {
					continue
				}

				dir := filepath.Join(config.DATA_DIR, task.SeriesName)
				_ = os.MkdirAll(dir, 0755)

				meta, err := storage.FlushDODChunk(task.SeriesName, dodChunk)
				if err != nil {
					continue
				}

				s := kv.GetOrCreateSeries(task.SeriesName)

				s.chunks = append(s.chunks, meta)
			}
		}()
	}
}

func (kv *KV) GetOrCreateSeries(metric string) *Series {
	s, ok := kv.Series[metric]

	if ok {
		return s
	}

	if s, ok = kv.Series[metric]; ok {
		return s
	}

	s = &Series{
		name: metric,
		activeChunk: &storage.Chunk{
			Points: make([]storage.Point, 0, ChunkSize),
		},
	}
	kv.Series[metric] = s
	return s
}

func (kv *KV) Put(metric string, timestamp int64, value float64) error {
	series := kv.GetOrCreateSeries(metric)

	fullChunk := series.Append(storage.Point{Timestamp: timestamp, Value: value})
	if fullChunk != nil {
		kv.flushQueue <- &FlushTask{
			SeriesName: metric,
			Chunk:      fullChunk,
		}
	}

	return nil
}

func (kv *KV) Get(metric string, timestamp int64) (float64, error) {
	series, ok := kv.Series[metric]
	if !ok {
		return -1, errMetricNotFound(metric)
	}

	return series.Get(timestamp)
}

func (kv *KV) Range(metric string, start, end int64) []storage.Point {
	series, ok := kv.Series[metric]
	if !ok {
		return nil
	}

	return series.Range(start, end)
}
