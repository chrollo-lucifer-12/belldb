package db

import "github.com/belldb/internal/storage"

const ChunkSize = 5000

type Series struct {
	activeChunk *storage.Chunk
	name        string
	chunks      []storage.ChunkMetaData

	decodeTimestamps []byte
	decodeValues     []byte

	cache *Cache
}

func (s *Series) Append(p storage.Point) (fullChunk *storage.Chunk) {

	s.activeChunk.Points =
		append(s.activeChunk.Points, p)

	if len(s.activeChunk.Points) >= ChunkSize {
		fullChunk = s.activeChunk

		s.activeChunk = &storage.Chunk{
			Points: make([]storage.Point, 0, ChunkSize),
		}

		return fullChunk
	}

	return nil
}

func (s *Series) Get(timestamp int64) (float64, error) {

	chunkIdx := s.findChunk(timestamp)

	if chunkIdx == -1 {
		return -1, errTimestampNotFound(timestamp)
	}

	var points []storage.Point
	var err error

	if chunkIdx == len(s.chunks) {
		points = s.activeChunk.Points
	} else {
		points, err = s.loadChunk(chunkIdx)
		if err != nil {
			return -1, err
		}
	}

	idx := lowerBound(points, timestamp)

	if idx < len(points) && points[idx].Timestamp == timestamp {
		return points[idx].Value, nil
	}

	return -1, errTimestampNotFound(timestamp)

}

func (s *Series) Range(start, end int64) []storage.Point {
	startChunk := s.findChunk(start)
	endChunk := s.findChunk(end - 1)

	if startChunk == -1 {
		return nil
	}

	if endChunk == -1 {
		return nil
	}

	capacity := 0

	for i := startChunk; i <= endChunk; i++ {
		if i == len(s.chunks) {
			capacity += len(s.activeChunk.Points)
		} else {
			capacity += s.chunks[i].Count
		}
	}

	result := make([]storage.Point, 0, capacity)

	for i := startChunk; i <= endChunk; i++ {
		var points []storage.Point

		if i == len(s.chunks) {
			points = s.activeChunk.Points
		} else {
			var err error
			points, err = s.loadChunk(i)
			if err != nil {
				return nil
			}
		}

		l := 0
		r := len(points)

		if i == startChunk {
			l = lowerBound(points, start)
		}

		if i == endChunk {
			r = lowerBound(points, end)
		}

		if l < r {
			result = append(result, points[l:r]...)
		}
	}

	return result
}

func lowerBound(points []storage.Point, timestamp int64) int {

	low, high := 0, len(points)

	for low < high {
		mid := low + (high-low)/2

		if points[mid].Timestamp < timestamp {
			low = mid + 1
		} else {
			high = mid
		}
	}

	return low
}

func (s *Series) findChunk(timestamp int64) int {
	if len(s.chunks) == 0 {
		if s.activeChunk != nil && len(s.activeChunk.Points) > 0 {
			return len(s.chunks)
		}
		return -1
	}

	low, high := 0, len(s.chunks)-1

	for low <= high {
		mid := low + (high-low)/2
		chunk := s.chunks[mid]

		if timestamp < chunk.MinTs {
			high = mid - 1
		} else if timestamp > chunk.MaxTs {
			low = mid + 1
		} else {
			return mid
		}
	}

	if low == 0 {
		return 0
	}

	if low == len(s.chunks) {
		if s.activeChunk != nil && len(s.activeChunk.Points) > 0 {
			return len(s.chunks)
		}

		return len(s.chunks) - 1
	}

	return low
}

func (s *Series) loadChunk(idx int) ([]storage.Point, error) {
	if points, ok := s.cache.Get(idx); ok {
		return points, nil
	}

	dodChunk, err := storage.LoadDODChunk(s.chunks[idx].Path, &s.decodeTimestamps, &s.decodeValues)
	if err != nil {
		return nil, err
	}

	chunk, err := storage.DecompressChunk(dodChunk)
	if err != nil {
		return nil, err
	}

	s.cache.Put(idx, chunk.Points)

	return chunk.Points, nil
}
