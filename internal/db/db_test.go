package db

import (
	"testing"

	"github.com/belldb/internal/config"
)

func setupBenchmarkDB(b *testing.B) *DB {
	b.Helper()

	config.DATA_DIR = b.TempDir()
	config.LOG_DIR = b.TempDir()

	db := NewDB()

	if err := db.Open(); err != nil {
		b.Fatal(err)
	}

	b.Cleanup(func() {
		db.Close()
	})

	return db
}

func BenchmarkDBPut(b *testing.B) {
	db := setupBenchmarkDB(b)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := db.Put(
			"cpu_usage",
			int64(i),
			float64(i),
		); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDBGet(b *testing.B) {
	db := setupBenchmarkDB(b)

	for i := 0; i < 10000; i++ {
		if err := db.Put("cpu_usage", int64(i), float64(i)); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := db.Get("cpu_usage", int64(i%10000))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDBRange(b *testing.B) {
	db := setupBenchmarkDB(b)

	const points = 10000

	for i := 0; i < points; i++ {
		if err := db.Put("cpu_usage", int64(i), float64(i)); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		result := db.Range("cpu_usage", 2500, 7500)

		if len(result) != 5000 {
			b.Fatalf("expected 5000 points, got %d", len(result))
		}
	}

}
