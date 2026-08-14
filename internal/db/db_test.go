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
