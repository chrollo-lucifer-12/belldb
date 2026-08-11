package db

import "testing"

func TestPutGet(t *testing.T) {
	db := NewDB()

	db.Put("cpu", 100, 42.5)

	got, err := db.Get("cpu", 100)
	if err != nil {
		t.Fatal(err)
	}

	if got != 42.5 {
		t.Fatalf("expected 42.5, got %v", got)
	}
}

func TestGetMissing(t *testing.T) {
	db := NewDB()

	db.Put("cpu", 100, 42.5)

	_, err := db.Get("cpu", 200)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRange(t *testing.T) {
	db := NewDB()

	db.Put("cpu", 100, 10)
	db.Put("cpu", 200, 20)
	db.Put("cpu", 300, 30)

	got := db.Range("cpu", 100, 300)

	if len(got) != 2 {
		t.Fatalf("expected 2 points, got %d", len(got))
	}

	if got[0].Value != 10 || got[1].Value != 20 {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestRangeNonExistingBounds(t *testing.T) {
	db := NewDB()

	db.Put("cpu", 100, 10)
	db.Put("cpu", 200, 20)
	db.Put("cpu", 300, 30)

	got := db.Range("cpu", 150, 250)

	if len(got) != 1 {
		t.Fatalf("expected 1 point, got %d", len(got))
	}

	if got[0].Value != 20 {
		t.Fatalf("expected 20, got %v", got[0].Value)
	}
}

func TestMissingMetric(t *testing.T) {
	db := NewDB()

	got := db.Range("cpu", 100, 200)

	if got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}
