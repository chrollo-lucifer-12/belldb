package db

import "testing"

func TestSeriesAppend(t *testing.T) {
	s := &Series{}

	for i := 0; i < 2500; i++ {
		s.Append(Point{
			Timestamp: int64(i),
			Value:     float64(i) * 10,
		})
	}

	if len(s.Chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(s.Chunks))
	}

	if len(s.Chunks[0].Points) != 1000 {
		t.Fatalf("chunk 0: expected 1000 points, got %d", len(s.Chunks[0].Points))
	}

	if len(s.Chunks[1].Points) != 1000 {
		t.Fatalf("chunk 1: expected 1000 points, got %d", len(s.Chunks[1].Points))
	}

	if len(s.Chunks[2].Points) != 500 {
		t.Fatalf("chunk 2: expected 500 points, got %d", len(s.Chunks[2].Points))
	}
}

func TestSeriesGet(t *testing.T) {
	oldDataDir := DATA_DIR
	DATA_DIR = t.TempDir()
	defer func() {
		DATA_DIR = oldDataDir
	}()

	s := &Series{
		name: "test",
	}

	for i := 0; i < 2500; i++ {
		if err := s.Append(Point{
			Timestamp: int64(i),
			Value:     float64(i),
		}); err != nil {
			t.Fatalf("Append(%d) failed: %v", i, err)
		}
	}

	got, err := s.Get(1000)
	if err != nil {
		t.Fatalf("Get(1000) returned error: %v", err)
	}

	if got != 1000 {
		t.Fatalf("expected value 1000, got %v", got)
	}
}

func TestSeriesGetNotFound(t *testing.T) {
	s := &Series{}

	for i := 0; i < 2500; i++ {
		s.Append(Point{
			Timestamp: int64(i),
			Value:     float64(i),
		})
	}

	tests := []int64{-1, 2500, 9999}

	for _, timestamp := range tests {
		_, err := s.Get(timestamp)

		if err == nil {
			t.Fatalf("Get(%d): expected error", timestamp)
		}
	}
}

func TestSeriesRange(t *testing.T) {
	oldDataDir := DATA_DIR
	DATA_DIR = t.TempDir()
	defer func() {
		DATA_DIR = oldDataDir
	}()

	s := &Series{
		name: "test",
	}

	for i := 0; i < 2500; i++ {
		err := s.Append(Point{
			Timestamp: int64(i),
			Value:     float64(i),
		})
		if err != nil {
			t.Fatalf("Append(%d) failed: %v", i, err)
		}
	}

	result := s.Range(950, 1050)

	if len(result) != 100 {
		t.Fatalf("expected 100 points, got %d", len(result))
	}

	if result[0].Timestamp != 950 {
		t.Fatalf("expected first timestamp 950, got %d",
			result[0].Timestamp)
	}

	if result[len(result)-1].Timestamp != 1049 {
		t.Fatalf("expected last timestamp 1049, got %d",
			result[len(result)-1].Timestamp)
	}
}

func TestSeriesRangeSingleChunk(t *testing.T) {
	s := &Series{}

	for i := 0; i < 100; i++ {
		s.Append(Point{
			Timestamp: int64(i),
			Value:     float64(i),
		})
	}

	result := s.Range(20, 50)

	if len(result) != 30 {
		t.Fatalf("expected 30 points, got %d", len(result))
	}

	for i, p := range result {
		expected := int64(i + 20)

		if p.Timestamp != expected {
			t.Fatalf("index %d: expected timestamp %d, got %d",
				i, expected, p.Timestamp)
		}
	}
}

func TestLowerBound(t *testing.T) {
	points := []Point{
		{Timestamp: 10},
		{Timestamp: 20},
		{Timestamp: 30},
		{Timestamp: 40},
		{Timestamp: 50},
	}

	tests := []struct {
		timestamp int64
		expected  int
	}{
		{5, 0},
		{10, 0},
		{15, 1},
		{20, 1},
		{35, 3},
		{50, 4},
		{60, 5},
	}

	for _, tt := range tests {
		got := lowerBound(points, tt.timestamp)

		if got != tt.expected {
			t.Fatalf("lowerBound(%d): expected %d, got %d",
				tt.timestamp, tt.expected, got)
		}
	}
}
