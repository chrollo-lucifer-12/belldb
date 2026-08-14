package db

import "github.com/belldb/internal/storage"

const ChunkSize = 1000

type Series struct {
	activeChunk *storage.Chunk
	name        string
	chunks      []storage.ChunkMetaData
}

func (s *Series) Append(p storage.Point) (flushed bool, err error) {
	if s.activeChunk == nil {
		s.activeChunk = &storage.Chunk{}
	}

	flushed = false
	if len(s.activeChunk.Points) == ChunkSize {
		meta, err := storage.Flush(s.name, s.activeChunk.Points)
		if err != nil {
			return flushed, err
		}
		flushed = true
		s.chunks = append(s.chunks, meta)
		s.activeChunk = &storage.Chunk{
			Points: make([]storage.Point, 0, ChunkSize),
		}
	}

	s.activeChunk.Points =
		append(s.activeChunk.Points, p)

	return flushed, nil
}

func (s *Series) Get(timestamp int64) (float64, error) {

	chunkIdx := s.findChunk(timestamp)

	if chunkIdx == -1 {
		return -1, errTimestampNotFound(timestamp)
	}

	var err error
	var points []storage.Point

	if chunkIdx == len(s.chunks) {
		points = s.activeChunk.Points
	} else {
		points, err = storage.LoadChunk(s.chunks[chunkIdx].Path)
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
	var result []storage.Point

	startChunk := s.findChunk(start)
	endChunk := s.findChunk(end - 1)

	if startChunk == -1 {
		startChunk = 0
	}

	if endChunk == -1 {
		endChunk = len(s.chunks)
	}

	for startChunk <= endChunk {

		var points []storage.Point
		var err error

		if startChunk == len(s.chunks) {
			points = s.activeChunk.Points
		} else {
			points, err = storage.LoadChunk(s.chunks[startChunk].Path)
			if err != nil {
				return nil
			}
		}

		l := lowerBound(points, start)
		r := lowerBound(points, end)

		result = append(result, points[l:r]...)

		startChunk++
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

	if s.activeChunk != nil && len(s.activeChunk.Points) > 0 {
		points := s.activeChunk.Points

		if points[0].Timestamp <= timestamp &&
			timestamp <= points[len(points)-1].Timestamp {
			return len(s.chunks)
		}
	}

	return -1
}
