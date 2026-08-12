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
		err := storage.Flush(s.name, s.activeChunk.Points)
		if err != nil {

			return flushed, err
		}
		flushed = true
		s.activeChunk = &storage.Chunk{}
	}

	s.activeChunk.Points =
		append(s.activeChunk.Points, p)

	return flushed, nil
}

func (s *Series) Get(timestamp int64) (float64, error) {
	for _, chunk := range s.chunks {

		if timestamp > chunk.MaxTs || timestamp < chunk.MinTs {
			continue
		}

		points, err := storage.LoadChunk(chunk.Path)
		if err != nil {
			return -1, err
		}

		idx := lowerBound(points, timestamp)

		if idx < len(points) && points[idx].Timestamp == timestamp {
			return points[idx].Value, nil
		}

		return -1, errTimestampNotFound(timestamp)
	}

	return -1, errTimestampNotFound(timestamp)
}

func (s *Series) Range(start, end int64) []storage.Point {
	var result []storage.Point

	for _, chunk := range s.chunks {

		if chunk.MaxTs < start {
			continue
		}

		if chunk.MinTs >= end {
			break
		}

		points, err := storage.LoadChunk(chunk.Path)
		if err != nil {
			return nil
		}

		l := lowerBound(points, start)
		r := lowerBound(points, end)

		result = append(result, points[l:r]...)
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
