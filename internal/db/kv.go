package db

import (
	"os"
	"path/filepath"
	"sync"

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

	wg sync.WaitGroup
}

func NewKV(flushQueueSize int) *KV {
	kv := &KV{Series: make(map[string]*Series), flushQueue: make(chan *FlushTask, flushQueueSize)}
	go kv.startWorkers(1)
	return kv
}

func (kv *KV) GetOrCreateSeries(metric string) *Series {
	s, ok := kv.Series[metric]

	if ok {
		return s
	}

	dir := filepath.Join(config.DATA_DIR, metric)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil
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
		kv.wg.Add(1)

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

func (kv *KV) startWorkers(numWorkers int) {
	for i := 0; i < numWorkers; i++ {
		go func() {
			for task := range kv.flushQueue {
				dodChunk, err := storage.CompressChunk(task.Chunk)
				if err != nil {
					continue
				}

				meta, err := storage.FlushDODChunk(
					task.SeriesName,
					dodChunk,
				)
				if err != nil {
					continue
				}

				kv.wg.Done()

				series := kv.GetOrCreateSeries(task.SeriesName)
				series.chunks = append(series.chunks, meta)
			}
		}()
	}
}

func (kv *KV) WaitForFlush() {
	kv.wg.Wait()
}
