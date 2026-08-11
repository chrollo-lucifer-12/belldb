package db

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestDB(t *testing.T) *DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test.db")

	db := NewDB()

	if err := db.Open(path); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
	})

	return db
}

func TestPutGet(t *testing.T) {
	db := newTestDB(t)

	if err := db.Put("cpu", 100, 42.5); err != nil {
		t.Fatal(err)
	}

	got, err := db.Get("cpu", 100)
	if err != nil {
		t.Fatal(err)
	}

	if got != 42.5 {
		t.Fatalf("expected 42.5, got %v", got)
	}
}

func TestGetMissing(t *testing.T) {
	db := newTestDB(t)

	if err := db.Put("cpu", 100, 42.5); err != nil {
		t.Fatal(err)
	}

	_, err := db.Get("cpu", 200)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRange(t *testing.T) {
	db := newTestDB(t)

	if err := db.Put("cpu", 100, 10); err != nil {
		t.Fatal(err)
	}

	if err := db.Put("cpu", 200, 20); err != nil {
		t.Fatal(err)
	}

	if err := db.Put("cpu", 300, 30); err != nil {
		t.Fatal(err)
	}

	got := db.Range("cpu", 100, 300)

	if len(got) != 2 {
		t.Fatalf("expected 2 points, got %d", len(got))
	}

	if got[0].Value != 10 || got[1].Value != 20 {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestRangeNonExistingBounds(t *testing.T) {
	db := newTestDB(t)

	if err := db.Put("cpu", 100, 10); err != nil {
		t.Fatal(err)
	}

	if err := db.Put("cpu", 200, 20); err != nil {
		t.Fatal(err)
	}

	if err := db.Put("cpu", 300, 30); err != nil {
		t.Fatal(err)
	}

	got := db.Range("cpu", 150, 250)

	if len(got) != 1 {
		t.Fatalf("expected 1 point, got %d", len(got))
	}

	if got[0].Value != 20 {
		t.Fatalf("expected 20, got %v", got[0].Value)
	}
}

func TestMissingMetric(t *testing.T) {
	db := newTestDB(t)

	got := db.Range("cpu", 100, 200)

	if got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}

func TestRecoveryFromPartialRecord(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	// Write two records.
	db := NewDB()

	if err := db.Open(path); err != nil {
		t.Fatal(err)
	}

	if err := db.Put("cpu", 1000, 42); err != nil {
		t.Fatal(err)
	}

	if err := db.Put("cpu", 2000, 50); err != nil {
		t.Fatal(err)
	}

	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	// Simulate power loss by truncating the last record.
	fp, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		t.Fatal(err)
	}

	info, err := fp.Stat()
	if err != nil {
		fp.Close()
		t.Fatal(err)
	}

	if err := fp.Truncate(info.Size() - 5); err != nil {
		fp.Close()
		t.Fatal(err)
	}

	if err := fp.Close(); err != nil {
		t.Fatal(err)
	}

	// Recover.
	db = NewDB()

	if err := db.Open(path); err != nil {
		t.Fatal(err)
	}

	defer db.Close()

	// First record should survive.
	value, err := db.Get("cpu", 1000)
	if err != nil {
		t.Fatal(err)
	}

	if value != 42 {
		t.Fatalf("expected 42, got %v", value)
	}

	// Second record should have been discarded.
	_, err = db.Get("cpu", 2000)
	if err == nil {
		t.Fatal("expected second record to be lost")
	}
}
