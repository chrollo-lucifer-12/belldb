package db

const ChunkSize = 1000

type Point struct {
	Timestamp int64
	Value     float64
}

type Chunk struct {
	Points []Point
}

type Series struct {
	Chunks      []*Chunk
	activeChunk int
	name        string
}

func (s *Series) AddChunk() {
	s.Chunks = append(s.Chunks, &Chunk{})
}

func (s *Series) Append(p Point) error {
	if len(s.Chunks) == 0 {
		s.AddChunk()
	}

	if len(s.Chunks[s.activeChunk].Points) == ChunkSize {
		err := s.Flush(s.name)
		if err != nil {
			return err
		}
		s.activeChunk++
		s.AddChunk()
	}

	s.Chunks[s.activeChunk].Points =
		append(s.Chunks[s.activeChunk].Points, p)

	return nil
}

func (s *Series) Get(timestamp int64) (float64, error) {
	for _, chunk := range s.Chunks {
		points := chunk.Points

		if len(points) == 0 {
			continue
		}

		if timestamp > points[len(points)-1].Timestamp || timestamp < points[0].Timestamp {
			continue
		}

		idx := lowerBound(points, timestamp)

		if idx < len(points) && points[idx].Timestamp == timestamp {
			return points[idx].Value, nil
		}

		return -1, errTimestampNotFound(timestamp)
	}

	return -1, errTimestampNotFound(timestamp)
}

func (s *Series) Range(start, end int64) []Point {
	var result []Point

	for _, chunk := range s.Chunks {
		points := chunk.Points
		if len(points) == 0 {
			continue
		}

		if points[len(points)-1].Timestamp < start {
			continue
		}

		if points[0].Timestamp >= end {
			break
		}

		l := lowerBound(points, start)
		r := lowerBound(points, end)

		result = append(result, points[l:r]...)
	}

	return result
}

func lowerBound(points []Point, timestamp int64) int {

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
